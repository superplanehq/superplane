export const organizationKeys = {
  all: ["organization"] as const,
  details: (orgId: string) => [...organizationKeys.all, "details", orgId] as const,
  users: (orgId: string) => [...organizationKeys.all, "users", orgId] as const,
  roles: (orgId: string) => [...organizationKeys.all, "roles", orgId] as const,
  groups: (orgId: string) => [...organizationKeys.all, "groups", orgId] as const,
  group: (orgId: string, groupName: string) => [...organizationKeys.all, "group", orgId, groupName] as const,
  groupUsers: (orgId: string, groupName: string) => [...organizationKeys.all, "groupUsers", orgId, groupName] as const,
  role: (orgId: string, roleName: string) => [...organizationKeys.all, "role", orgId, roleName] as const,
  canvases: (orgId: string) => [...organizationKeys.all, "canvases", orgId] as const,
  inviteLink: (orgId: string) => [...organizationKeys.all, "inviteLink", orgId] as const,
  usage: (orgId: string) => [...organizationKeys.all, "usage", orgId] as const,
};
