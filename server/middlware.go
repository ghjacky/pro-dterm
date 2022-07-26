package server

import (
	"bytes"
	"crypto/tls"
	"dterm/base"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"path"

	"github.com/gin-gonic/gin"
)

func beforeRequest() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		token := ctx.Query("access_token")
		if auth := checkAccessTokenFromEasy(token); auth != nil {
			username, _ := auth["username"].(string)
			ctx.Set("username", username)
			ctx.Next()
		} else {
			ctx.Abort()
		}
	}
}

func checkAccessTokenFromEasy(token string) map[string]interface{} {
	// return map[string]interface{}{"username": "gmy"}
	url := base.Conf.EasyConfiguration.Schema + "://" + path.Join(base.Conf.EasyConfiguration.Domain, base.Conf.EasyConfiguration.ApiCheckToken)
	data := map[string]interface{}{
		"access_token": token,
	}
	hc := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	b, _ := json.Marshal(data)
	hq, err := http.NewRequest("POST", url, bytes.NewBuffer(b))
	if err != nil {
		base.Log.Errorf("failed to create token checking request: %s", err.Error())
		return nil
	}
	res, err := hc.Do(hq)
	if err != nil {
		base.Log.Errorf("failed to get response of token checking from easy: %s", err.Error())
		return nil
	}
	d, err := ioutil.ReadAll(res.Body)
	if err != nil {
		base.Log.Errorf("failed to read response of token checking from easy: %s", err.Error())
		return nil
	}
	var m = map[string]interface{}{}
	err = json.Unmarshal(d, &m)
	if err != nil {
		base.Log.Errorf("failed to unmarshal response of token checking from easy: %s", err.Error())
		return nil
	}
	var code interface{} = float64(200)
	mc, _ := m["code"]
	if mc != code {
		base.Log.Errorf("invalid token: %s", m["message"])
		return nil
	}
	md, _ := m["data"].(map[string]interface{})
	return md
}
