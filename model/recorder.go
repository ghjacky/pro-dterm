package model

import "gorm.io/gorm"

type MRecord struct {
	BaseModel
	Instance string `json:"instance" gorm:"not null;comment:登陆主机ip或主机名或容器名称"`
	Username string `json:"username" gorm:"not null;comment:用户名"`
	Filepath string `json:"filepath" gorm:"type:varchar(64);not null;uniqueIndex;comment:记录文件相对路径"`
	StartAt  int64  `json:"startAt" gorm:"not null;comment:记录开始时间"`
	EndAt    int64  `json:"endAt" gorm:"not null;comment:记录结束时间"`
}

type MRecords struct {
	PQ  PageQuery
	TX  *gorm.DB
	ALL []MRecord
}

func (*MRecord) TableName() string {
	return "tb_webterminal_recorder"
}

func (rcd *MRecord) Add() error {
	return rcd.TX.Create(rcd).Error
}

func (rcd *MRecord) Update() error {
	return rcd.TX.Save(rcd).Error
}

func (rcd *MRecord) Get() error {
	return rcd.TX.First(rcd).Error
}

func (rcds *MRecords) FetchList() error {
	return rcds.PQ.Query(rcds.TX, &rcds.ALL).Error
}
