package kk

import (
	"context"
	"dterm/base"
	"dterm/pkg/internal/pty"
	"dterm/pkg/internal/ws"
	"io"

	"github.com/gorilla/websocket"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/remotecommand"
)

type KPod struct {
	*KC
	Resource v1.PodInterface
	Manifest corev1.Pod
}

func StreamPodLog(conn *websocket.Conn, namespace, name, container string) error {
	var wss = ws.NewWSStreamBuffer(1024)
	defer wss.Close()
	var wsc = ws.NewWSConn(conn, websocket.TextMessage)
	defer wsc.Close()

	pod, err := Kc.GetPod(namespace, name)
	if err != nil {
		base.Log.Errorf("failed to get pod: %s", err.Error())
		return err
	}
	plogstream, err := pod.streamLog(container)
	if err != nil {
		base.Log.Errorf("nil pod log stream: %s", err.Error())
		return err
	}
	defer plogstream.Close()

	var errChan = make(chan error, 2)
	go func() {
		_, err := io.Copy(wss, plogstream)
		errChan <- err
	}()
	go func() {
		_, err := io.Copy(wsc, wss)
		errChan <- err
	}()
	return <-errChan
}

func (p *KPod) streamLog(container string) (io.ReadCloser, error) {
	sinceSeconds := int64(10)
	req := p.Resource.GetLogs(p.Manifest.Name, &corev1.PodLogOptions{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		Container:                    container,
		InsecureSkipTLSVerifyBackend: true,
		Follow:                       true,
		SinceSeconds:                 &sinceSeconds,
	})
	if stream, err := req.Stream(context.Background()); err != nil {
		base.Log.Errorf("failed to streaming pod log: %s", err.Error())
		return nil, err
	} else {
		return stream, nil
	}
}

func StreamPodShell(conn *websocket.Conn, namespace, name, username, container string) error {

	var wsc = ws.NewWSConn(conn, websocket.TextMessage)
	defer wsc.Close()

	base.Log.Debugf("podExec() - before kk.Kc.GetPod")
	pod, err := Kc.GetPod(namespace, name)
	if err != nil {
		base.Log.Errorf("failed to get pod: %s", err.Error())
		return err
	}
	var exechandler = pty.NewKExecSessionHandler(wsc, username, container)
	defer func() {
		exechandler.Write([]byte("Connection closed !"))
		exechandler.Close()
	}()
	if err := pod.streamExec(container, exechandler); err != nil {
		base.Log.Errorf("failed to get exec stream: %s", err.Error())
		return err
	}
	return nil
}

func (p *KPod) streamExec(container string, session pty.PTY) error {
	req := p.KC.Client.CoreV1().RESTClient().Post().Resource("Pods").Namespace(p.Manifest.Namespace).Name(p.Manifest.Name).SubResource("exec")
	req.VersionedParams(&corev1.PodExecOptions{
		Container: container,
		Command:   []string{"/bin/sh", "-i"},
		Stdin:     true,
		Stdout:    true,
		Stderr:    true,
		TTY:       true,
	}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(p.RestConfig, "POST", req.URL())
	if err != nil {
		base.Log.Errorf("failed to new SPDY executor: %s", err.Error())
		return err
	}

	return executor.Stream(remotecommand.StreamOptions{
		Stdin:             session,
		Stdout:            session,
		Stderr:            session,
		Tty:               true,
		TerminalSizeQueue: session,
	})
}
