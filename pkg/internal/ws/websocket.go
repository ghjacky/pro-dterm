package ws

import (
	"dterm/base"
	"dterm/pkg/internal/stream"
	"io"

	"github.com/gorilla/websocket"
)

func NewWSStreamBuffer(cap int) io.ReadWriteCloser {
	return stream.NewStreamBuffer(cap)
}

type WSConn struct {
	MessageType int
	*websocket.Conn
}

func NewWSConn(conn *websocket.Conn, msgType int) *WSConn {
	return &WSConn{
		MessageType: msgType,
		Conn:        conn,
	}
}

func (wsc *WSConn) Write(p []byte) (int, error) {
	wsc.EnableWriteCompression(true)
	err := wsc.WriteMessage(wsc.MessageType, p)
	if err != nil {
		return 0, err
	} else {
		return len(p), nil
	}
}

func (wsc *WSConn) Close() error {
	defer func() {
		if r := recover(); r != nil {
			base.Log.Warnf("recovered from *WSConn.Close(): %v", r)
		}
	}()
	return wsc.Conn.Close()
}

func (wsc *WSConn) Read(p []byte) (int, error) {
	_, _p, err := wsc.ReadMessage()
	if err != nil {
		return 0, err
	} else {
		return copy(p, _p), nil
	}
}
