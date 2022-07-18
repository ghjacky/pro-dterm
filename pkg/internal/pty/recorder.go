package pty

import (
	"dterm/base"
	"dterm/model"
	"encoding/json"
	"errors"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

type EventItem struct {
	Data []byte `json:"data"`
	At   int64  `json:"at"`
}

type Recorder struct {
	Done     chan struct{}
	Buffer   chan EventItem
	Username string
	Instance string
	Filepath string
}

func NewRecorder(username, instance string) *Recorder {
	now := time.Now().Local()
	filepath := path.Join(now.Format("2006-01-02"), uuid.New().String())
	ps := strings.Split(filepath, "/")
	bp := base.Conf.MainConfiguration.DataDir
	dir := path.Join(bp, strings.Join(ps[:len(ps)-1], "/"))
	if err := os.MkdirAll(dir, 0755); err != nil {
		base.Log.Errorf("failed to create dir (%s) : %s", dir, err.Error())
		return nil
	}
	// 入库
	var evRcd = model.MRecord{Username: username, Instance: instance, Filepath: filepath}
	evRcd.TX = base.DB()
	if err := evRcd.Add(); err != nil {
		base.Log.Errorf("failed to save recorder file path to db!")
	}
	return &Recorder{
		Done:     make(chan struct{}),
		Buffer:   make(chan EventItem),
		Username: username,
		Instance: instance,
		Filepath: filepath,
	}
}

func (rcd *Recorder) Write(p []byte) (int, error) {
	var evItem = EventItem{
		Data: regexp.MustCompile(`\x1b\[[0-9;]*[RHfmGn]`).ReplaceAll(p, []byte{}),
		At:   time.Now().Local().UnixNano(),
	}
	select {
	case rcd.Buffer <- evItem:
		return len(p), nil
	case <-time.After(3 * time.Second):
		return 0, errors.New("timeout to write to event buffer")
	}
}

func (rcd *Recorder) Flush() {
	file := path.Join(base.Conf.MainConfiguration.DataDir, rcd.Filepath)
	f, e := os.OpenFile(file, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if e != nil {
		base.Log.Errorf("Failed to open record file: %s", e.Error())
		return
	}
	defer f.Close()
	for v := range rcd.Buffer {
		b, _ := json.Marshal(v)
		f.Write(b)
		f.Write([]byte("\r\n"))
	}
}

func (rcd *Recorder) AutoFlushInBg() {
	for {
		select {
		case <-time.After(10):
			rcd.Flush()
		case <-rcd.Done:
			rcd.Flush()
			base.Log.Infof("recorder done (user: %s - instance: %s) !", rcd.Username, rcd.Instance)
			break
		}
	}
}
