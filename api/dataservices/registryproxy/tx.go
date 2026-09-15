package registryproxy

import (
	portainer "github.com/portainer/portainer/api"
	"github.com/portainer/portainer/api/dataservices"
)

type ServiceTx struct {
	dataservices.BaseDataServiceTx[portainer.RegistryProxy, portainer.RegistryProxyID]
}

// Create creates a new registry proxy.
func (service ServiceTx) Create(registryProxy *portainer.RegistryProxy) error {
	return service.Tx.CreateObject(
		BucketName,
		func(id uint64) (int, any) {
			registryProxy.ID = portainer.RegistryProxyID(id)
			return int(registryProxy.ID), registryProxy
		},
	)
}
