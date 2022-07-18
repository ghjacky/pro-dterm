package pty

import (
	"io"
	"k8s.io/client-go/tools/remotecommand"
)

const (
	BufferCap = 1024
)

type PTY interface {
	io.ReadWriteCloser
	remotecommand.TerminalSizeQueue
	Done() <-chan struct{}
}

type PTYMessage interface {
	Parse([]byte) error
}
