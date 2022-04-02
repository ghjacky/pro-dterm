package stream

import "dterm/base"

type StreamBuffer struct {
	// MsgChan chan StreamMessage
	C chan []byte
}

func NewStreamBuffer(cap int) *StreamBuffer {
	var sb = &StreamBuffer{}
	// sb.MsgChan = make(chan StreamMessage)
	sb.C = make(chan []byte, cap)
	return sb
}

func (sb *StreamBuffer) Write(p []byte) (int, error) {
	sb.C <- p
	return len(p), nil
}

func (sb *StreamBuffer) Close() error {
	defer func() {
		if r := recover(); r != nil {
			base.Log.Warnf("recovered from *StreamBuffer.Close(): %v", r)
		}
	}()
	close(sb.C)
	return nil
}

func (sb *StreamBuffer) Read(p []byte) (int, error) {
	b := <-sb.C
	return copy(p, b), nil
}
