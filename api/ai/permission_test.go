package ai

import (
	"testing"

	portainer "github.com/portainer/portainer/api"
	"github.com/portainer/portainer/api/internal/testhelpers"

	"github.com/stretchr/testify/require"
)

func adminUser() *portainer.User {
	return &portainer.User{
		ID:       1,
		Username: "admin",
		Role:     portainer.AdministratorRole,
	}
}

func standardUser() *portainer.User {
	return &portainer.User{
		ID:       2,
		Username: "bob",
		Role:     portainer.StandardUserRole,
	}
}

func endpointWithAccess(id portainer.EndpointID, userIDs ...portainer.UserID) portainer.Endpoint {
	policies := portainer.UserAccessPolicies{}
	for _, uid := range userIDs {
		policies[uid] = portainer.AccessPolicy{RoleID: 1}
	}

	return portainer.Endpoint{
		ID:                 id,
		Name:               "endpoint-" + string(rune('a'+id-1)),
		Type:               portainer.DockerEnvironment,
		GroupID:            1,
		UserAccessPolicies: policies,
	}
}

func TestCanAccessEndpoint_AdminAlwaysAllowed(t *testing.T) {
	t.Parallel()

	tc := &ToolContext{
		User: adminUser(),
	}

	endpoint := endpointWithAccess(1) // no access policies at all
	require.True(t, tc.CanAccessEndpoint(&endpoint))
}

func TestCanAccessEndpoint_StandardUserWithDirectAccess(t *testing.T) {
	t.Parallel()

	tc := &ToolContext{
		User: standardUser(),
	}

	endpoint := endpointWithAccess(1, 2) // user 2 has direct access
	require.True(t, tc.CanAccessEndpoint(&endpoint))
}

func TestCanAccessEndpoint_StandardUserWithoutAccess(t *testing.T) {
	t.Parallel()

	tc := &ToolContext{
		User: standardUser(),
	}

	endpoint := endpointWithAccess(1, 3) // only user 3 has access
	require.False(t, tc.CanAccessEndpoint(&endpoint))
}

func TestCanAccessEndpoint_ViaTeamMembership(t *testing.T) {
	t.Parallel()

	tc := &ToolContext{
		User: standardUser(),
		TeamMemberships: []portainer.TeamMembership{
			{UserID: 2, TeamID: 10, Role: portainer.TeamMember},
		},
	}

	endpoint := portainer.Endpoint{
		ID:     1,
		Name:   "team-endpoint",
		Type:   portainer.DockerEnvironment,
		GroupID: 1,
		TeamAccessPolicies: portainer.TeamAccessPolicies{
			10: {RoleID: 1},
		},
	}

	require.True(t, tc.CanAccessEndpoint(&endpoint))
}

func TestCanAccessEndpoint_ViaEndpointGroup(t *testing.T) {
	t.Parallel()

	tc := &ToolContext{
		User: standardUser(),
		EndpointGroups: []portainer.EndpointGroup{
			{
				ID:     1,
				Name:   "default-group",
				UserAccessPolicies: portainer.UserAccessPolicies{
					2: {RoleID: 1},
				},
			},
		},
	}

	// Endpoint itself has no direct policies, but its group grants access.
	endpoint := endpointWithAccess(1)
	require.True(t, tc.CanAccessEndpoint(&endpoint))
}

func TestAuthorizedEndpoints_FiltersForStandardUser(t *testing.T) {
	t.Parallel()

	ds := testhelpers.NewDatastore(testhelpers.WithEndpoints([]portainer.Endpoint{
		endpointWithAccess(1, 2), // accessible to user 2
		endpointWithAccess(2, 3), // not accessible to user 2
	}))

	tc := &ToolContext{
		DataStore: ds,
		User:      standardUser(),
		EndpointGroups: []portainer.EndpointGroup{
			{ID: 1, Name: "default"},
		},
	}

	endpoints, err := tc.AuthorizedEndpoints()
	require.NoError(t, err)
	require.Len(t, endpoints, 1)
	require.Equal(t, portainer.EndpointID(1), endpoints[0].ID)
}

func TestAuthorizedEndpoints_AdminSeesAll(t *testing.T) {
	t.Parallel()

	ds := testhelpers.NewDatastore(testhelpers.WithEndpoints([]portainer.Endpoint{
		endpointWithAccess(1, 3),
		endpointWithAccess(2, 3),
	}))

	tc := &ToolContext{
		DataStore: ds,
		User:      adminUser(),
		EndpointGroups: []portainer.EndpointGroup{
			{ID: 1, Name: "default"},
		},
	}

	endpoints, err := tc.AuthorizedEndpoints()
	require.NoError(t, err)
	require.Len(t, endpoints, 2)
}

func TestAuthorizedEndpoint_DeniedForStandardUser(t *testing.T) {
	t.Parallel()

	ds := testhelpers.NewDatastore(testhelpers.WithEndpoints([]portainer.Endpoint{
		endpointWithAccess(1, 3), // only user 3
	}))

	tc := &ToolContext{
		DataStore: ds,
		User:      standardUser(),
	}

	_, err := tc.AuthorizedEndpoint(1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "permission denied")
}

func TestAuthorizedEndpoint_NotFound(t *testing.T) {
	t.Parallel()

	ds := testhelpers.NewDatastore(testhelpers.WithEndpoints([]portainer.Endpoint{}))

	tc := &ToolContext{
		DataStore: ds,
		User:      adminUser(),
	}

	_, err := tc.AuthorizedEndpoint(99)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}
