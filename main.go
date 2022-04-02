package main

import (
	"dterm/base"
	"dterm/server"

	"github.com/gin-gonic/gin"
)

func main() {
	base.ParseFlag()
	base.Init()
	server.RunForever(base.Conf.MainConfiguration.Listen, gin.DebugMode)
}
