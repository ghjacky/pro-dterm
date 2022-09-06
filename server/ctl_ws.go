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
	var wsupgrader = websocket.Upgrader{
		EnableCompression: true,
		CheckOrigin:       func(r *http.Request) bool { return true },
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
	var wsupgrader = websocket.Upgrader{
		EnableCompression: true,
		CheckOrigin:       func(r *http.Request) bool { return true },
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

func streamK8sContainerLog(ctx *gin.Context) {
	namespace := ctx.Query("namespace")
	podname := ctx.Param("podname")
	container := ctx.Query("container")
	if len(podname) <= 0 {
		ctx.JSON(http.StatusBadRequest, newResponse(1600, "no podname provided", nil))
	}
	if len(container) <= 0 {
		container = podname
	}
	if len(namespace) <= 0 {
		namespace = "default"
	}
	var wsupgrader = websocket.Upgrader{
		EnableCompression: true,
		CheckOrigin:       func(r *http.Request) bool { return true },
	}
	conn, err := wsupgrader.Upgrade(ctx.Writer, ctx.Request, http.Header{})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, newResponse(1601, err.Error(), nil))
		return
	}
	defer conn.Close()

	if err := kk.StreamPodLog(conn, namespace, podname, container); err != nil {
		base.Log.Errorf("streaming log error: %s", err.Error())
	}
}

func streamK8sContainerExec(ctx *gin.Context) {
	namespace := ctx.Query("namespace")
	podname := ctx.Param("podname")
	container := ctx.Query("container")
	user := ctx.GetString("username")
	if len(podname) <= 0 {
		ctx.JSON(http.StatusBadRequest, newResponse(1600, "no podname provided", nil))
	}
	if len(user) <= 0 {
		ctx.JSON(http.StatusBadRequest, newResponse(1600, "no user provided", nil))
	}
	if len(container) <= 0 {
		container = podname
	}
	if len(namespace) <= 0 {
		namespace = "default"
	}
	var wsupgrader = websocket.Upgrader{
		EnableCompression: true,
		CheckOrigin:       func(r *http.Request) bool { return true },
	}
	conn, err := wsupgrader.Upgrade(ctx.Writer, ctx.Request, http.Header{})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, newResponse(1601, err.Error(), nil))
		return
	}
	defer conn.Close()

	if err := kk.StreamPodShell(conn, namespace, podname, user, container); err != nil {
		base.Log.Errorf("pod exec error: %s", err.Error())
	}
}

func streamRecorderPlayback(ctx *gin.Context) {
	commandId, _ := strconv.Atoi(ctx.Param("cid"))
	if commandId == 0 {
		ctx.JSON(http.StatusBadRequest, newResponse(1700, "wrong command id", nil))
		return
	}
	var wsupgrader = websocket.Upgrader{
		EnableCompression: true,
		CheckOrigin:       func(r *http.Request) bool { return true },
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
