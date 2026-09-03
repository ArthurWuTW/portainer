package serviceinstancescheduledbuild

import (
	portainer "github.com/portainer/portainer/api"
	"github.com/portainer/portainer/api/dataservices"
)

// BucketName represents the name of the bucket where this service stores data.
const BucketName = "service_instance_scheduled_builds"

// Service represents a service for managing service instance scheduled build data.
type Service struct {
	dataservices.BaseDataService[portainer.ServiceInstanceScheduledBuild, portainer.ServiceInstanceScheduledBuildID]
}

// NewService creates a new instance of a service.
func NewService(connection portainer.Connection) (*Service, error) {
	err := connection.SetServiceName(BucketName)
	if err != nil {
		return nil, err
	}

	return &Service{
		BaseDataService: dataservices.BaseDataService[portainer.ServiceInstanceScheduledBuild, portainer.ServiceInstanceScheduledBuildID]{
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

// GetNextIdentifier returns the next identifier for a service instance scheduled build.
func (service *Service) GetNextIdentifier() int {
	return service.Connection.GetNextIdentifier(BucketName)
}

// Create creates a new service instance scheduled build.
func (service *Service) Create(build *portainer.ServiceInstanceScheduledBuild) error {
	return service.Connection.CreateObjectWithId(BucketName, int(build.ID), build)
}

// ReadAllByServiceInstanceID returns all scheduled builds for the given service instance.
func (service *Service) ReadAllByServiceInstanceID(serviceInstanceID portainer.ServiceInstanceID) ([]portainer.ServiceInstanceScheduledBuild, error) {
	return service.ReadAll(func(build portainer.ServiceInstanceScheduledBuild) bool {
		return build.ServiceInstanceID == serviceInstanceID
	})
}
