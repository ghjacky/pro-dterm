package pty

import (
	"bytes"
	"dterm/base"
	"fmt"
	"regexp"
	"strconv"
)

const (
	ModeInitial uint8 = iota
	ModeInputing
	ModeWaitingExec
	ModeVim
)

var (
	EnterKey     = []byte{0xd}
	BackspaceKey = []byte{0x08}
	EraseKey     = []byte{0x1b, 0x5b, 0x4a}
	EraseKey0    = []byte{0x1b, 0x5b, 0x30, 0x4a}
	LeftKey      = []byte{0x1b, 0x5b, 0x44}
	RightKey     = []byte{0x1b, 0x5b, 0x43}
	DownKey      = []byte{0x1b, 0x5b, 0x42}
	UpKey        = []byte{0x1b, 0x5b, 0x41}
	CancelKey    = []byte{0x3}
	NewLineKey   = []byte{0xd, 0xa}
	TabKey       = []byte{0x9}
	DelKey       = []byte{0x7F}

	EraseMarks = [][]byte{
		[]byte("\x1b[J"),
		[]byte("\x1b[0J"),
		[]byte("\x1b[1J"),
		[]byte("\x1b[2J"),
		[]byte("\x1b[3J"),
		[]byte("\x1b[K"),
		[]byte("\x1b[0K"),
		[]byte("\x1b[1K"),
		[]byte("\x1b[2K"),
	}
	VimEnterMarks = [][]byte{
		[]byte("\x1b[?1049h"),
		[]byte("\x1b[?1048h"),
		[]byte("\x1b[?1047h"),
		[]byte("\x1b[?47h"),
	}

	VimExitMarks = [][]byte{
		[]byte("\x1b[?1049l"),
		[]byte("\x1b[?1048l"),
		[]byte("\x1b[?1047l"),
		[]byte("\x1b[?47l"),
	}

	RequestPositionPattern   = regexp.MustCompile(`\x1b\[6n`)
	ResponsePositionPattern  = regexp.MustCompile(`\x1b\[(?P<pos>[0-9;]*)[RHf]`)
	ControlCharactersPattern = regexp.MustCompile(`\x1b\[[0-9;\?]*[RHflJKmMGn]`)
)

type cmd struct {
	command          []byte
	result           []byte
	currsorPositions [2]uint16
	prompt           []byte
}

type StreamParser struct {
	inChan  chan []byte
	outChan chan []byte
	mode    chan uint8
	done    chan struct{}
	cmd
}

func NewStreamParser() *StreamParser {
	mod := make(chan uint8, 1)
	mod <- ModeInitial
	return &StreamParser{
		inChan:  make(chan []byte),
		outChan: make(chan []byte),
		mode:    mod,
		done:    make(chan struct{}),
		cmd: cmd{
			command: make([]byte, 0),
			result:  make([]byte, 0),
		},
	}
}

func (sp *StreamParser) StartParse() {
	sp.lifecycle()
}

func (sp *StreamParser) rcvInRaw(p []byte) {
	sp.inChan <- p
}

func (sp *StreamParser) rcvOutRaw(p []byte) {
	sp.outChan <- p
}

func (sp *StreamParser) lifecycle() {
	for {
		select {
		case mod := <-sp.mode:
			switch mod {
			case ModeInitial:
				sp.handleInitial()
			case ModeInputing:
				sp.handleInputing()
			case ModeWaitingExec:
				sp.handleWaitingExec()
			case ModeVim:
				sp.handleVim()
			default:
				base.Log.Errorf("Unknown lifecycle: %s", sp.mode)
			}
		case <-sp.done:
			return
		}
	}
}

func (sp *StreamParser) handleInitial() {
	out := <-sp.outChan
	if isRequestForCursorPosition(out) {
		sp.setPrompt(escapeForPrompt(out))
		sp.initCursorPosition(sp.waitForCursorPosition().Column)
	} else {
		base.Log.Errorf("Unkown control characters from output flow: %s (stream parser exiting...)", out)
		sp.done <- struct{}{}
		return
	}
	out = <-sp.outChan
	sp.initCommand()
	sp.resetResult()
	sp.setMode(ModeInputing)
}

func (sp *StreamParser) handleInputing() {
	for {
		in := <-sp.inChan
		if bytes.Compare(in, EnterKey) == 0 {
			sp.setCurrentCursorPosition(uint16(len(sp.cmd.command)))
			sp.setMode(ModeWaitingExec)
			break
		} else if bytes.Compare(in, DelKey) == 0 {
			sp.delCharInCommand()
		} else if bytes.Compare(in, TabKey) == 0 {
			out := escapeControlCharacters(<-sp.outChan)
			sp.commandAutoCompletion(out)
		} else if bytes.Compare(in, LeftKey) == 0 {
			if sp.currentCursorPosition() > uint16(len(sp.cmd.prompt)) {
				sp.setCurrentCursorPosition(sp.currentCursorPosition() - 1)
			}
		} else if bytes.Compare(in, RightKey) == 0 {
			if sp.currentCursorPosition() < uint16(len(sp.cmd.command)) {
				sp.setCurrentCursorPosition(sp.currentCursorPosition() + 1)
			}
		} else if bytes.Compare(in, UpKey) == 0 {

		} else if bytes.Compare(in, DownKey) == 0 {

		} else if bytes.Compare(in, CancelKey) == 0 {
			out := <-sp.outChan
			sp.setPrompt(escapeForPrompt(out))
			sp.initCommand()
			sp.initCursorPosition(sp.waitForCursorPosition().Column)
		} else {
			out := <-sp.outChan
			// some input control key checkin based on out flow (like, remove characters hot key)
			if isEraseMarks(out) {
				if bytes.HasPrefix(out, EnterKey) && (bytes.HasSuffix(out, EraseKey) || bytes.HasSuffix(out, EraseKey0)) {
					sp.setCommand(sp.cmd.prompt)
					sp.initCursorPosition(uint16(len(sp.cmd.prompt)))
				} else {
					eraseTag := append(BackspaceKey, EraseKey...)
					length := bytes.Count(out, eraseTag)
					for i := 0; i < length; i++ {
						sp.delCharInCommand()
					}
				}
			} else {
				sp.pushCommandCharacters(in)
			}
		}
	}
}

func (sp *StreamParser) handleWaitingExec() {
	for {
		out := <-sp.outChan
		if isRequestForCursorPosition(out) {
			sp.calcCommand()
			sp.appendResult(escapeControlCharacters(out))
			sp.initCursorPosition(sp.waitForCursorPosition().Column)
			sp.calcResult()
			sp.resetCommand()
			sp.resetResult()
			sp.initCommand()
			sp.setMode(ModeInputing)
			break
		} else if isPrintableCharacter(out) {
			sp.appendResult(out)
			continue
		} else if enterVim(out) || enterTopLikeMode(out) {
			sp.calcCommand()
			sp.resetCommand()
			sp.resetResult()
			sp.initCommand()
			sp.setMode(ModeVim)
			break
		} else {
			continue
		}
	}
}

func (sp *StreamParser) handleVim() {
	for {
		select {
		case <-sp.inChan:
		case out := <-sp.outChan:
			if exitVim(out) {
				sp.resetResult()
				sp.initCommand()
				sp.initCursorPosition(sp.waitForCursorPosition().Column)
				sp.setMode(ModeInputing)
				return
			}
		}
	}
}

type cursorPosition struct {
	Line   uint16
	Column uint16
}

func (sp *StreamParser) waitForCursorPosition() cursorPosition {
	var cp = cursorPosition{}
	var err error
	for {
		in := <-sp.inChan
		if isResponseForCursorPosition(in) {
			if cp, err = parseCursorPosition(in); err != nil {
				// base.Log.Errorf("Parse cursor position failed from input flow: %s (stream parser exiting...)", in)
				continue
			} else {
				return cp
			}
		} else {
			// base.Log.Errorf("Unkown control characters from input flow: %s (stream parser exiting...)", in)
			continue
		}
	}
}

func (sp *StreamParser) setPrompt(p []byte) {
	sp.cmd.prompt = p
}

func (sp *StreamParser) commandAutoCompletion(p []byte) {
	if uint16(len(p)) <= sp.lastCursorPosition() {
		return
	}
	if bytes.Contains(p, NewLineKey) {
		_ps := bytes.Split(p, NewLineKey)
		p = _ps[len(_ps)-1]
	}
	p = bytes.TrimLeft(p, string(EnterKey))
	sp.cmd.command = append(sp.cmd.command[:sp.lastCursorPosition()], p[sp.lastCursorPosition():]...)
	sp.setCurrentCursorPosition(uint16(len(sp.cmd.command)))
}

func (sp *StreamParser) initCursorPosition(c uint16) {
	sp.cmd.currsorPositions[0] = c - 1
	sp.cmd.currsorPositions[1] = c - 1
}

func (sp *StreamParser) setCurrentCursorPosition(c uint16) {
	sp.cmd.currsorPositions[1] = c
}

func (sp *StreamParser) initCommand() {
	sp.cmd.command = sp.cmd.prompt
	sp.initCursorPosition(uint16(len(sp.cmd.prompt)) + 1)
}
func (sp *StreamParser) setCommand(p []byte) {
	sp.cmd.command = p
}
func (sp *StreamParser) resetCommand() {
	sp.cmd.command = []byte{}
}
func (sp *StreamParser) delCharInCommand() {
	if sp.currentCursorPosition() <= sp.lastCursorPosition() {
		return
	}
	if sp.currentCursorPosition() >= uint16(len(sp.cmd.command)) {
		sp.setCurrentCursorPosition(uint16(len(sp.cmd.command)) - 1)
	}
	sp.setCommand(append(sp.cmd.command[:sp.currentCursorPosition()-1], sp.cmd.command[sp.currentCursorPosition():]...))
	sp.setCurrentCursorPosition(sp.currentCursorPosition() - 1)
}
func (sp *StreamParser) calcCommand() {
	defer sp.initCommand()
	if sp.currentCursorPosition() > sp.lastCursorPosition() {
		sp.cmd.command = sp.cmd.command[sp.lastCursorPosition():sp.currentCursorPosition()]
		fmt.Printf("[Command] - %s\n\n", sp.cmd.command)
	}
}
func (sp *StreamParser) setMode(mode uint8) {
	sp.mode <- mode
}
func (sp *StreamParser) lastCursorPosition() uint16 {
	return sp.cmd.currsorPositions[0]
}
func (sp *StreamParser) currentCursorPosition() uint16 {
	return sp.cmd.currsorPositions[1]
}

func (sp *StreamParser) pushCommandCharacters(p []byte) {
	if sp.currentCursorPosition() >= uint16(len(sp.cmd.command)) {
		sp.cmd.command = append(sp.cmd.command, p...)
	} else {
		before := sp.cmd.command[:sp.currentCursorPosition()-1]
		behind := sp.cmd.command[sp.currentCursorPosition()-1:]
		sp.cmd.command = append(append(before, p...), behind...)
	}
	sp.setCurrentCursorPosition(sp.currentCursorPosition() + uint16(len(p)))
}
func (sp *StreamParser) resetResult() {
	sp.cmd.result = []byte{}
}
func (sp *StreamParser) appendResult(p []byte) {
	sp.cmd.result = append(sp.cmd.result, p...)
}
func (sp *StreamParser) calcResult() {
	defer sp.resetResult()
	ls := bytes.Split(sp.cmd.result, NewLineKey)
	if len(ls) >= 2 {
		sp.setPrompt(ls[len(ls)-1])
	} else {
		return
	}
	r := bytes.TrimLeft(bytes.Join(ls[:len(ls)-1], NewLineKey), string(NewLineKey))
	if len(r) > 0 {
		fmt.Printf("[Result] - %s\n\n", r)
	}
}

func isPrintableCharacter(p []byte) bool {
	return len(p) == 1 && p[0] < 127 && p[0] > 32
}

func isRequestForCursorPosition(p []byte) bool {
	return RequestPositionPattern.Match(p)
}
func isResponseForCursorPosition(p []byte) bool {
	return ResponsePositionPattern.Match(p)
}

func parseCursorPosition(p []byte) (cursorPosition, error) {
	var cp = cursorPosition{}
	r := ResponsePositionPattern.FindSubmatch(p)
	if len(r) != 2 {
		return cp, fmt.Errorf("wrong cursor position control characters: %x", p)
	} else {
		lcs := bytes.Split(r[1], []byte(";"))
		if len(lcs) != 2 {
			return cp, fmt.Errorf("wrong cursor position control characters: %x", p)
		} else {
			l, e1 := strconv.Atoi(string(lcs[0]))
			c, e2 := strconv.Atoi(string(lcs[1]))
			if e1 != nil || e2 != nil {
				return cp, fmt.Errorf("wrong cursor position control characters: %x", p)
			} else {
				cp.Line = uint16(l)
				cp.Column = uint16(c)
				return cp, nil
			}
		}
	}
}

func escapeControlCharacters(p []byte) []byte {
	return ControlCharactersPattern.ReplaceAll(p, []byte{})
}

func escapeForPrompt(p []byte) []byte {
	patt0 := regexp.MustCompile(`.*\r\n`)
	patt1 := regexp.MustCompile(`.*\r`)
	return patt1.ReplaceAll(patt0.ReplaceAll(escapeControlCharacters(p), []byte{}), []byte{})
}

func isEraseMarks(p []byte) bool {
	for _, v := range EraseMarks {
		if bytes.Contains(p, v) {
			return true
		}
	}
	return false
}

func enterVim(p []byte) bool {
	for _, v := range VimEnterMarks {
		if bytes.Contains(p, v) {
			return true
		}
	}
	return false
}

func exitVim(p []byte) bool {
	for _, v := range VimExitMarks {
		if bytes.Contains(p, v) {
			return true
		}
	}
	return false
}

func enterTopLikeMode(p []byte) bool {
	fsb := []byte(`\x1b[H\x1b[J`)
	patt0 := regexp.MustCompile(`.*\r\n`)
	patt1 := regexp.MustCompile(`.*\r`)
	return bytes.HasPrefix(patt1.ReplaceAll(patt0.ReplaceAll(p, []byte{}), []byte{}), fsb)
}
