package main

import (
	"dterm/base"
	"dterm/model"
	"dterm/server"

	"github.com/gin-gonic/gin"
)

func main() {
	base.ParseFlag()
	base.Init()
	base.MigrateDB(&model.MRecord{})
	server.RunForever(base.Conf.MainConfiguration.Listen, gin.DebugMode)
}
