package pty

import (
	"dterm/base"
	"dterm/model"
	"regexp"
	"time"
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
	ControlCharactersPattern = regexp.MustCompile(`\x1b\[[0-9;\?]*[RHflJKmMGnhC@]`)
)

type cmd struct {
	command          []byte
	result           []byte
	currsorPositions [2]uint16
	cursor           int
	prompt           []byte
	done             chan struct{}
}

type StreamParser struct {
	inChan    chan []byte
	doneIn    chan struct{}
	outChan   chan []byte
	doneOut   chan struct{}
	allChan   chan []byte
	shell     string
	roundDone chan struct{}
	stage     uint8
	mode      chan uint8
	done      chan struct{}
	cmdChan   chan model.MCommand
	username  string
	instance  string
	filepath  string
	cmd
}

func NewStreamParser(username, instance, filepath string) *StreamParser {
	mod := make(chan uint8, 1)
	mod <- ModeInitial
	return &StreamParser{
		inChan:   make(chan []byte),
		doneIn:   make(chan struct{}, 1),
		outChan:  make(chan []byte),
		doneOut:  make(chan struct{}, 1),
		allChan:  make(chan []byte),
		mode:     mod,
		stage:    StageInitial,
		done:     make(chan struct{}),
		cmdChan:  make(chan model.MCommand, 10),
		username: username,
		instance: instance,
		filepath: filepath,
		cmd: cmd{
			command:          make([]byte, 0),
			result:           make([]byte, 0),
			currsorPositions: [2]uint16{0, 0},
			cursor:           0,
			done:             make(chan struct{}, 1),
		},
	}
}

func (sp *StreamParser) StartParse() {
	sp.StartBashParser()
	// out := bytes.TrimRight(<-sp.outChan, string(EndLineKey))
	// if sp.isBash(out) {
	// 	sp.StartBashParser()
	// } else if isRequestForCursorPosition(out) {
	// 	sp.setPrompt(escapeForPrompt(out))
	// 	sp.lifecycleForSh()
	// } else {
	// 	base.Log.Errorf("unknown initial control characters: %s (cmd stream parser exiting ...)", out)
	// 	return
	// }
}

func (sp *StreamParser) rcvInRaw(p []byte) {
	select {
	case <-time.After(10 * time.Second):
		return
	case sp.inChan <- p:
	}
}

func (sp *StreamParser) rcvOutRaw(p []byte) {
	select {
	case <-time.After(10 * time.Second):
		return
	case sp.outChan <- p:
	}
}

func (sp *StreamParser) StartRecordCmdInBg() {
	for {
		select {
		case mcmd := <-sp.cmdChan:
			if err := mcmd.Add(); err != nil {
				base.Log.Errorf("failed to record command to db")
			}
		case <-sp.cmd.done:
			return
		}
	}
}

func (sp *StreamParser) isBash(p []byte) bool {
	return sp.isRequestForPrompt(p)
}
