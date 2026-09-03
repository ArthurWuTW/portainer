package serviceinstanceoperation

import (
	portainer "github.com/portainer/portainer/api"
	"github.com/portainer/portainer/api/dataservices"
)

type ServiceTx struct {
	dataservices.BaseDataServiceTx[portainer.ServiceInstanceOperation, portainer.ServiceInstanceOperationID]
}

// GetNextIdentifier returns the next identifier for a service instance operation.
func (service ServiceTx) GetNextIdentifier() int {
	return service.Tx.GetNextIdentifier(BucketName)
}

// Create creates a new service instance operation.
func (service ServiceTx) Create(operation *portainer.ServiceInstanceOperation) error {
	return service.Tx.CreateObjectWithId(BucketName, int(operation.ID), operation)
}

// ReadAllByServiceInstanceID returns all operations for the given service instance.
func (service ServiceTx) ReadAllByServiceInstanceID(serviceInstanceID portainer.ServiceInstanceID) ([]portainer.ServiceInstanceOperation, error) {
	return service.ReadAll(func(op portainer.ServiceInstanceOperation) bool {
		return op.ServiceInstanceID == serviceInstanceID
	})
}
