package actions

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/authorization"
)

func convertOrgRoleToString(role pb.OrganizationRole) string {
	switch role {
	case pb.OrganizationRole_ORG_VIEWER:
		return authorization.RoleOrgViewer
	case pb.OrganizationRole_ORG_ADMIN:
		return authorization.RoleOrgAdmin
	case pb.OrganizationRole_ORG_OWNER:
		return authorization.RoleOrgOwner
	default:
		return ""
	}
}

func convertCanvasRoleToString(role pb.CanvasRole) string {
	switch role {
	case pb.CanvasRole_CANVAS_VIEWER:
		return authorization.RoleCanvasViewer
	case pb.CanvasRole_CANVAS_ADMIN:
		return authorization.RoleCanvasAdmin
	case pb.CanvasRole_CANVAS_OWNER:
		return authorization.RoleCanvasOwner
	default:
		return ""
	}
}

func convertStringToOrgRole(roleStr string) pb.OrganizationRole {
	switch roleStr {
	case authorization.RoleOrgViewer:
		return pb.OrganizationRole_ORG_VIEWER
	case authorization.RoleOrgAdmin:
		return pb.OrganizationRole_ORG_ADMIN
	case authorization.RoleOrgOwner:
		return pb.OrganizationRole_ORG_OWNER
	default:
		return pb.OrganizationRole_ORG_ROLE_UNSPECIFIED
	}
}

// Helper function to convert string role to protobuf canvas role
func convertStringToCanvasRole(roleStr string) pb.CanvasRole {
	switch roleStr {
	case authorization.RoleCanvasViewer:
		return pb.CanvasRole_CANVAS_VIEWER
	case authorization.RoleCanvasAdmin:
		return pb.CanvasRole_CANVAS_ADMIN
	case authorization.RoleCanvasOwner:
		return pb.CanvasRole_CANVAS_OWNER
	default:
		return pb.CanvasRole_CANVAS_ROLE_UNSPECIFIED
	}
}

func getRolePermissions(roleName string, domainType pb.DomainType) []*pb.Permission {
	switch domainType {
	case pb.DomainType_DOMAIN_TYPE_ORGANIZATION:
		role := buildOrgRole(roleName, domainType)
		return getAllPermissions(role)
	case pb.DomainType_DOMAIN_TYPE_CANVAS:
		role := buildCanvasRole(roleName, domainType)
		return getAllPermissions(role)
	default:
		return []*pb.Permission{}
	}
}

func getAllPermissions(role *pb.Role) []*pb.Permission {
	permissionMap := make(map[string]*pb.Permission)

	// Add direct permissions
	for _, perm := range role.Permissions {
		key := fmt.Sprintf("%s:%s", perm.Resource, perm.Action)
		permissionMap[key] = perm
	}

	// Add inherited permissions (single level)
	if role.InheritedRole != nil {
		for _, perm := range getAllPermissions(role.InheritedRole) {
			key := fmt.Sprintf("%s:%s", perm.Resource, perm.Action)
			permissionMap[key] = perm
		}
	}

	// Convert map to slice
	permissions := make([]*pb.Permission, 0, len(permissionMap))
	for _, perm := range permissionMap {
		permissions = append(permissions, perm)
	}

	return permissions
}

func buildOrgRole(roleName string, domainType pb.DomainType) *pb.Role {
	role := &pb.Role{
		Name:        roleName,
		DomainType:  domainType,
		Readonly:    true,
		Permissions: []*pb.Permission{},
	}

	switch roleName {
	case authorization.RoleOrgViewer:
		role.Description = "Organization viewer with read-only access to canvases"
		role.Permissions = []*pb.Permission{
			{Resource: "canvas", Action: "read", Description: "View canvases", DomainType: domainType},
		}
		// Base role - no inherited role, but maps to org_admin
		role.MapsTo = buildOrgRole(authorization.RoleOrgAdmin, domainType)
	case authorization.RoleOrgAdmin:
		role.Description = "Organization administrator with full canvas management and user invitation rights"
		role.Permissions = []*pb.Permission{
			{Resource: "canvas", Action: "create", Description: "Create canvases", DomainType: domainType},
			{Resource: "canvas", Action: "update", Description: "Update canvases", DomainType: domainType},
			{Resource: "canvas", Action: "delete", Description: "Delete canvases", DomainType: domainType},
			{Resource: "user", Action: "invite", Description: "Invite users", DomainType: domainType},
			{Resource: "user", Action: "remove", Description: "Remove users", DomainType: domainType},
		}
		// Inherits from org_viewer and maps to org_owner
		role.InheritedRole = buildOrgRole(authorization.RoleOrgViewer, domainType)
		role.MapsTo = buildOrgRole(authorization.RoleOrgOwner, domainType)
	case authorization.RoleOrgOwner:
		role.Description = "Organization owner with full organization management rights"
		role.Permissions = []*pb.Permission{
			{Resource: "org", Action: "update", Description: "Update organization", DomainType: domainType},
			{Resource: "org", Action: "delete", Description: "Delete organization", DomainType: domainType},
		}
		// Inherits from org_admin - no child role
		role.InheritedRole = buildOrgRole(authorization.RoleOrgAdmin, domainType)
	}

	return role
}

// buildCanvasRole builds canvas role definitions with permissions and inheritance
func buildCanvasRole(roleName string, domainType pb.DomainType) *pb.Role {
	role := &pb.Role{
		Name:        roleName,
		DomainType:  domainType,
		Readonly:    true,
		Permissions: []*pb.Permission{},
	}

	switch roleName {
	case authorization.RoleCanvasViewer:
		role.Description = "Canvas viewer with read-only access to canvas resources"
		role.Permissions = []*pb.Permission{
			{Resource: "eventsource", Action: "read", Description: "View event sources", DomainType: domainType},
			{Resource: "stage", Action: "read", Description: "View stages", DomainType: domainType},
			{Resource: "stageevent", Action: "read", Description: "View stage events", DomainType: domainType},
		}
		// Base role - no inherited role, but maps to canvas_admin
		role.MapsTo = buildCanvasRole(authorization.RoleCanvasAdmin, domainType)
	case authorization.RoleCanvasAdmin:
		role.Description = "Canvas administrator with full canvas management and approval rights"
		role.Permissions = []*pb.Permission{
			{Resource: "eventsource", Action: "create", Description: "Create event sources", DomainType: domainType},
			{Resource: "eventsource", Action: "update", Description: "Update event sources", DomainType: domainType},
			{Resource: "eventsource", Action: "delete", Description: "Delete event sources", DomainType: domainType},
			{Resource: "stage", Action: "create", Description: "Create stages", DomainType: domainType},
			{Resource: "stage", Action: "update", Description: "Update stages", DomainType: domainType},
			{Resource: "stage", Action: "delete", Description: "Delete stages", DomainType: domainType},
			{Resource: "stageevent", Action: "approve", Description: "Approve stage events", DomainType: domainType},
			{Resource: "member", Action: "invite", Description: "Invite canvas members", DomainType: domainType},
		}
		role.InheritedRole = buildCanvasRole(authorization.RoleCanvasViewer, domainType)
		role.MapsTo = buildCanvasRole(authorization.RoleCanvasOwner, domainType)
	case authorization.RoleCanvasOwner:
		role.Description = "Canvas owner with full canvas ownership rights"
		role.Permissions = []*pb.Permission{
			{Resource: "member", Action: "remove", Description: "Remove canvas members", DomainType: domainType},
		}
		role.InheritedRole = buildCanvasRole(authorization.RoleCanvasAdmin, domainType)
	}

	return role
}
