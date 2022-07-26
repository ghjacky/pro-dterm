package pty

import (
	"bytes"
	"dterm/base"
	"dterm/model"
	"fmt"
	"regexp"
	"strconv"
	"time"
)

func (sp *StreamParser) lifecycleForSh() {
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
				sp.cmd.done <- struct{}{}
			}
		case <-sp.done:
			sp.cmd.done <- struct{}{}
			return
		}
	}
}

func (sp *StreamParser) handleInitial() {
	sp.initCursorPosition(sp.waitForCursorPosition().Column)
	_ = <-sp.outChan
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
			sp.appendResult(escapeControlCharacters(out))
			sp.calcCmd()
			sp.initCursorPosition(sp.waitForCursorPosition().Column)
			sp.resetCommand()
			sp.resetResult()
			sp.initCommand()
			sp.setMode(ModeInputing)
			break
		} else if isPrintableCharacter(out) {
			sp.appendResult(out)
			continue
		} else if enterVim(out) || enterTopLikeMode(out) {
			sp.resetResult()
			sp.calcCmd()
			sp.resetCommand()
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

func (sp *StreamParser) calcCmd() {
	defer sp.initCommand()
	defer sp.resetResult()
	c := model.MCommand{
		Username: sp.username,
		Instance: sp.instance,
	}
	if sp.currentCursorPosition() <= sp.lastCursorPosition() {
		return
	}
	sp.cmd.command = sp.cmd.command[sp.lastCursorPosition():sp.currentCursorPosition()]
	ls := bytes.Split(sp.cmd.result, NewLineKey)
	if len(ls) >= 2 {
		sp.setPrompt(ls[len(ls)-1])
	} else {
		return
	}
	r := bytes.TrimLeft(bytes.Join(ls[:len(ls)-1], NewLineKey), string(NewLineKey))
	c.Command = string(sp.cmd.command)
	c.Result = string(r)
	c.At = time.Now().Local().UnixNano()
	c.TX = base.DB()
	c.RecordFile = sp.filepath
	sp.cmdChan <- c
	fmt.Println("command: ", c.Command)
	fmt.Println("result: ", c.Result)
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
	sp.cmd.prompt = p[:]
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
	sp.cmd.command = sp.cmd.prompt[:]
	sp.initCursorPosition(uint16(len(sp.cmd.prompt)) + 1)
}
func (sp *StreamParser) setCommand(p []byte) {
	sp.cmd.command = p[:]
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
