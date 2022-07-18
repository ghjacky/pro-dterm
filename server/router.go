package server

import "github.com/gin-gonic/gin"

func newResponse(code int, message string, data interface{}, extras ...map[string]interface{}) interface{} {
	var d = map[string]interface{}{
		"code":    code,
		"message": message,
		"data":    data,
	}
	if len(extras) <= 0 {
		return d
	}
	for k, v := range extras[0] {
		d[k] = v
	}
	return d
}

func httpRoutes(r *gin.Engine) {
	tr := r.Group("/api/recorder")
	{
		tr.GET("", fetchRecords)
		tr.GET("/:id", getRecord)
	}
}

func wsRoutes(r *gin.Engine) {
	wr := r.Group("ws")
	{
		wr.GET("/container/log/:name", streamLog)
		wr.GET("/container/exec/:name", streamExec)
	}
}
