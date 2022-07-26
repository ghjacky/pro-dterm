package server

import (
	"dterm/base"
	"dterm/pkg/kk"
	"dterm/pkg/play"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var wsupgrader = websocket.Upgrader{
	ReadBufferSize:    1024 * 1024,
	WriteBufferSize:   1024 * 1024,
	EnableCompression: true,
	CheckOrigin:       func(r *http.Request) bool { return true },
}

func streamLog(ctx *gin.Context) {
	name := ctx.Param("name")
	dproxy := ctx.Query("dproxy")
	if len(dproxy) <= 0 {
		ctx.JSON(http.StatusBadRequest, newResponse(1600, "no docker server provided", nil))
		return
	}
	if len(name) <= 0 {
		ctx.JSON(http.StatusBadRequest, newResponse(1600, "no container name provided", nil))
		return
	}
	conn, err := wsupgrader.Upgrade(ctx.Writer, ctx.Request, http.Header{})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, newResponse(1601, err.Error(), nil))
		return
	}
	defer conn.Close()

	if err := kk.StreamContainerLog(conn, name, dproxy); err != nil {
		base.Log.Errorf("streaming container log error: %s", err.Error())
	}
}

func streamExec(ctx *gin.Context) {
	name := ctx.Param("name")
	dproxy := ctx.Query("dproxy")
	user := ctx.GetString("username")
	if len(dproxy) <= 0 {
		ctx.JSON(http.StatusBadRequest, newResponse(1600, "no docker server provided", nil))
		return
	}
	if len(name) <= 0 {
		ctx.JSON(http.StatusBadRequest, newResponse(1600, "no podname provided", nil))
		return
	}
	if len(user) <= 0 {
		ctx.JSON(http.StatusBadRequest, newResponse(1600, "no user provided", nil))
	}
	conn, err := wsupgrader.Upgrade(ctx.Writer, ctx.Request, http.Header{})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, newResponse(1601, err.Error(), nil))
		return
	}
	defer conn.Close()

	if err := kk.StreamContainerShell(conn, name, dproxy, user); err != nil {
		base.Log.Errorf("docker container exec error: %s", err.Error())
	}
}

func streamRecorderPlayback(ctx *gin.Context) {
	commandId, _ := strconv.Atoi(ctx.Param("cid"))
	if commandId == 0 {
		ctx.JSON(http.StatusBadRequest, newResponse(1700, "wrong command id", nil))
		return
	}
	conn, err := wsupgrader.Upgrade(ctx.Writer, ctx.Request, http.Header{})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, newResponse(1601, err.Error(), nil))
		return
	}
	defer conn.Close()
	// playback stream
	if err := play.Play(uint(commandId), conn); err != nil {
		base.Log.Errorf("record playback error: %s", err.Error())
	}
}
