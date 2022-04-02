package kk

import (
	"context"
	"dterm/base"
	"dterm/pkg/internal/pty.go"
	"dterm/pkg/internal/ws"
	"errors"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/gorilla/websocket"
)

func StreamContainerShell(conn *websocket.Conn, name string, dproxy string) error {

	var wsc = ws.NewWSConn(conn, websocket.TextMessage)
	defer wsc.Close()
	var dc = NewDContainer(dproxy)
	if err := dc.GetByIp(name); err != nil {
		base.Log.Errorf("failed to get container by name(%s): %s", name, err.Error())
		return err
	}
	if len(dc.Containers) < 1 {
		return errors.New("no avaliable containers")
	}
	var exechandler = pty.NewKExecSessionHandler(wsc)
	defer exechandler.Close()
	if err := dc.streamExec(dc.Containers[0].ID, exechandler); err != nil {
		base.Log.Errorf("failed to get exec stream: %s", err.Error())
		return err
	}
	return nil
}

func (c *DContainer) streamExec(container string, session pty.PTY) error {
	id, err := c.DC.Client.ContainerExecCreate(context.Background(), container, types.ExecConfig{
		User:         "root",
		Privileged:   true,
		Tty:          true,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Detach:       true,
		DetachKeys:   "ctrl-p,ctrl-q",
		Cmd:          []string{"/bin/sh", "-i"},
		WorkingDir:   "/tmp",
	})
	if err != nil {
		base.Log.Errorf("failed to bind shell to container(%s): %s", container, err.Error())
		return err
	}

	att, err := c.DC.Client.ContainerExecAttach(context.Background(), id.ID, types.ExecStartCheck{Detach: false, Tty: true})
	if err != nil {
		base.Log.Errorf("failed to attach exec with id(%s): %s", id.ID, err.Error())
		return err
	}
	defer att.Close()

	var errChan = make(chan error, 2)
	go func() {
		_, err := io.Copy(att.Conn, session)
		errChan <- err
	}()
	go func() {
		_, err := io.Copy(session, att.Conn)
		errChan <- err
	}()
	go func() {
		for {
			resize := session.Next()
			if resize == nil {
				break
			}
			c.resizeTty(id.ID, resize.Width, resize.Height)
		}
	}()
	return <-errChan
}

func (c *DContainer) resizeTty(id string, width, height uint16) error {
	return c.DC.Client.ContainerExecResize(context.Background(), id, types.ResizeOptions{Width: uint(width), Height: uint(height)})
}
