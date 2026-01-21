import { useMemo } from "react";
import { useNavigate } from "react-router-dom";
import { RolesRole } from "../../../api-client/types.gen";
import { Icon } from "../../../components/Icon";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "../../../components/Table/table";
import { useDeleteRole, useOrganizationRoles } from "../../../hooks/useOrganizationData";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";

interface RolesProps {
  organizationId: string;
}

export function Roles({ organizationId }: RolesProps) {
  const navigate = useNavigate();
  // Use React Query hooks for data fetching
  const { data: roles = [], isLoading: loadingRoles, error } = useOrganizationRoles(organizationId);

  // Mutation for role deletion
  const deleteRoleMutation = useDeleteRole(organizationId);

  const handleCreateRole = () => {
    navigate(`/${organizationId}/settings/create-role`);
  };

  const handleEditRole = (role: RolesRole) => {
    navigate(`/${organizationId}/settings/create-role/${role.metadata?.name}`);
  };

  const handleDeleteRole = async (role: RolesRole) => {
    if (!role.metadata?.name) return;

    const confirmed = window.confirm(
      `Are you sure you want to delete the role "${role.metadata?.name}"? This cannot be undone.`,
    );

    if (!confirmed) return;

    try {
      await deleteRoleMutation.mutateAsync({
        roleName: role.metadata?.name,
        domainType: "DOMAIN_TYPE_ORGANIZATION",
        domainId: organizationId,
      });
    } catch (err) {
      console.error("Error deleting role:", err);
    }
  };

  const getSortedData = (data: RolesRole[]) => {
    const defaultOrder = ["org_admin", "org_owner", "org_viewer"];
    const defaultOrderIndex = new Map(defaultOrder.map((role, index) => [role, index]));
    const defaultRoles: RolesRole[] = [];
    const customRoles: RolesRole[] = [];

    data.forEach((role) => {
      if (isDefaultRole(role.metadata?.name)) {
        defaultRoles.push(role);
      } else {
        customRoles.push(role);
      }
    });

    const sortedCustomRoles = [...customRoles].sort((a, b) => {
      const aValue = (a.spec?.displayName || a.metadata?.name || "").toLowerCase();
      const bValue = (b.spec?.displayName || b.metadata?.name || "").toLowerCase();
      return aValue.localeCompare(bValue);
    });

    const sortedDefaultRoles = [...defaultRoles].sort((a, b) => {
      const aIndex = defaultOrderIndex.get(a.metadata?.name || "") ?? Number.MAX_SAFE_INTEGER;
      const bIndex = defaultOrderIndex.get(b.metadata?.name || "") ?? Number.MAX_SAFE_INTEGER;
      return aIndex - bIndex;
    });

    return [...sortedCustomRoles, ...sortedDefaultRoles];
  };

  const isDefaultRole = (roleName: string | undefined) => {
    if (!roleName) return false;
    const defaultRoles = ["org_viewer", "org_admin", "org_owner"];
    return defaultRoles.includes(roleName);
  };

  const filteredAndSortedRoles = useMemo(() => {
    return getSortedData(roles);
  }, [roles]);

  return (
    <div className="space-y-6 pt-6">
      {error && (
        <div className="bg-white border border-red-300 text-red-500 px-4 py-2 rounded">
          <p>{error instanceof Error ? error.message : "Failed to fetch roles"}</p>
        </div>
      )}

      <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-300 dark:border-gray-800 overflow-hidden">
        <div className="px-6 pt-6 pb-4 flex items-center justify-start">
          <Button className="flex items-center" onClick={handleCreateRole}>
            <Icon name="plus" />
            New Organization Role
          </Button>
        </div>
        <div className="px-6 pb-6">
          {loadingRoles ? (
            <div className="flex justify-center items-center h-32">
              <p className="text-gray-500 dark:text-gray-400">Loading roles...</p>
            </div>
          ) : (
            <Table dense>
              <TableHead>
                <TableRow>
                  <TableHeader>Role name</TableHeader>
                  <TableHeader>Permissions</TableHeader>
                  <TableHeader></TableHeader>
                </TableRow>
              </TableHead>
              <TableBody>
                {filteredAndSortedRoles.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={3} className="text-center py-8 text-gray-500 dark:text-gray-400">
                      No roles found
                    </TableCell>
                  </TableRow>
                ) : (
                  filteredAndSortedRoles.map((role, index) => {
                    const isDefault = isDefaultRole(role.metadata?.name);
                    return (
                      <TableRow key={role.metadata?.name || index} className="last:[&>td]:border-b-0">
                        <TableCell className="font-semibold">{role.spec?.displayName || role.metadata?.name}</TableCell>
                        <TableCell>{role.spec?.permissions?.length || 0}</TableCell>
                        <TableCell>
                          <div className="flex justify-end">
                            {isDefault ? (
                              <span className="text-xs text-gray-700 dark:text-gray-400 px-2 py-1 bg-gray-200 dark:bg-gray-800 rounded">
                                Default Role
                              </span>
                            ) : (
                              <TooltipProvider delayDuration={200}>
                                <div className="flex items-center gap-1">
                                  <Tooltip>
                                    <TooltipTrigger asChild>
                                      <button
                                        type="button"
                                        onClick={() => handleEditRole(role)}
                                        className="p-1 rounded-sm text-gray-800 hover:bg-gray-100 transition-colors dark:text-gray-100 dark:hover:bg-gray-800"
                                        aria-label="Edit role"
                                      >
                                        <Icon name="edit" size="sm" />
                                      </button>
                                    </TooltipTrigger>
                                    <TooltipContent side="top">Edit Role</TooltipContent>
                                  </Tooltip>
                                  <Tooltip>
                                    <TooltipTrigger asChild>
                                      <button
                                        type="button"
                                        onClick={() => handleDeleteRole(role)}
                                        className="p-1 rounded-sm text-gray-800 hover:bg-gray-100 transition-colors dark:text-gray-100 dark:hover:bg-gray-800"
                                        aria-label="Delete role"
                                        disabled={deleteRoleMutation.isPending}
                                      >
                                        <Icon name="trash-2" size="sm" />
                                      </button>
                                    </TooltipTrigger>
                                    <TooltipContent side="top">
                                      {deleteRoleMutation.isPending ? "Deleting..." : "Delete Role"}
                                    </TooltipContent>
                                  </Tooltip>
                                </div>
                              </TooltipProvider>
                            )}
                          </div>
                        </TableCell>
                      </TableRow>
                    );
                  })
                )}
              </TableBody>
            </Table>
          )}
        </div>
      </div>
    </div>
  );
}
