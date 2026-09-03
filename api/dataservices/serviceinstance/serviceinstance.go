package serviceinstance

import (
	portainer "github.com/portainer/portainer/api"
	"github.com/portainer/portainer/api/dataservices"
)

// BucketName represents the name of the bucket where this service stores data.
const BucketName = "service_instances"

// Service represents a service for managing service instance data.
type Service struct {
	dataservices.BaseDataService[portainer.ServiceInstance, portainer.ServiceInstanceID]
}

// NewService creates a new instance of a service.
func NewService(connection portainer.Connection) (*Service, error) {
	err := connection.SetServiceName(BucketName)
	if err != nil {
		return nil, err
	}

	return &Service{
		BaseDataService: dataservices.BaseDataService[portainer.ServiceInstance, portainer.ServiceInstanceID]{
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

// GetNextIdentifier returns the next identifier for a service instance.
func (service *Service) GetNextIdentifier() int {
	return service.Connection.GetNextIdentifier(BucketName)
}

// Create creates a new service instance.
func (service *Service) Create(instance *portainer.ServiceInstance) error {
	return service.Connection.CreateObjectWithId(BucketName, int(instance.ID), instance)
}
