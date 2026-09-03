package serviceinstances

import (
	"net/http"

	portainer "github.com/portainer/portainer/api"
	"github.com/portainer/portainer/api/dataservices"
	"github.com/portainer/portainer/api/http/security"
	httperror "github.com/portainer/portainer/pkg/libhttp/error"
	"github.com/portainer/portainer/pkg/libhttp/response"
)

// @id ServiceInstanceList
// @summary List service instances
// @description List all service instances based on the current user authorizations.
// @description **Access policy**: authenticated
// @tags service_instances
// @security ApiKeyAuth
// @security jwt
// @success 200 {array} portainer.ServiceInstance "Success"
// @failure 500 "Server error"
// @router /service-instances [get]
func (handler *Handler) serviceInstanceList(w http.ResponseWriter, r *http.Request) *httperror.HandlerError {
	instances, err := handler.Service.ReadAll()
	if err != nil {
		return httperror.InternalServerError("Unable to retrieve service instances from the database", err)
	}

	securityContext, err := security.RetrieveRestrictedRequestContext(r)
	if err != nil {
		return httperror.InternalServerError("Unable to retrieve info from request context", err)
	}

	if !securityContext.IsAdmin {
		instances = filterAuthorizedServiceInstances(instances, securityContext, handler.DataStore)
	}

	return response.JSON(w, instances)
}

// filterAuthorizedServiceInstances keeps only the service instances for which
// the user has access to every target environment.
func filterAuthorizedServiceInstances(instances []portainer.ServiceInstance, securityContext *security.RestrictedRequestContext, dataStore dataservices.DataStore) []portainer.ServiceInstance {
	endpoints, err := dataStore.Endpoint().Endpoints()
	if err != nil {
		return nil
	}

	groups, err := dataStore.EndpointGroup().ReadAll()
	if err != nil {
		return nil
	}

	endpointByID := make(map[portainer.EndpointID]portainer.Endpoint, len(endpoints))
	for _, e := range endpoints {
		endpointByID[e.ID] = e
	}

	groupByID := make(map[portainer.EndpointGroupID]portainer.EndpointGroup, len(groups))
	for _, g := range groups {
		groupByID[g.ID] = g
	}

	authorized := make([]portainer.ServiceInstance, 0, len(instances))
	for _, instance := range instances {
		if userHasAccessToAllTargets(instance, securityContext, endpointByID, groupByID) {
			authorized = append(authorized, instance)
		}
	}

	return authorized
}

func userHasAccessToAllTargets(instance portainer.ServiceInstance, securityContext *security.RestrictedRequestContext, endpointByID map[portainer.EndpointID]portainer.Endpoint, groupByID map[portainer.EndpointGroupID]portainer.EndpointGroup) bool {
	var targetIDs []portainer.EndpointID

	switch instance.TargetType {
	case portainer.ServiceInstanceTargetGroup:
		for _, e := range endpointByID {
			if e.GroupID == instance.GroupID {
				targetIDs = append(targetIDs, e.ID)
			}
		}
	case portainer.ServiceInstanceTargetEnvironments:
		targetIDs = instance.EnvironmentIDs
	default:
		return false
	}

	if len(targetIDs) == 0 {
		return false
	}

	for _, id := range targetIDs {
		endpoint, ok := endpointByID[id]
		if !ok {
			continue
		}
		group, ok := groupByID[endpoint.GroupID]
		if !ok {
			continue
		}
		if !security.AuthorizedEndpointAccess(&endpoint, &group, securityContext.UserID, securityContext.UserMemberships) {
			return false
		}
	}

	return true
}
