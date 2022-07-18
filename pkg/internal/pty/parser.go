package pty

import (
	"bytes"
	"sync"
)

const (
	StatusInitial uint8 = iota
	StatusInputing
	StatusWaitingExec
	StatusVimMode
	StatusCancel
	StatusHistory
	StatusAutoCompletion
)

var (
	EnterKey     = []byte{0xd}
	BackspaceKey = []byte{0x7f}
	LeftKey      = []byte{0x1b, 0x5b, 0x44}
	RightKey     = []byte{0x1b, 0x5b, 0x43}
	DownKey      = []byte{0x1b, 0x5b, 0x42}
	UpKey        = []byte{0x1b, 0x5b, 0x41}
	CancelKey    = []byte{0x3}
	NewLineKey   = []byte{0xd, 0xa}
	TabKey       = []byte{0x9}
	DelKey       = []byte{0x7F}

	EraseMarks   = [][]byte{
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

	RequestPosition         = []byte("\x1b[6n")
	ResponsePositionPattern = `\x1b\[(?P<pos>[0-9;]*)[RHf]`
)

type cmd struct {
	command  []byte
	result   []byte
	prompt   []byte
	position uint16
}

type StreamParser struct {
	mu       sync.Mutex
	inChan   chan []byte
	outChan  chan []byte
	waitChan chan struct{}
	status   uint8
	cmd
}

func NewStreamParser() *StreamParser {
	return &StreamParser{
		mu:       sync.Mutex{},
		inChan:   make(chan []byte),
		outChan:  make(chan []byte),
		waitChan: make(chan struct{}),
		cmd: cmd{
			command: make([]byte, 0),
			result:  make([]byte, 0),
		},
	}
}

func (sp *StreamParser) StartParse() {
	go sp.parseIn()
	go sp.parseOut()
}

func (sp *StreamParser) parseIn() {
	for {
		sp.mu.Lock()
		select {
		case in :=<- sp.inChan:
			if bytes.Compare(in, EnterKey) == 0 {

			} else if bytes.Compare(in, TabKey) == 0 {

			} else if bytes.Compare(in, DelKey) == 0 {

			} else if bytes.Compare(in, CancelKey) == 0 {}
		}
	}
}

func (sp *StreamParser) parseOut() {

}
