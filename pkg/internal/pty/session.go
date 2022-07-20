package pty

import (
	"dterm/base"
	"dterm/pkg/internal/ws"
	"encoding/json"
	"sync"

	"k8s.io/client-go/tools/remotecommand"
)

const (
	KExecMessageOpStdin       string = "0"
	KExecMessageOpResizeEvent string = "4"
)

type KExecSessionHandler struct {
	sizeChan     chan *remotecommand.TerminalSize
	clientSocket *ws.WSConn
	done         chan struct{}
	// buffer       *stream.StreamBuffer
	message       *KExecSessionMessage
	cmdParser     *StreamParser
	recorder      *Recorder
	startParser   sync.Once
	startRecorder sync.Once
}

type KExecSessionMessage struct {
	Op       string
	Raw      []byte
	TermSize remotecommand.TerminalSize
}

func (kesm *KExecSessionMessage) Parse(p []byte) error {
	kesm.Op = string(p)[:1]
	if kesm.Op == KExecMessageOpResizeEvent {
		if err := json.Unmarshal([]byte(string(p)[1:]), &kesm.TermSize); err == nil {
			return nil
		}
	}
	kesm.Op = KExecMessageOpStdin
	kesm.Raw = p[:]
	return nil
}

func NewKExecSessionHandler(wsConn *ws.WSConn, username, instance string) *KExecSessionHandler {
	return &KExecSessionHandler{
		sizeChan:     make(chan *remotecommand.TerminalSize),
		clientSocket: wsConn,
		done:         make(chan struct{}),
		// buffer:       stream.NewStreamBuffer(BufferCap),
		message:       &KExecSessionMessage{},
		cmdParser:     NewStreamParser(),
		recorder:      NewRecorder(username, instance),
		startParser:   sync.Once{},
		startRecorder: sync.Once{},
	}
}

// Next for tty resize event
func (kesh *KExecSessionHandler) Next() *remotecommand.TerminalSize {
	for {
		select {
		case s := <-kesh.sizeChan:
			if s == nil {
				continue
			} else {
				return s
			}
		case <-kesh.done:
			kesh.recorder.Done <- struct{}{}
			kesh.cmdParser.done <- struct{}{}
			return nil
		}
	}
}

func (kesh *KExecSessionHandler) Done() <-chan struct{} {
	return kesh.done
}

// Read for k8s exec session stdin, read from client socket input message (clientSocket -> buffer -> p)
func (kesh *KExecSessionHandler) Read(p []byte) (int, error) {
	var _p = make([]byte, 1024)
	if n, err := kesh.clientSocket.Read(_p); err != nil {
		base.Log.Errorf("failed to read from client socket: %s", err.Error())
		return 0, err
	} else {
		_p = _p[:n]
	}
	if err := kesh.message.Parse(_p); err != nil {
		base.Log.Errorf("failed to parse message read from client socket: %s", err.Error())
		return 0, err
	}
	if kesh.message.Op == KExecMessageOpResizeEvent {
		base.Log.Tracef("got term resize event: %v", kesh.message.TermSize)
		kesh.sizeChan <- &kesh.message.TermSize
		return 0, nil
	} else {
		// fmt.Printf("[input] - : %s\n\n", string(kesh.message.Raw))
		go kesh.cmdParser.rcvInRaw(kesh.message.Raw)
		return copy(p, kesh.message.Raw), nil
	}
}

// Write for k8s exec session stdout\stderr, write message to client socket (p -> buffer -> clientSocket)
func (kesh *KExecSessionHandler) Write(p []byte) (int, error) {
	kesh.startRecorder.Do(func() {
		go kesh.recorder.AutoFlushInBg()
	})
	kesh.startParser.Do(func() {
		go kesh.cmdParser.StartParse()
	})
	// fmt.Printf("%02x\n", p)
	// fmt.Printf("[output] - : %s\n\n", string(p))
	_p := p[:]
	go kesh.recorder.Write(_p)
	go kesh.cmdParser.rcvOutRaw(_p)
	return kesh.clientSocket.Write(p)
}

func (kesh *KExecSessionHandler) Close() error {
	defer func() {
		if r := recover(); r != nil {
			base.Log.Warnf("recoverd in *KExecSessionHandler.Close(): %v", r)
		}
	}()
	close(kesh.sizeChan)
	kesh.done <- struct{}{}
	base.Log.Infof("connection closed (user: %s - instance: %s)", kesh.recorder.Username, kesh.recorder.Instance)
	return kesh.clientSocket.Close()
}
