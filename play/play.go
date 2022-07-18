package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"os"
	"sync"
	"time"
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
	At int64 `json:"at"`
}

func Play(filepath string) {
	wg := sync.WaitGroup{}
	wg.Add(1)
	var donec = make(chan bool)
	f, err := os.OpenFile(filepath, os.O_RDONLY, 0644)
	if err != nil {
		//log.Fatalf("couldn't open file %s", filepath)
	}
	defer func() {
		if err := f.Close(); err != nil {
			//log.Printf("couldn't close file %s", filepath)
		}
	}()
	var allRecoredBytesData = []EvItem{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		recordItem := EvItem{}
		line := scanner.Text()
		//log.Printf(fmt.Sprintf("%d", sleeping))
		lineBytes := []byte(line)
		if err != nil {
			//log.Fatalf("couldn't parse line string from file %s", filepath)
		}
		//log.Printf("got line string: %s", line)
		if err := json.Unmarshal(lineBytes, &recordItem); err != nil {
			//log.Fatalf("couldn't parse line bytes: %s", err.Error())
		}
		allRecoredBytesData = append(allRecoredBytesData, recordItem)
	}

	//log.Printf("start play")
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
				_, err := os.Stdout.Write(recordItem.Data)
				lastTime = recordItem.At
				if err != nil {
					continue
				}
			}
		}
	}(signal)

	go func() {
		key := make([]byte, 1)
		signal <- 1
		keySpaceCounter := 0
		for {
			_, err := os.Stdin.Read(key)
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
}

func wait(c chan bool) {
	<-c
}

func main() {
	fl := flag.String("f", "", "")
	flag.Parse()
	Play(*fl)
}