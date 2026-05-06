package openai

import (
	"sort"

	"github.com/superplanehq/superplane/pkg/integrations/openai/components"
)

const (
	PermissionAccessRead  uint8 = 0
	PermissionAccessWrite uint8 = 1

	PermissionEndpointModels    = "/v1/models"
	PermissionEndpointResponses = "/v1/responses"
)

type CapabilityMapper struct {
	capabilities map[string][]PermissionRequirement
}

type PermissionRequirement struct {
	Endpoint string
	Access   uint8
}

type Permission struct {
	Name   string
	Access string
}

type PermissionSet map[string]uint8

func NewCapabilityMapper() *CapabilityMapper {
	return &CapabilityMapper{
		capabilities: map[string][]PermissionRequirement{
			(&components.CreateResponse{}).Name(): {
				{Endpoint: PermissionEndpointResponses, Access: PermissionAccessWrite},
			},
		},
	}
}

func (m *CapabilityMapper) PermissionSet(capabilities []string, includeBaseline bool) PermissionSet {
	out := PermissionSet{}
	if includeBaseline {
		out.add(PermissionEndpointModels, PermissionAccessRead)
	}

	for _, capability := range capabilities {
		for _, permission := range m.capabilities[capability] {
			out.add(permission.Endpoint, permission.Access)
		}
	}

	return out
}

func (s PermissionSet) IsEmpty() bool {
	return len(s) == 0
}

func (s PermissionSet) ForHuman() []Permission {
	endpoints := make([]string, 0, len(s))
	for endpoint := range s {
		endpoints = append(endpoints, endpoint)
	}
	sort.Strings(endpoints)

	permissions := []Permission{}
	for _, endpoint := range endpoints {
		permissions = append(permissions, Permission{
			Name:   permissionName(endpoint),
			Access: accessString(s[endpoint]),
		})
	}

	return permissions
}

func (s PermissionSet) add(endpoint string, access uint8) {
	current, ok := s[endpoint]
	if !ok || access > current {
		s[endpoint] = access
	}
}

func FindPermissionUpdates(existing PermissionSet, requested PermissionSet) PermissionSet {
	diff := PermissionSet{}
	for endpoint, requestedAccess := range requested {
		existingAccess, ok := existing[endpoint]
		if !ok || requestedAccess > existingAccess {
			diff[endpoint] = requestedAccess
		}
	}

	return diff
}

func accessString(access uint8) string {
	if access == PermissionAccessWrite {
		return "Write"
	}

	return "Read"
}

func permissionName(endpoint string) string {
	switch endpoint {
	case PermissionEndpointModels:
		return "List models"
	case PermissionEndpointResponses:
		return "Responses (/v1/responses)"
	default:
		return endpoint
	}
}
