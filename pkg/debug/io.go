package debug

import "dterm/base"

type DRWCloser struct {
	Msg string
	IO  chan []byte
}

func NewDRWCloser(msg string, cap int) DRWCloser {
	return DRWCloser{
		Msg: msg,
		IO:  make(chan []byte, cap),
	}
}

func (rw DRWCloser) Read(p []byte) (int, error) {
	b := <-rw.IO
	for i, v := range b {
		if i < cap(p) {
			p[i] = v
		} else {
			break
		}
	}
	base.Log.Tracef("Trace Read() - %s", string(p))
	return len(p), nil
}

func (rw DRWCloser) Write(p []byte) (int, error) {
	base.Log.Tracef("Trace Write() - %s %s", rw.Msg, string(p))
	return len(p), nil
}

func (rw DRWCloser) Close() error {
	close(rw.IO)
	return nil
}
