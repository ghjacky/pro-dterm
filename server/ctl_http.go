package server

import (
	"dterm/base"
	"dterm/model"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func fetchRecords(ctx *gin.Context) {
	var pq = model.PageQuery{}
	if err := ctx.BindQuery(&pq); err != nil {
		ctx.JSON(http.StatusOK, newResponse(1004, "bad parameters", nil))
		return
	}
	var evrcds = model.MRecords{
		TX:  base.DB(),
		PQ:  pq,
		ALL: []model.MRecord{},
	}
	if err := evrcds.FetchList(); err != nil {
		ctx.JSON(http.StatusOK, newResponse(1005, err.Error(), nil))
		base.Log.Errorf("failed to fetch records: %s", err.Error())
		return
	}
	ctx.JSON(http.StatusOK, newResponse(0, "ok", evrcds.ALL, map[string]interface{}{"total": evrcds.PQ.Total}))
}

func getRecord(ctx *gin.Context) {
	id, _ := strconv.Atoi(ctx.Param("id"))
	if id == 0 {
		ctx.JSON(http.StatusOK, newResponse(1014, "bad record id", nil))
		return
	}
	var evrcd = model.MRecord{}
	evrcd.TX = base.DB()
	evrcd.ID = uint(id)
	if err := evrcd.Get(); err != nil {
		ctx.JSON(http.StatusOK, newResponse(1015, err.Error(), nil))
		base.Log.Errorf("failed to get record by id: %s", err.Error())
		return
	}
	ctx.JSON(http.StatusOK, newResponse(0, "ok", evrcd))
}
