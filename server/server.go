package server

import (
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
)

func RunForever(addr, mod string) {
	gin.SetMode(mod)
	r := gin.New()
	r.Use(beforeRequest())
	registerAllSubRoutes(r)
	pprof.Register(r)
	r.Run(addr)
}

func registerAllSubRoutes(r *gin.Engine) {
	httpRoutes(r)
	wsRoutes(r)
}
