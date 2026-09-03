package serviceinstance

import (
	portainer "github.com/portainer/portainer/api"
	"github.com/portainer/portainer/api/dataservices"
)

type ServiceTx struct {
	dataservices.BaseDataServiceTx[portainer.ServiceInstance, portainer.ServiceInstanceID]
}

// GetNextIdentifier returns the next identifier for a service instance.
func (service ServiceTx) GetNextIdentifier() int {
	return service.Tx.GetNextIdentifier(BucketName)
}

// Create creates a new service instance.
func (service ServiceTx) Create(instance *portainer.ServiceInstance) error {
	return service.Tx.CreateObjectWithId(BucketName, int(instance.ID), instance)
}
