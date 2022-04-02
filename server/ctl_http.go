package server

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func testHttpGet(ctx *gin.Context) {
	fmt.Println("handling reqeust...")
	ctx.JSON(http.StatusOK, newResponse(TESTFAILED, "ok", nil))
}