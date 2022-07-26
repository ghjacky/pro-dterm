package play

import (
	"bufio"
	"dterm/base"
	"dterm/model"
	"encoding/json"
	"os"
	"path"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	keyCtrlC = 3
	keyCtrlD = 4
	keySpace = 32
	keyLeft  = 68
	keyRight = 67
	keyUp    = 65
	keyDown  = 66
)

type EvItem struct {
	Data []byte `json:"data"`
	At   int64  `json:"at"`
}

func Play(cid uint, conn *websocket.Conn) error {
	mcmd := model.MCommand{}
	mcmd.TX = base.DB()
	mcmd.ID = cid
	if err := mcmd.Get(); err != nil {
		base.Log.Errorf("failed to get command by id: %s", err.Error)
		return err
	}
	filepath := mcmd.RecordFile
	wg := sync.WaitGroup{}
	wg.Add(1)
	var donec = make(chan bool)
	filepath = path.Join(base.Conf.MainConfiguration.DataDir, filepath)
	f, err := os.OpenFile(filepath, os.O_RDONLY, 0644)
	if err != nil {
		base.Log.Errorf("couldn't open file %s", filepath)
		return err
	}
	defer func() {
		f.Close()
	}()
	var allRecoredBytesData = []EvItem{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		recordItem := EvItem{}
		line := scanner.Text()
		lineBytes := []byte(line)
		if err := json.Unmarshal(lineBytes, &recordItem); err != nil {
			base.Log.Fatalf("couldn't parse line bytes: %s", err.Error())
			return err
		}
		if (mcmd.At - recordItem.At) <= 3*1000*1000*1000 {
			allRecoredBytesData = append(allRecoredBytesData, recordItem)
		}
	}

	os.Stdout.Write([]byte("\x1bc"))
	var signal = make(chan int, 1)
	go func(sc <-chan int) {
		once := sync.Once{}
		defer wg.Done()
		//log.Printf("start scan file")
		sleeping := 0 * time.Nanosecond
		lastTime := int64(0)
		var sig = make(chan int, 0)
		for _, recordItem := range allRecoredBytesData {
			select {
			case s := <-sc:
				switch s {
				case 1:
					continue
				case 2:
					wait(donec)
				default:
					break
				}
			default:
				currentTime := recordItem.At
				once.Do(func() {
					lastTime = recordItem.At
				})
				sleeping = time.Duration(currentTime-lastTime) * time.Nanosecond
				go func() {
					time.Sleep(sleeping / 1)
					<-sig
				}()
				//if strings.Contains(string(recordItem.Data), "bash-3.2$ "){
				//	time.Sleep(sleeping / 10 / 1)
				//}else {
				//	time.Sleep(sleeping / 1)
				//}
				sig <- 1
				err := conn.WriteMessage(websocket.BinaryMessage, recordItem.Data)
				lastTime = recordItem.At
				if err != nil {
					base.Log.Errorf("failed to play: %s", err.Error())
					return
				}
			}
		}
	}(signal)

	go func() {
		signal <- 1
		keySpaceCounter := 0
		for {
			_, key, err := conn.ReadMessage()
			if err != nil {
				return
			}
			switch key[0] {
			case keyCtrlC:
				signal <- -1
			case keySpace:
				keySpaceCounter += 1
				if keySpaceCounter%2 == 0 {
					donec <- true
					signal <- 1
				} else {
					//log.Printf("Pausing ... ")
					signal <- 2
				}
			default:
				continue
			}
		}
	}()
	wg.Wait()
	return nil
}

func wait(c chan bool) {
	<-c
}
