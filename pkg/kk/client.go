package kk

import (
	// "crypto/tls"
	// "net/http"

	"dterm/base"

	docker_client "github.com/docker/docker/client"
	"github.com/gorilla/websocket"
)

type DC struct {
	Client   *docker_client.Client
	WSClient *websocket.Dialer
}

func newClient(host string) *DC {
	dc, err := docker_client.NewClientWithOpts(docker_client.WithAPIVersionNegotiation(),
		docker_client.WithHost(host),
		docker_client.WithScheme("http"),
		docker_client.WithAPIVersionNegotiation(),
		// docker_client.WithHTTPClient(&http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}),
	)
	if err != nil {
		base.Log.Errorf("failed to create docker client with addr(%s): %s", host, err.Error())
		return nil
	}
	return &DC{
		Client:   dc,
		WSClient: websocket.DefaultDialer,
	}
}
