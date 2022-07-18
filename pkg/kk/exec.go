package kk

import (
	"context"
	"dterm/base"
	"dterm/pkg/internal/pty"
	"dterm/pkg/internal/ws"
	"errors"
	"io"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/gorilla/websocket"
)

func StreamContainerShell(conn *websocket.Conn, name string, dproxy string, username string) error {

	var wsc = ws.NewWSConn(conn, websocket.BinaryMessage)
	defer wsc.Close()
	var dc = NewDContainer(dproxy)
	defer func() {
		if dc.Client != nil {
			dc.Client.Close()
		}
	}()
	if err := dc.GetByName(name); err != nil {
		base.Log.Errorf("failed to get container by name(%s): %s", name, err.Error())
		return err
	}
	if len(dc.Containers) < 1 {
		return errors.New("no avaliable containers")
	}
	var exechandler = pty.NewKExecSessionHandler(wsc, username, name)
	defer func() {
		exechandler.Write([]byte("Connection closed !"))
		exechandler.Close()
	}()
	if err := dc.streamExec(dc.Containers[0].ID, exechandler); err != nil {
		base.Log.Errorf("failed to get exec stream: %s", err.Error())
		return err
	}
	return nil
}

func (c *DContainer) streamExec(container string, session pty.PTY) error {
	id, err := c.DC.Client.ContainerExecCreate(context.Background(), container, types.ExecConfig{
		User:         "root",
		Privileged:   false,
		Tty:          true,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Detach:       false,
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
	defer func() {
		// EXIT EXEC SHELL
		att.Conn.Write([]byte{'\003'})
		time.Sleep(100 * time.Millisecond)
		att.Conn.Write([]byte{13, 10})
		time.Sleep(100 * time.Millisecond)
		att.Conn.Write([]byte{'\004'})
		att.Close()
	}()

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
