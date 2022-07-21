package model

import (
	"gorm.io/gorm"
)

const (
	CmdLevelDefault uint8 = iota
	CmdLevelWarn
	CmdLevelDanger
)

type MCommand struct {
	BaseModel
	Username string `json:"username"`
	Instance string `json:"instance"`
	Command  string `json:"command"`
	Result   string `json:"result"`
	Level    uint8  `json:"level"`
	At       int64  `json:"at"`
}

type MCommands struct {
	PQ  PageQuery
	TX  *gorm.DB
	ALL []MCommand
}

func (*MCommand) TableName() string {
	return "tb_webterminal_command"
}

func (cmd *MCommand) Add() error {
	return cmd.TX.Create(cmd).Error
}

func (cmd *MCommand) Get() error {
	return cmd.TX.First(cmd).Error
}

func (cmds *MCommands) Fetch() error {
	return cmds.PQ.Query(cmds.TX, &cmds.ALL).Error
}
