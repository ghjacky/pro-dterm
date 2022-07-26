package pty

import (
	"bytes"
	"dterm/base"
	"dterm/model"
	"time"
)

const (
	StageInitial uint8 = iota
	StageInputing
	StageCancel
	StageTabKey
	StageDelKey
	StageLeftKey
	StageRightKey
	StageUpKey
	StageDownKey
	StageRemoveOneWorldKey
	StageRemoveOneLineKey
	StageWaitingExec
	StageInteracting
	StageExit
)

var (
	EndLineKey        = []byte{0xa, 0xa}
	RemoveOneWorldKey = []byte{0x17}
	RemoveOneLineKey  = []byte{0x15}

	DelCharacter         = []byte("\x1b[1P")
	ControlCharacterFlag = []byte("\x1b")
	PromptCommand        = []byte("\x1b]0;")
	BellCharacter        = []byte("\a")
	XtermMetaModeOn      = []byte("\x1b[?1034h")
	XtermMetaModeOff     = []byte("\x1b[?1034l")
	XtermPasteModeOn     = []byte("\x1b[?2004h")
	XtermPasteModeOff    = []byte("\x1b[?2004l")
)

func (sp *StreamParser) StartBashParser() {
	go sp.HandleOutput()
	go sp.HandleInput()
}

func (sp *StreamParser) HandleAutoCompletion(p []byte) {
	if bytes.Compare(p, BellCharacter) == 0 {
		return
	} else if !bytes.Contains(p, sp.cmd.prompt) {
		sp.AppendCommandChar(removeControlCharacters(removeBellCharacter(bytes.Split(p, []byte(" "))[0])))
	} else {
		return
	}
}

func (sp *StreamParser) HandleRemoveOneCharacter(p []byte) {
	if sp.cmd.cursor <= len(sp.cmd.prompt) {
		return
	} else {
		sp.cmd.command = append(sp.cmd.command[:sp.cmd.cursor-1], sp.cmd.command[sp.cmd.cursor:]...)
		sp.MoveCursor(sp.cmd.cursor - 1)
	}
}

func (sp *StreamParser) HandleRemoveOneWorld(p []byte) {
	for c := bytes.Count(p, BackspaceKey); c > 0; c-- {
		sp.HandleRemoveOneCharacter([]byte{})
	}
}

func (sp *StreamParser) HandleRemoveOneLine(p []byte) {
	sp.InitCommand()
}

func (sp *StreamParser) HandleInput() {
	for {
		select {
		case in := <-sp.inChan:
			if bytes.Compare(in, EnterKey) == 0 {
				// calculate command && change stage to waiting command exec (indicating the input stage done)
				sp.ChangeStage(StageWaitingExec)
				continue
			} else if bytes.Compare(in, CancelKey) == 0 {
				// reset and init command and result (init with a new input stage)
				sp.ChangeStage(StageCancel)
				continue
			} else {
				if bytes.Compare(in, TabKey) == 0 {
					// auto completion with the rest chars of the command
					sp.ChangeStage(StageTabKey)
					continue
				} else if bytes.Compare(in, DelKey) == 0 {
					// remove a char from current position to left && change cursor position and command
					sp.ChangeStage(StageDelKey)
					continue
				} else if bytes.Compare(in, RemoveOneWorldKey) == 0 {
					sp.ChangeStage(StageRemoveOneWorldKey)
					continue
				} else if bytes.Compare(in, RemoveOneLineKey) == 0 {
					sp.ChangeStage(StageRemoveOneLineKey)
					continue
				} else if bytes.Compare(in, LeftKey) == 0 {
					// move cursor one index from current position to left
					if sp.cmd.cursor > len(sp.cmd.prompt) {
						sp.MoveCursor(sp.cmd.cursor - 1)
					}
				} else if bytes.Compare(in, RightKey) == 0 {
					// move cursor one index from current position to right
					if sp.cmd.cursor < len(sp.cmd.command) {
						sp.MoveCursor(sp.cmd.cursor + 1)
					}
				} else if bytes.Compare(in, UpKey) == 0 {
					// reset whole command to new one
					sp.ChangeStage(StageUpKey)
					continue
				} else if bytes.Compare(in, DownKey) == 0 {
					// similar acting as upkey
					sp.ChangeStage(StageDownKey)
					continue
				} else if sp.TreatAsNormalCommandChars(in) {
					sp.AppendCommandChar(in)
				}
				continue
			}
		}
	}
}

func (sp *StreamParser) HandleOutput() {
	var outb []byte = nil
	for {
		select {
		case out := <-sp.outChan:
			switch sp.stage {
			case StageInitial:
				outb = append(outb, out...)
				if sp.StageDone(StageInitial, out) {
					sp.ParseInitialOutputStream(outb)
					outb = nil
					sp.ChangeStage(StageInputing)
					continue
				} else {
					continue
				}
			case StageInputing:
				sp.ParseInputingOutputStream(out)
				continue
			case StageCancel:
				sp.ParseCancelOutputStream(out)
				sp.ChangeStage(StageInputing)
			case StageTabKey:
				sp.HandleAutoCompletion(out)
				sp.ChangeStage(StageInputing)
				continue
			case StageDelKey:
				sp.HandleRemoveOneCharacter(out)
				sp.ChangeStage(StageInputing)
				continue
			case StageLeftKey:
				continue
			case StageRightKey:
				continue
			case StageUpKey:
				sp.HandleUpOrDownKey(out)
				sp.ChangeStage(StageInputing)
				continue
			case StageDownKey:
				sp.HandleUpOrDownKey(out)
				sp.ChangeStage(StageInputing)
				continue
			case StageRemoveOneWorldKey:
				sp.HandleRemoveOneWorld(out)
				sp.ChangeStage(StageInputing)
				continue
			case StageRemoveOneLineKey:
				sp.HandleRemoveOneLine(out)
				sp.ChangeStage(StageInputing)
				continue
			case StageWaitingExec:
				outb = append(outb, out...)
				if sp.StageDone(StageWaitingExec, out) {
					sp.ParseExecOutputStream(outb)
					outb = nil
					sp.ChangeStage(StageInputing)
					continue
				} else {
					continue
				}
			case StageInteracting:
			case StageExit:
			default:
				continue
			}
		}
	}
}

func (sp *StreamParser) HandleUpOrDownKey(p []byte) {
	var first bool = true
	for len(p) > 0 {
		if !bytes.Contains(p, BackspaceKey) && !bytes.Contains(p, DelCharacter) && !bytes.Contains(p, RightKey) && first {
			sp.cmd.command = append(sp.cmd.prompt, p...)
			sp.MoveCursor(sp.cmd.cursor + len(p))
			break
		} else if bytes.HasPrefix(p, BackspaceKey) {
			first = false
			sp.MoveCursor(sp.cmd.cursor - 1)
			p = p[len(BackspaceKey):]
		} else if bytes.HasPrefix(p, DelCharacter) {
			first = false
			sp.cmd.command = append(sp.cmd.command[:sp.cmd.cursor], sp.cmd.command[sp.cmd.cursor+1:]...)
			p = p[len(DelCharacter):]
		} else if bytes.HasPrefix(p, RightKey) {
			first = false
			sp.MoveCursor(sp.cmd.cursor + 1)
			p = p[len(RightKey):]
		} else {
			if sp.cmd.cursor >= len(sp.cmd.command) {
				sp.cmd.command = append(sp.cmd.command, p[0])
			} else {
				sp.cmd.command[sp.cmd.cursor] = p[0]
			}
			sp.MoveCursor(sp.cmd.cursor + 1)
			if len(p) >= 2 {
				p = p[1:]
			} else {
				p = []byte{}
			}
		}
	}
}

func removeBackspaceCharacters(p []byte) []byte {
	return bytes.ReplaceAll(p, BackspaceKey, []byte{})
}

func removeControlCharacters(p []byte) []byte {
	return ControlCharactersPattern.ReplaceAll(p, []byte{})
}

func removeEndlineCharacters(p []byte) []byte {
	return bytes.ReplaceAll(p, EndLineKey, []byte{})
}

func removeBellCharacter(p []byte) []byte {
	return bytes.ReplaceAll(p, BellCharacter, []byte{})
}

func (sp *StreamParser) ParseAndInitCommandWithSettingPrompt(p []byte) {
	s := bytes.Split(removeControlCharacters(removeEndlineCharacters(p)), BellCharacter)
	if len(s) >= 2 && len(s[1]) > 0 {
		sp.SetPrompt(s[1])
		sp.InitCommand()
	} else {
		base.Log.Warnf("maybe got wrong prompt string: %s", p)
	}
}

func (sp *StreamParser) ParseInitialOutputStream(p []byte) {
	// parse and set prompt string and init command with correct cursor position
	sp.ParseAndInitCommandWithSettingPrompt(p)
}

func (sp *StreamParser) ParseInputingOutputStream(p []byte) {

}
func (sp *StreamParser) ParseCancelOutputStream(p []byte) {
	sp.ParseAndInitCommandWithSettingPrompt(p)
}

func (sp *StreamParser) ParseExecOutputStream(p []byte) {
	// parse result and set new prompt and init command with correct cursor position
	s := bytes.Split(removeEndlineCharacters(removeControlCharacters(p)), PromptCommand)
	if enterVim(p) && exitVim(p) {
		sp.CalcCMDWithResult()
		sp.ParseAndInitCommandWithSettingPrompt(s[1])
		return
	}
	if len(s) >= 2 {
		sp.cmd.result = s[0]
		// calculate command and result
		sp.CalcCMDWithResult()
		sp.ParseAndInitCommandWithSettingPrompt(s[1])
	}
}
func (sp *StreamParser) ParseInteractOutputStream(p []byte) {

}

func (sp *StreamParser) StageDone(stage uint8, p []byte) bool {
	switch stage {
	case StageInitial:
		if bytes.Contains(p, XtermMetaModeOn) || bytes.Contains(p, XtermPasteModeOn) {
			return true
		} else {
			return false
		}
	case StageInputing:
	case StageCancel:
	case StageWaitingExec:
		if sp.isRequestForPrompt(p) {
			return true
		} else {
			return false
		}
	case StageInteracting:
	case StageExit:
	default:
		return false
	}
	return false
}

func (sp *StreamParser) CalcCMDWithResult() {
	defer sp.ResetResult()
	var cmd = model.MCommand{
		Username: sp.username,
		Instance: sp.instance,
	}
	cmd.Command = string(sp.cmd.command[len(sp.cmd.prompt):])
	if len(cmd.Command) <= 0 {
		return
	}
	cmd.Result = string(sp.cmd.result)
	cmd.At = time.Now().Local().UnixNano()
	cmd.TX = base.DB()
	cmd.RecordFile = sp.filepath
	sp.cmdChan <- cmd
	// fmt.Printf("command: %s\n\n", cmd.Command)
	// fmt.Printf("result: %s\n\n", cmd.Result)
}

func (sp *StreamParser) AppendCommandChar(p []byte) {
	if sp.cmd.cursor >= len(sp.cmd.command) {
		sp.cmd.command = append(sp.cmd.command, p...)
		sp.MoveCursor(sp.cmd.cursor + len(p))
	} else {
		var before, behind = make([]byte, sp.cmd.cursor), make([]byte, len(sp.cmd.command)-sp.cmd.cursor)
		copy(before, sp.cmd.command[:sp.cmd.cursor])
		copy(behind, sp.cmd.command[sp.cmd.cursor:])
		c := append(before, p...)
		sp.cmd.command = append(c, behind...)
		sp.MoveCursor(sp.cmd.cursor + len(p))
	}
}

func (sp *StreamParser) TreatAsNormalCommandChars(in []byte) bool {
	if len(in) > 1 && !bytes.Contains(in, ControlCharacterFlag) {
		return true
	} else if len(in) == 1 && in[0] < 127 && in[0] >= 32 {
		return true
	} else {
		return false
	}
}

func (sp *StreamParser) ChangeStage(stage uint8) {
	sp.stage = stage
}

func (sp *StreamParser) SetPrompt(prompt []byte) {
	sp.cmd.prompt = prompt
}

func (sp *StreamParser) MoveCursor(to int) {
	sp.cmd.cursor = to
}

func (sp *StreamParser) InitCommand() {
	if len(sp.cmd.prompt) <= 0 {
		base.Log.Warnf("got empty prompt !!!")
	} else {
		sp.cmd.command = sp.cmd.prompt[:]
		sp.MoveCursor(len(sp.cmd.prompt))
	}
}

func (sp *StreamParser) ResetResult() {
	sp.cmd.result = make([]byte, 0)
}

func (sp *StreamParser) isRequestForPrompt(p []byte) bool {
	return bytes.Contains(p, PromptCommand)
}
