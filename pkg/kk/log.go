package kk

import (
	"context"
	"dterm/base"
	"dterm/pkg/internal/ws"
	"errors"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/gorilla/websocket"
)

func StreamContainerLog(conn *websocket.Conn, name string, dproxy string) error {
	var wsc = ws.NewWSConn(conn, websocket.TextMessage)
	defer wsc.Close()
	var dc = NewDContainer(dproxy)
	if err := dc.GetByName(name); err != nil {
		base.Log.Errorf("failed to get container by name(%s): %s", name, err.Error())
		return err
	}
	if len(dc.Containers) < 1 {
		return errors.New("no avaliable containers")
	}
	if err := dc.streamLog(dc.Containers[0].ID, wsc); err != nil {
		base.Log.Errorf("failed to stream container log: %s", err.Error())
		return err
	}
	return nil
}

func (c *DContainer) streamLog(container string, w io.WriteCloser) error {
	s, err := c.DC.Client.ContainerLogs(context.Background(), container, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Tail:       "10",
		Details:    true,
	})
	if err != nil {
		base.Log.Errorf("failed to get container(%s) logs: %s", container, err.Error())
		return err
	}
	defer s.Close()
	var errChan = make(chan error)
	go func() {
		_, err := io.Copy(w, s)
		errChan <- err
	}()
	return <-errChan
}
