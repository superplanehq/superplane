import { formatRelativeTime } from "@/utils/timezone";
import debounce from "lodash.debounce";
import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Avatar } from "../../../components/Avatar/avatar";
import {
  Dropdown,
  DropdownButton,
  DropdownDescription,
  DropdownItem,
  DropdownLabel,
  DropdownMenu,
} from "../../../components/Dropdown/dropdown";
import { Heading } from "../../../components/Heading/heading";
import { Icon } from "../../../components/Icon";
import { Input, InputGroup } from "../../../components/Input/input";
import { Link } from "../../../components/Link/link";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "../../../components/Table/table";
import {
  useDeleteGroup,
  useOrganizationGroups,
  useOrganizationRoles,
  useUpdateGroup,
} from "../../../hooks/useOrganizationData";
import { Button } from "../../../ui/button";

interface GroupsProps {
  organizationId: string;
}

export function Groups({ organizationId }: GroupsProps) {
  const navigate = useNavigate();
  const [search, setSearch] = useState("");
  const [sortConfig, setSortConfig] = useState<{
    key: string | null;
    direction: "asc" | "desc";
  }>({
    key: null,
    direction: "asc",
  });

  const setDebouncedSearch = debounce(setSearch, 300);

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

  const handleViewMembers = (groupName: string) => {
    navigate(`/${organizationId}/settings/groups/${groupName}/members`);
  };

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
    const filtered = groups.filter((group) => {
      if (search === "") {
        return true;
      }
      return group.metadata?.name?.toLowerCase().includes(search.toLowerCase());
    });

    if (!sortConfig.key) return filtered;

    return [...filtered].sort((a, b) => {
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
  }, [groups, search, sortConfig]);

  return (
    <div className="space-y-6 pt-6">
      <div className="flex items-center justify-between">
        <Heading level={2} className="text-2xl font-semibold text-gray-800 dark:text-white">
          Groups
        </Heading>
      </div>

      {error && (
        <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">
          <p>{error instanceof Error ? error.message : "Failed to fetch data"}</p>
        </div>
      )}

      <div className="bg-white dark:bg-gray-950 rounded-lg border border-gray-200 dark:border-gray-800 overflow-hidden">
        <div className="px-6 pt-6 pb-4 flex items-center justify-between">
          <InputGroup>
            <Input
              name="search"
              placeholder="Search Groups…"
              aria-label="Search"
              className="w-xs"
              onChange={(e) => setDebouncedSearch(e.target.value)}
            />
          </InputGroup>
          <Button className="flex items-center" onClick={handleCreateGroup}>
            <Icon name="plus" />
            Create New Group
          </Button>
        </div>
        <div className="px-6 pb-6">
          {loadingGroups ? (
            <div className="flex justify-center items-center h-32">
              <p className="text-gray-500 dark:text-gray-400">Loading groups...</p>
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
                {filteredAndSortedGroups.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={5} className="text-center py-8 text-gray-500 dark:text-gray-400">
                      No groups found
                    </TableCell>
                  </TableRow>
                ) : (
                  filteredAndSortedGroups.map((group, index) => (
                    <TableRow key={index}>
                      <TableCell>
                        <div className="flex items-center gap-3">
                          <Avatar
                            className="w-9"
                            square
                            initials={group.spec?.displayName?.charAt(0).toUpperCase() || "G"}
                          />
                          <div>
                            <Link
                              href={`/${organizationId}/settings/groups/${group.metadata?.name}/members`}
                              className="cursor-pointer text-sm font-medium text-blue-600 dark:text-blue-400"
                            >
                              {group.spec?.displayName}
                            </Link>
                            <p className="text-xs text-gray-500 dark:text-gray-400">{group.spec?.description || ""}</p>
                          </div>
                        </div>
                      </TableCell>

                      <TableCell>
                        <span className="text-sm text-gray-600 dark:text-gray-400">
                          {formatRelativeTime(group.metadata?.createdAt)}
                        </span>
                      </TableCell>
                      <TableCell>
                        <span className="text-sm text-gray-600 dark:text-gray-400">
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
                          <Dropdown>
                            <DropdownButton disabled={deleteGroupMutation.isPending}>
                              <Icon name="ellipsis-vertical" size="sm" />
                            </DropdownButton>
                            <DropdownMenu>
                              <DropdownItem onClick={() => handleViewMembers(group.metadata!.name!)}>
                                <Icon name="group" />
                                View Members
                              </DropdownItem>
                              <DropdownItem
                                onClick={() => handleDeleteGroup(group.metadata!.name!)}
                                className="text-red-600 dark:text-red-400"
                              >
                                <Icon name="delete" />
                                {deleteGroupMutation.isPending ? "Deleting..." : "Delete Group"}
                              </DropdownItem>
                            </DropdownMenu>
                          </Dropdown>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          )}
        </div>
      </div>
    </div>
  );
}
