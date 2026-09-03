package serviceinstanceoperation

import (
	portainer "github.com/portainer/portainer/api"
	"github.com/portainer/portainer/api/dataservices"
)

// BucketName represents the name of the bucket where this service stores data.
const BucketName = "service_instance_operations"

// Service represents a service for managing service instance operation data.
type Service struct {
	dataservices.BaseDataService[portainer.ServiceInstanceOperation, portainer.ServiceInstanceOperationID]
}

// NewService creates a new instance of a service.
func NewService(connection portainer.Connection) (*Service, error) {
	err := connection.SetServiceName(BucketName)
	if err != nil {
		return nil, err
	}

	return &Service{
		BaseDataService: dataservices.BaseDataService[portainer.ServiceInstanceOperation, portainer.ServiceInstanceOperationID]{
			Bucket:     BucketName,
			Connection: connection,
		},
	}, nil
}

func (service *Service) Tx(tx portainer.Transaction) ServiceTx {
	return ServiceTx{
		BaseDataServiceTx: service.BaseDataService.Tx(tx),
	}
}

// GetNextIdentifier returns the next identifier for a service instance operation.
func (service *Service) GetNextIdentifier() int {
	return service.Connection.GetNextIdentifier(BucketName)
}

// Create creates a new service instance operation.
func (service *Service) Create(operation *portainer.ServiceInstanceOperation) error {
	return service.Connection.CreateObjectWithId(BucketName, int(operation.ID), operation)
}

// ReadAllByServiceInstanceID returns all operations for the given service instance.
func (service *Service) ReadAllByServiceInstanceID(serviceInstanceID portainer.ServiceInstanceID) ([]portainer.ServiceInstanceOperation, error) {
	return service.ReadAll(func(op portainer.ServiceInstanceOperation) bool {
		return op.ServiceInstanceID == serviceInstanceID
	})
}
