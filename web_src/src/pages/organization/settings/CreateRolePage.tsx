import { Heading } from "@/components/Heading/heading";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { Description, Label } from "../../../components/Fieldset/fieldset";
import { Input } from "../../../components/Input/input";
import { Text } from "../../../components/Text/text";
import { useCreateRole, useRole, useUpdateRole } from "../../../hooks/useOrganizationData";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/ui/checkbox";

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
        id: "user.read",
        name: "View Members",
        description: "View organization members and their details",
        category: "People & Groups",
        resource: "user",
        action: "read",
      },
      {
        id: "user.invite",
        name: "Invite Members",
        description: "Invite new members to the organization",
        category: "People & Groups",
        resource: "user",
        action: "invite",
      },
      {
        id: "user.remove",
        name: "Remove Members",
        description: "Remove members from the organization",
        category: "People & Groups",
        resource: "user",
        action: "remove",
      },
      {
        id: "group.read",
        name: "View Groups",
        description: "View organization groups and their members",
        category: "People & Groups",
        resource: "group",
        action: "read",
      },
      {
        id: "group.create",
        name: "Create Groups",
        description: "Create new groups within the organization",
        category: "People & Groups",
        resource: "group",
        action: "create",
      },
      {
        id: "group.update",
        name: "Manage Groups",
        description: "Update group settings and membership",
        category: "People & Groups",
        resource: "group",
        action: "update",
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
        resource: "role",
        action: "read",
      },
      {
        id: "role.create",
        name: "Create Roles",
        description: "Create new roles within the organization",
        category: "Roles & Permissions",
        resource: "role",
        action: "create",
      },
      {
        id: "role.update",
        name: "Manage Roles",
        description: "Update role permissions and settings",
        category: "Roles & Permissions",
        resource: "role",
        action: "update",
      },
      {
        id: "role.delete",
        name: "Delete Roles",
        description: "Delete roles from the organization",
        category: "Roles & Permissions",
        resource: "role",
        action: "delete",
      },
      {
        id: "role.assign",
        name: "Assign Roles",
        description: "Assign roles to users and groups",
        category: "Roles & Permissions",
        resource: "role",
        action: "assign",
      },
      {
        id: "role.remove",
        name: "Remove Roles",
        description: "Remove roles from users and groups",
        category: "Roles & Permissions",
        resource: "role",
        action: "remove",
      },
    ],
  },
  {
    category: "Projects & Resources",
    icon: "folder",
    permissions: [
      {
        id: "canvas.read",
        name: "View Canvases",
        description: "View organization canvases",
        category: "Projects & Resources",
        resource: "canvas",
        action: "read",
      },
      {
        id: "canvas.create",
        name: "Create Canvases",
        description: "Create new canvases within the organization",
        category: "Projects & Resources",
        resource: "canvas",
        action: "create",
      },
      {
        id: "canvas.update",
        name: "Manage Canvases",
        description: "Update canvas settings and configuration",
        category: "Projects & Resources",
        resource: "canvas",
        action: "update",
      },
      {
        id: "canvas.delete",
        name: "Delete Canvases",
        description: "Delete canvases from the organization",
        category: "Projects & Resources",
        resource: "canvas",
        action: "delete",
      },
    ],
  },
];

export function CreateRolePage() {
  const { roleName: roleNameParam } = useParams<{ roleName?: string }>();
  const navigate = useNavigate();
  const { organizationId } = useParams<{ organizationId: string }>();
  const orgId = organizationId;
  const isEditMode = !!roleNameParam;
  usePageTitle([isEditMode ? "Edit Role" : "Create Role"]);

  const [roleName, setRoleName] = useState("");
  const [selectedPermissions, setSelectedPermissions] = useState<Set<string>>(new Set());

  // React Query hooks
  const { data: existingRole, isLoading, error } = useRole(orgId || "", roleNameParam || "");
  const createRoleMutation = useCreateRole(orgId || "");
  const updateRoleMutation = useUpdateRole(orgId || "");

  const isSubmitting = createRoleMutation.isPending || updateRoleMutation.isPending;

  const handleCategoryToggle = (permissions: Permission[]) => {
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
    if (!roleName.trim() || selectedPermissions.size === 0 || !orgId) return;

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
              name: roleName,
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
      console.error("Failed to create role");
    }
  };

  return (
    <div className="min-h-screen text-left">
      <div className="max-w-8xl mx-auto py-8">
        {/* Header */}
        <div className="mb-6">
          <div className="flex items-center text-left">
            <div>
              <Heading level={2} className="text-2xl font-medium text-gray-800 dark:text-white mb-2">
                {isEditMode ? "Edit Role" : "Create New Role"}
              </Heading>
            </div>
          </div>
        </div>

        {/* Role Form */}
        <div className="space-y-6">
          {isLoading ? (
            <div className="bg-white dark:bg-neutral-800 rounded-lg border border-gray-300 dark:border-neutral-700 p-6">
              <div className="flex justify-center items-center h-32">
                <p className="text-gray-500 dark:text-gray-400">Loading role data...</p>
              </div>
            </div>
          ) : (
            <div className="bg-white dark:bg-neutral-800 rounded-lg border border-gray-300 dark:border-neutral-700 p-6">
              {error && (
                <div className="bg-white border border-red-300 text-red-500 px-4 py-2 rounded mb-6">
                  <p className="text-sm">{error instanceof Error ? error.message : "Failed to load role data"}</p>
                </div>
              )}

              <div className="space-y-6">
                {/* Role Name */}
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Role Name *</label>
                  <Input
                    type="text"
                    placeholder="Enter role name"
                    value={roleName}
                    onChange={(e) => setRoleName(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === "Enter" && !e.shiftKey) {
                        e.preventDefault();
                        handleSubmitRole();
                      }
                    }}
                    className="max-w-lg"
                    disabled={isEditMode}
                  />
                  {isEditMode && (
                    <Text className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                      Role name cannot be changed when editing
                    </Text>
                  )}
                </div>

                {/* Permissions */}
                <div className="pt-4 mb-4">
                  <h2 className="text-base font-semibold text-gray-800 dark:text-white mb-2">
                    Organization Permissions
                  </h2>
                  <Text className="text-sm text-gray-500 dark:text-gray-400">
                    Select the permissions this role should have within the organization.
                  </Text>
                </div>

                <div className="space-y-6">
                  {ORGANIZATION_PERMISSIONS.map((category) => (
                    <div key={category.category} className="space-y-4">
                      <div className="flex items-center mb-3">
                        <h3 className="text-md font-semibold text-gray-800 dark:text-white">{category.category}</h3>
                        <button
                          type="button"
                          className="text-xs font-medium text-gray-500 ml-3 bg-transparent border-none cursor-pointer"
                          onClick={() => handleCategoryToggle(category.permissions)}
                        >
                          {isCategorySelected(category.permissions) ? "Deselect all" : "Select all"}
                        </button>
                      </div>
                      <div className="space-y-3">
                        {category.permissions.map((permission) => {
                          const checkboxId = `permission-${permission.id}`;
                          return (
                            <div key={permission.id} className="flex items-start gap-3">
                              <Checkbox
                                id={checkboxId}
                                checked={selectedPermissions.has(permission.id)}
                                onCheckedChange={(checked) => {
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
                                <Label htmlFor={checkboxId} className="cursor-pointer">
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

                {selectedPermissions.size === 0 && (
                  <Text className="text-sm text-red-600 dark:text-red-400 mt-2">
                    Please select at least one permission for this role
                  </Text>
                )}
              </div>
            </div>
          )}

          {/* Action Buttons */}
          <div className="flex justify-start gap-3">
            <Button
              onClick={handleSubmitRole}
              disabled={!roleName.trim() || selectedPermissions.size === 0 || isSubmitting || isLoading}
            >
              {isSubmitting ? (isEditMode ? "Updating..." : "Creating...") : isEditMode ? "Update Role" : "Create Role"}
            </Button>
            <Link to={`/${orgId}/settings/roles`}>
              <Button variant="outline">Cancel</Button>
            </Link>
          </div>
        </div>
      </div>
    </div>
  );
}
