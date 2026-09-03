package serviceinstancescheduledbuild

import (
	portainer "github.com/portainer/portainer/api"
	"github.com/portainer/portainer/api/dataservices"
)

type ServiceTx struct {
	dataservices.BaseDataServiceTx[portainer.ServiceInstanceScheduledBuild, portainer.ServiceInstanceScheduledBuildID]
}

// GetNextIdentifier returns the next identifier for a service instance scheduled build.
func (service ServiceTx) GetNextIdentifier() int {
	return service.Tx.GetNextIdentifier(BucketName)
}

// Create creates a new service instance scheduled build.
func (service ServiceTx) Create(build *portainer.ServiceInstanceScheduledBuild) error {
	return service.Tx.CreateObjectWithId(BucketName, int(build.ID), build)
}

// ReadAllByServiceInstanceID returns all scheduled builds for the given service instance.
func (service ServiceTx) ReadAllByServiceInstanceID(serviceInstanceID portainer.ServiceInstanceID) ([]portainer.ServiceInstanceScheduledBuild, error) {
	return service.ReadAll(func(build portainer.ServiceInstanceScheduledBuild) bool {
		return build.ServiceInstanceID == serviceInstanceID
	})
}
