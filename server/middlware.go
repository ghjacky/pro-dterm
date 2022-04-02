package server

import (
	"github.com/gin-gonic/gin"
)

func beforeRequest() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Next()
	}
}
