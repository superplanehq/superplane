import { formatRelativeTime } from "@/utils/timezone";
import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  Dropdown,
  DropdownButton,
  DropdownDescription,
  DropdownItem,
  DropdownLabel,
  DropdownMenu,
} from "../../../components/Dropdown/dropdown";
import { Icon } from "../../../components/Icon";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { Link } from "../../../components/Link/link";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "../../../components/Table/table";
import {
  useDeleteGroup,
  useOrganizationGroups,
  useOrganizationRoles,
  useUpdateGroup,
} from "../../../hooks/useOrganizationData";
import { Button } from "@/components/ui/button";

interface GroupsProps {
  organizationId: string;
}

export function Groups({ organizationId }: GroupsProps) {
  const navigate = useNavigate();
  const [sortConfig, setSortConfig] = useState<{
    key: string | null;
    direction: "asc" | "desc";
  }>({
    key: null,
    direction: "asc",
  });

  // Use React Query hooks for data fetching
  const { data: groups = [], isLoading: loadingGroups, error: groupsError } = useOrganizationGroups(organizationId);
  const { data: roles = [], error: rolesError } = useOrganizationRoles(organizationId);

  // Mutations
  const updateGroupMutation = useUpdateGroup(organizationId);
  const deleteGroupMutation = useDeleteGroup(organizationId);

  const error = groupsError || rolesError;

  const handleCreateGroup = () => {
    navigate(`/${organizationId}/settings/create-group`);
  };

  const getGroupMembersPath = (groupName: string) =>
    `/${organizationId}/settings/groups/${encodeURIComponent(groupName)}/members`;

  const handleDeleteGroup = async (groupName: string) => {
    const confirmed = window.confirm(
      `Are you sure you want to delete the group "${groupName}"? This action cannot be undone.`,
    );

    if (!confirmed) return;

    try {
      await deleteGroupMutation.mutateAsync({
        groupName,
        organizationId,
      });
    } catch (err) {
      console.error("Error deleting group:", err);
    }
  };

  const handleRoleUpdate = async (groupName: string, newRoleName: string) => {
    try {
      await updateGroupMutation.mutateAsync({
        groupName,
        organizationId,
        role: newRoleName,
      });
    } catch (err) {
      console.error("Error updating group role:", err);
    }
  };

  const handleSort = (key: string) => {
    setSortConfig((prevConfig) => ({
      key,
      direction: prevConfig.key === key && prevConfig.direction === "asc" ? "desc" : "asc",
    }));
  };

  const getSortIcon = (columnKey: string) => {
    if (sortConfig.key !== columnKey) {
      return "chevrons-up-down";
    }
    return sortConfig.direction === "asc" ? "chevron-up" : "chevron-down";
  };

  const filteredAndSortedGroups = useMemo(() => {
    const source = groups;

    if (!sortConfig.key) return source;

    return [...source].sort((a, b) => {
      let aValue: string | number;
      let bValue: string | number;

      switch (sortConfig.key) {
        case "name":
          aValue = (a.metadata?.name || "").toLowerCase();
          bValue = (b.metadata?.name || "").toLowerCase();
          break;
        case "role":
          aValue = (a.spec?.role || "").toLowerCase();
          bValue = (b.spec?.role || "").toLowerCase();
          break;
        case "created":
          aValue = a.metadata?.createdAt ? new Date(a.metadata.createdAt).getTime() : 0;
          bValue = b.metadata?.createdAt ? new Date(b.metadata.createdAt).getTime() : 0;
          break;
        case "members":
          aValue = a.status?.membersCount || 0;
          bValue = b.status?.membersCount || 0;
          break;
        default:
          return 0;
      }

      if (aValue < bValue) {
        return sortConfig.direction === "asc" ? -1 : 1;
      }
      if (aValue > bValue) {
        return sortConfig.direction === "asc" ? 1 : -1;
      }
      return 0;
    });
  }, [groups, sortConfig]);

  return (
    <div className="space-y-6 pt-6">
      {error && (
        <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">
          <p>{error instanceof Error ? error.message : "Failed to fetch data"}</p>
        </div>
      )}

      <div className="bg-white dark:bg-gray-950 rounded-lg border border-gray-300 dark:border-gray-800 overflow-hidden">
        {filteredAndSortedGroups.length > 0 && (
          <div className="px-6 pt-6 pb-4 flex items-center justify-start">
            <Button className="flex items-center" onClick={handleCreateGroup}>
              <Icon name="plus" />
              Create New Group
            </Button>
          </div>
        )}
        <div className="px-6 pb-6 min-h-96">
          {loadingGroups ? (
            <div className="flex justify-center items-center h-32">
              <p className="text-gray-500 dark:text-gray-400">Loading groups...</p>
            </div>
          ) : filteredAndSortedGroups.length === 0 ? (
            <div className="flex min-h-96 flex-col items-center justify-center text-center">
              <div className="flex items-center justify-center text-gray-800">
                <Icon name="users" size="xl" />
              </div>
              <p className="mt-3 text-sm text-gray-800">Create your first group</p>
              <Button className="mt-4 flex items-center" onClick={handleCreateGroup}>
                <Icon name="plus" />
                Create New Group
              </Button>
            </div>
          ) : (
            <Table dense>
              <TableHead>
                <TableRow>
                  <TableHeader
                    className="cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-700/50"
                    onClick={() => handleSort("name")}
                  >
                    <div className="flex items-center gap-2">
                      Team name
                      <Icon name={getSortIcon("name")} size="sm" className="text-gray-400" />
                    </div>
                  </TableHeader>
                  <TableHeader
                    className="cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-700/50"
                    onClick={() => handleSort("created")}
                  >
                    <div className="flex items-center gap-2">
                      Created
                      <Icon name={getSortIcon("created")} size="sm" className="text-gray-400" />
                    </div>
                  </TableHeader>
                  <TableHeader
                    className="cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-700/50"
                    onClick={() => handleSort("members")}
                  >
                    <div className="flex items-center gap-2">
                      Members
                      <Icon name={getSortIcon("members")} size="sm" className="text-gray-400" />
                    </div>
                  </TableHeader>
                  <TableHeader
                    className="cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-700/50"
                    onClick={() => handleSort("role")}
                  >
                    <div className="flex items-center gap-2">
                      Role
                      <Icon name={getSortIcon("role")} size="sm" className="text-gray-400" />
                    </div>
                  </TableHeader>
                  <TableHeader></TableHeader>
                </TableRow>
              </TableHead>
              <TableBody>
                {filteredAndSortedGroups.map((group, index) => (
                  <TableRow key={index} className="last:[&>td]:border-b-0">
                    <TableCell>
                      <div className="flex items-center gap-2">
                        <Icon name="users" size="sm" className="text-gray-800" />
                        <Link
                          href={group.metadata?.name ? getGroupMembersPath(group.metadata.name) : "#"}
                          className="cursor-pointer text-sm !font-semibold text-gray-800 !underline underline-offset-2"
                        >
                          {group.spec?.displayName}
                        </Link>
                      </div>
                    </TableCell>

                    <TableCell>
                      <span className="text-sm text-gray-500 dark:text-gray-400">
                        {formatRelativeTime(group.metadata?.createdAt)}
                      </span>
                    </TableCell>
                    <TableCell>
                      <span className="text-sm text-gray-500 dark:text-gray-400">
                        {group.status?.membersCount || 0} member{group.status?.membersCount === 1 ? "" : "s"}
                      </span>
                    </TableCell>
                    <TableCell>
                      <Dropdown>
                        <DropdownButton
                          className="flex items-center gap-2 text-sm justify-between"
                          disabled={updateGroupMutation.isPending}
                        >
                          {updateGroupMutation.isPending
                            ? "Updating..."
                            : roles.find((r) => r?.metadata?.name === group.spec?.role)?.spec?.displayName ||
                              "Select Role"}
                          <Icon name="chevron-down" />
                        </DropdownButton>
                        <DropdownMenu>
                          {roles.map((role) => (
                            <DropdownItem
                              key={role.metadata?.name}
                              onClick={() => handleRoleUpdate(group.metadata!.name!, role.metadata!.name!)}
                            >
                              <DropdownLabel>{role.spec?.displayName || role.metadata!.name}</DropdownLabel>
                              <DropdownDescription>{role.spec?.description || ""}</DropdownDescription>
                            </DropdownItem>
                          ))}
                        </DropdownMenu>
                      </Dropdown>
                    </TableCell>
                    <TableCell>
                      <div className="flex justify-end">
                        <TooltipProvider delayDuration={200}>
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <button
                                type="button"
                                onClick={() => handleDeleteGroup(group.metadata!.name!)}
                                className="p-1 rounded-sm text-gray-800 hover:bg-gray-100 transition-colors"
                                aria-label="Delete group"
                                disabled={deleteGroupMutation.isPending}
                              >
                                <Icon name="trash-2" size="sm" />
                              </button>
                            </TooltipTrigger>
                            <TooltipContent side="top">Delete Group</TooltipContent>
                          </Tooltip>
                        </TooltipProvider>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </div>
      </div>
    </div>
  );
}
