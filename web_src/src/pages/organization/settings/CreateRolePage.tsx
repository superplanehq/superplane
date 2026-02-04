import { Heading } from "@/components/Heading/heading";
import { NotFoundPage } from "@/components/NotFoundPage";
import { usePermissions } from "@/contexts/PermissionsContext";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { Description, Label } from "../../../components/Fieldset/fieldset";
import { Input } from "../../../components/Input/input";
import { Text } from "../../../components/Text/text";
import { useCreateRole, useRole, useUpdateRole } from "../../../hooks/useOrganizationData";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/ui/checkbox";
import { showErrorToast } from "@/utils/toast";
import { isCustomComponentsEnabled } from "@/lib/env";

interface Permission {
  id: string;
  name: string;
  description: string;
  category: string;
  resource: string;
  action: string;
}

interface PermissionCategory {
  category: string;
  icon: string;
  permissions: Permission[];
}

// Organization permissions based on RBAC policy
const ORGANIZATION_PERMISSIONS: PermissionCategory[] = [
  {
    category: "General",
    icon: "business",
    permissions: [
      {
        id: "org.read",
        name: "View Organization",
        description: "View organization details and settings",
        category: "General",
        resource: "org",
        action: "read",
      },
      {
        id: "org.update",
        name: "Manage Organization",
        description: "Update organization settings and configuration",
        category: "General",
        resource: "org",
        action: "update",
      },
      {
        id: "org.delete",
        name: "Delete Organization",
        description: "Delete the organization (dangerous)",
        category: "General",
        resource: "org",
        action: "delete",
      },
    ],
  },
  {
    category: "People & Groups",
    icon: "group",
    permissions: [
      {
        id: "member.read",
        name: "View Members",
        description: "View organization members and their details",
        category: "People & Groups",
        resource: "members",
        action: "read",
      },
      {
        id: "member.create",
        name: "Add Members",
        description: "Invite or add members to the organization",
        category: "People & Groups",
        resource: "members",
        action: "create",
      },
      {
        id: "member.update",
        name: "Manage Members",
        description: "Update member roles and permissions",
        category: "People & Groups",
        resource: "members",
        action: "update",
      },
      {
        id: "member.delete",
        name: "Remove Members",
        description: "Remove members from the organization",
        category: "People & Groups",
        resource: "members",
        action: "delete",
      },
      {
        id: "group.read",
        name: "View Groups",
        description: "View organization groups and their members",
        category: "People & Groups",
        resource: "groups",
        action: "read",
      },
      {
        id: "group.create",
        name: "Create Groups",
        description: "Create new groups within the organization",
        category: "People & Groups",
        resource: "groups",
        action: "create",
      },
      {
        id: "group.update",
        name: "Manage Groups",
        description: "Update group settings and membership",
        category: "People & Groups",
        resource: "groups",
        action: "update",
      },
      {
        id: "group.delete",
        name: "Delete Groups",
        description: "Delete groups from the organization",
        category: "People & Groups",
        resource: "groups",
        action: "delete",
      },
    ],
  },
  {
    category: "Roles & Permissions",
    icon: "admin_panel_settings",
    permissions: [
      {
        id: "role.read",
        name: "View Roles",
        description: "View organization roles and their permissions",
        category: "Roles & Permissions",
        resource: "roles",
        action: "read",
      },
      {
        id: "role.create",
        name: "Create Roles",
        description: "Create new roles within the organization",
        category: "Roles & Permissions",
        resource: "roles",
        action: "create",
      },
      {
        id: "role.update",
        name: "Manage Roles",
        description: "Update role permissions and settings",
        category: "Roles & Permissions",
        resource: "roles",
        action: "update",
      },
      {
        id: "role.delete",
        name: "Delete Roles",
        description: "Delete roles from the organization",
        category: "Roles & Permissions",
        resource: "roles",
        action: "delete",
      },
    ],
  },
  {
    category: "Canvases",
    icon: "dashboard",
    permissions: [
      {
        id: "canvas.read",
        name: "View Canvases",
        description: "View organization canvases",
        category: "Canvases",
        resource: "canvases",
        action: "read",
      },
      {
        id: "canvas.create",
        name: "Create Canvases",
        description: "Create new canvases within the organization",
        category: "Canvases",
        resource: "canvases",
        action: "create",
      },
      {
        id: "canvas.update",
        name: "Manage Canvases",
        description: "Update canvas settings and configuration",
        category: "Canvases",
        resource: "canvases",
        action: "update",
      },
      {
        id: "canvas.delete",
        name: "Delete Canvases",
        description: "Delete canvases from the organization",
        category: "Canvases",
        resource: "canvases",
        action: "delete",
      },
    ],
  },
  ...(isCustomComponentsEnabled()
    ? [
        {
          category: "Custom Components",
          icon: "view_module",
          permissions: [
            {
              id: "blueprint.read",
              name: "View Custom Components",
              description: "View organization custom components",
              category: "Custom Components",
              resource: "blueprints",
              action: "read",
            },
            {
              id: "blueprint.create",
              name: "Create Custom Components",
              description: "Create new custom components",
              category: "Custom Components",
              resource: "blueprints",
              action: "create",
            },
            {
              id: "blueprint.update",
              name: "Manage Custom Components",
              description: "Update custom components settings and configuration",
              category: "Custom Components",
              resource: "blueprints",
              action: "update",
            },
            {
              id: "blueprint.delete",
              name: "Delete Custom Components",
              description: "Delete custom components from the organization",
              category: "Custom Components",
              resource: "blueprints",
              action: "delete",
            },
          ],
        },
      ]
    : []),
  {
    category: "Integrations",
    icon: "integration_instructions",
    permissions: [
      {
        id: "integration.read",
        name: "View Integrations",
        description: "View organization integrations",
        category: "Integrations",
        resource: "integrations",
        action: "read",
      },
      {
        id: "integration.create",
        name: "Create Integrations",
        description: "Create new integrations",
        category: "Integrations",
        resource: "integrations",
        action: "create",
      },
      {
        id: "integration.update",
        name: "Manage Integrations",
        description: "Update integration settings and configuration",
        category: "Integrations",
        resource: "integrations",
        action: "update",
      },
      {
        id: "integration.delete",
        name: "Delete Integrations",
        description: "Delete integrations from the organization",
        category: "Integrations",
        resource: "integrations",
        action: "delete",
      },
    ],
  },
  {
    category: "Secrets",
    icon: "lock",
    permissions: [
      {
        id: "secret.read",
        name: "View Secrets",
        description: "View organization secrets",
        category: "Secrets",
        resource: "secrets",
        action: "read",
      },
      {
        id: "secret.create",
        name: "Create Secrets",
        description: "Create new secrets",
        category: "Secrets",
        resource: "secrets",
        action: "create",
      },
      {
        id: "secret.update",
        name: "Manage Secrets",
        description: "Update secrets",
        category: "Secrets",
        resource: "secrets",
        action: "update",
      },
      {
        id: "secret.delete",
        name: "Delete Secrets",
        description: "Delete secrets from the organization",
        category: "Secrets",
        resource: "secrets",
        action: "delete",
      },
    ],
  },
];

const DEFAULT_ROLE_NAMES = ["org_viewer", "org_admin", "org_owner"];

const isDefaultRole = (roleName?: string | null) => {
  if (!roleName) return false;
  return DEFAULT_ROLE_NAMES.includes(roleName);
};

export function CreateRolePage() {
  const { roleName: roleNameParam } = useParams<{ roleName?: string }>();
  const navigate = useNavigate();
  const { organizationId } = useParams<{ organizationId: string }>();
  const orgId = organizationId;
  const isEditMode = !!roleNameParam;
  const { canAct, isLoading: permissionsLoading } = usePermissions();

  const [roleName, setRoleName] = useState("");
  const [selectedPermissions, setSelectedPermissions] = useState<Set<string>>(new Set());

  // React Query hooks
  const { data: existingRole, isLoading, error } = useRole(orgId || "", roleNameParam || "");
  const createRoleMutation = useCreateRole(orgId || "");
  const updateRoleMutation = useUpdateRole(orgId || "");

  const isSubmitting = createRoleMutation.isPending || updateRoleMutation.isPending;
  const isReadOnly = isDefaultRole(roleNameParam);
  const canReadRoles = canAct("roles", "read");
  const canCreateRoles = canAct("roles", "create");
  const canUpdateRoles = canAct("roles", "update");

  usePageTitle([isReadOnly ? "View Role" : isEditMode ? "Edit Role" : "Create Role"]);

  const handleCategoryToggle = (permissions: Permission[]) => {
    if (isReadOnly) return;
    const permissionIds = permissions.map((p) => p.id);
    const allSelected = permissionIds.every((id) => selectedPermissions.has(id));

    setSelectedPermissions((prev) => {
      const newSet = new Set(prev);
      if (allSelected) {
        // Deselect all in category
        permissionIds.forEach((id) => newSet.delete(id));
      } else {
        // Select all in category
        permissionIds.forEach((id) => newSet.add(id));
      }
      return newSet;
    });
  };

  const isCategorySelected = (permissions: Permission[]) => {
    const permissionIds = permissions.map((p) => p.id);
    return permissionIds.every((id) => selectedPermissions.has(id));
  };

  // Load role data when in edit mode
  useEffect(() => {
    if (isEditMode && existingRole) {
      setRoleName(existingRole.spec?.displayName || existingRole.metadata?.name || "");

      // Convert permissions back to selected format
      const permissionIds = new Set<string>();
      existingRole.spec?.permissions?.forEach((perm) => {
        const matchingPerm = ORGANIZATION_PERMISSIONS.flatMap((cat) => cat.permissions).find(
          (p) => p.resource === perm.resource && p.action === perm.action,
        );

        if (matchingPerm) {
          permissionIds.add(matchingPerm.id);
        }
      });
      setSelectedPermissions(permissionIds);
    }
  }, [isEditMode, existingRole]);

  const handleSubmitRole = async () => {
    if (isReadOnly) return;
    if (!roleName.trim() || selectedPermissions.size === 0 || !orgId) return;
    if (isEditMode && !canUpdateRoles) return;
    if (!isEditMode && !canCreateRoles) return;

    try {
      // Convert selected permissions to the protobuf format
      const permissions = Array.from(selectedPermissions).map((permId) => {
        const permission = ORGANIZATION_PERMISSIONS.flatMap((cat) => cat.permissions).find((p) => p.id === permId);

        if (!permission) {
          throw new Error(`Permission ${permId} not found`);
        }

        return {
          resource: permission.resource,
          action: permission.action,
          domainType: "DOMAIN_TYPE_ORGANIZATION" as const,
        };
      });

      if (isEditMode && roleNameParam) {
        // Update existing role
        await updateRoleMutation.mutateAsync({
          roleName: roleNameParam,
          domainType: "DOMAIN_TYPE_ORGANIZATION",
          domainId: orgId,
          permissions: permissions,
          displayName: roleName.trim(),
        });
      } else {
        // Create new role
        await createRoleMutation.mutateAsync({
          role: {
            metadata: {
              name: roleName.toLowerCase().replace(/\s+/g, "_"),
            },
            spec: {
              permissions: permissions,
              displayName: roleName.trim(),
            },
          },
          domainType: "DOMAIN_TYPE_ORGANIZATION",
          domainId: orgId,
        });
      }

      navigate(`/${orgId}/settings/roles`);
    } catch {
      showErrorToast("Failed to create role");
    }
  };

  if (!canReadRoles) {
    return <NotFoundPage />;
  }

  if (isEditMode && !isReadOnly && !canUpdateRoles) {
    return <NotFoundPage />;
  }

  if (!isEditMode && !canCreateRoles) {
    return <NotFoundPage />;
  }

  if (permissionsLoading) {
    return (
      <div className="flex justify-center items-center min-h-[40vh]">
        <p className="text-gray-500">Checking permissions...</p>
      </div>
    );
  }

  return (
    <div className="min-h-screen text-left">
      <div className="max-w-8xl mx-auto py-8">
        {/* Header */}
        <div className="mb-6">
          <div className="flex items-center text-left">
            <div>
              <Heading level={2} className="text-2xl font-medium text-gray-800 dark:text-white mb-2">
                {isReadOnly ? "View Role" : isEditMode ? "Edit Role" : "Create New Role"}
              </Heading>
              {isReadOnly && (
                <Text className="text-sm text-gray-500 dark:text-gray-400">
                  Default roles are read-only and cannot be edited.
                </Text>
              )}
            </div>
          </div>
        </div>

        {/* Role Form */}
        <div className="space-y-6">
          {isLoading ? (
            <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-300 dark:border-gray-800 p-6">
              <div className="flex justify-center items-center h-32">
                <p className="text-gray-500 dark:text-gray-400">Loading role data...</p>
              </div>
            </div>
          ) : (
            <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-300 dark:border-gray-800 p-6">
              {error && (
                <div className="bg-white border border-red-300 text-red-500 px-4 py-2 rounded mb-6">
                  <p className="text-sm">{error instanceof Error ? error.message : "Failed to load role data"}</p>
                </div>
              )}

              <div className="space-y-1">
                {/* Role Name */}
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Role Name *</label>
                  <Input
                    type="text"
                    placeholder={isEditMode ? "Enter role display name" : "Enter role name"}
                    value={roleName}
                    onChange={(e) => setRoleName(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === "Enter" && !e.shiftKey) {
                        e.preventDefault();
                        handleSubmitRole();
                      }
                    }}
                    className="max-w-lg"
                    disabled={isReadOnly}
                  />
                </div>

                {/* Permissions */}
                <div className="pt-4 mb-4">
                  <h2 className="text-base font-semibold text-gray-800 dark:text-white mb-2">
                    Organization Permissions
                  </h2>
                  <Text className="text-sm text-gray-500 dark:text-gray-400">
                    Select the permissions this role should have within the organization.
                  </Text>
                  {isReadOnly && (
                    <Text className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                      Permissions are read-only for default roles.
                    </Text>
                  )}
                </div>

                <div className="space-y-6">
                  {ORGANIZATION_PERMISSIONS.map((category) => (
                    <div key={category.category} className="space-y-4">
                      <div className="flex items-center mb-3">
                        <h3 className="text-md font-semibold text-gray-800 dark:text-white">{category.category}</h3>
                        {!isReadOnly && (
                          <button
                            type="button"
                            className="text-xs font-medium text-gray-500 ml-3 bg-transparent border-none cursor-pointer"
                            onClick={() => handleCategoryToggle(category.permissions)}
                          >
                            {isCategorySelected(category.permissions) ? "Deselect all" : "Select all"}
                          </button>
                        )}
                      </div>
                      <div className="space-y-3">
                        {category.permissions.map((permission) => {
                          const checkboxId = `permission-${permission.id}`;
                          return (
                            <div key={permission.id} className="flex items-start gap-3">
                              <Checkbox
                                id={checkboxId}
                                checked={selectedPermissions.has(permission.id)}
                                disabled={isReadOnly}
                                onCheckedChange={(checked) => {
                                  if (isReadOnly) return;
                                  setSelectedPermissions((prev) => {
                                    const newSet = new Set(prev);
                                    if (checked) {
                                      newSet.add(permission.id);
                                    } else {
                                      newSet.delete(permission.id);
                                    }
                                    return newSet;
                                  });
                                }}
                              />
                              <div className="space-y-1">
                                <Label htmlFor={checkboxId} className={isReadOnly ? "" : "cursor-pointer"}>
                                  {permission.name}
                                </Label>
                                <Description>{permission.description}</Description>
                              </div>
                            </div>
                          );
                        })}
                      </div>
                    </div>
                  ))}
                </div>

                {selectedPermissions.size === 0 && !isReadOnly && (
                  <Text className="text-sm text-red-600 dark:text-red-400 mt-2">
                    Please select at least one permission for this role
                  </Text>
                )}
              </div>
            </div>
          )}

          {/* Action Buttons */}
          <div className="flex justify-start gap-3">
            {!isReadOnly && (
              <Button
                onClick={handleSubmitRole}
                disabled={!roleName.trim() || selectedPermissions.size === 0 || isSubmitting || isLoading}
              >
                {isSubmitting
                  ? isEditMode
                    ? "Updating..."
                    : "Creating..."
                  : isEditMode
                    ? "Update Role"
                    : "Create Role"}
              </Button>
            )}
            <Link to={`/${orgId}/settings/roles`}>
              <Button variant="outline">{isReadOnly ? "Back to Roles" : "Cancel"}</Button>
            </Link>
          </div>
        </div>
      </div>
    </div>
  );
}
