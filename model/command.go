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
	Username   string  `json:"username" gorm:"not null;comment:用户名"`
	Instance   string  `json:"instance" gorm:"not null;comment:容器实例"`
	Command    string  `json:"command" gorm:"not null;comment:操作命令"`
	Result     string  `json:"result" gorm:"comment:命令执行结果"`
	Level      uint8   `json:"level" gorm:"not null;default:0;comment:命令安全等级"`
	At         int64   `json:"at" gorm:"not null;comment:命令执行时间点"`
	RecordFile string  `json:"recordFile" gorm:"type:varchar(255);not null;index"`
	Record     MRecord `json:"record" gorm:"foreignKey:RecordFile;references:Filepath"`
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

func (cmds *MCommands) FetchList() error {
	return cmds.PQ.Query(cmds.TX, &cmds.ALL).Error
}
