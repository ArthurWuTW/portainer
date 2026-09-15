package registryproxy

import (
	portainer "github.com/portainer/portainer/api"
	"github.com/portainer/portainer/api/dataservices"
)

// BucketName represents the name of the bucket where this service stores data.
const BucketName = "registry_proxies"

// Service represents a service for managing registry proxy data.
type Service struct {
	dataservices.BaseDataService[portainer.RegistryProxy, portainer.RegistryProxyID]
}

// NewService creates a new instance of a service.
func NewService(connection portainer.Connection) (*Service, error) {
	err := connection.SetServiceName(BucketName)
	if err != nil {
		return nil, err
	}

	return &Service{
		BaseDataService: dataservices.BaseDataService[portainer.RegistryProxy, portainer.RegistryProxyID]{
			Bucket:     BucketName,
			Connection: connection,
		},
	}, nil
}

func (service *Service) Tx(tx portainer.Transaction) ServiceTx {
	return ServiceTx{
		BaseDataServiceTx: dataservices.BaseDataServiceTx[portainer.RegistryProxy, portainer.RegistryProxyID]{
			Bucket:     BucketName,
			Connection: service.Connection,
			Tx:         tx,
		},
	}
}

// Create creates a new registry proxy.
func (service *Service) Create(registryProxy *portainer.RegistryProxy) error {
	return service.Connection.CreateObject(
		BucketName,
		func(id uint64) (int, any) {
			registryProxy.ID = portainer.RegistryProxyID(id)
			return int(registryProxy.ID), registryProxy
		},
	)
}
