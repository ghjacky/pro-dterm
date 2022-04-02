package kk

import (
	"context"
	"dterm/base"
	"errors"

	docker_types "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
)

type DContainer struct {
	*DC
	Containers []docker_types.Container
}

func NewDContainer(dproxy string) *DContainer {
	return &DContainer{
		DC: newClient(dproxy),
	}
}

func (c *DContainer) GetByIp(ip string) error {
	if c.DC == nil {
		return errors.New("nil docker client")
	}
	if cs, err := c.DC.Client.ContainerList(context.Background(), docker_types.ContainerListOptions{
		Filters: filters.NewArgs(
			filters.KeyValuePair{
				Key:   "name",
				Value: ip,
			},
		),
	}); err != nil {
		base.Log.Errorf("failed to get container by ip(%s): %s", ip, err.Error())
		return err
	} else {
		c.Containers = cs
		return nil
	}
}
