import { useState, useRef } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { MaterialSymbol } from "../../../components/MaterialSymbol/material-symbol";
import { Avatar } from "../../../components/Avatar/avatar";
import { Heading, Subheading } from "../../../components/Heading/heading";
import { Button } from "../../../components/Button/button";
import { Input, InputGroup } from "../../../components/Input/input";
import { Breadcrumbs } from "../../../components/Breadcrumbs/breadcrumbs";
import {
  Dropdown,
  DropdownButton,
  DropdownMenu,
  DropdownItem,
  DropdownLabel,
  DropdownDescription,
} from "../../../components/Dropdown/dropdown";
import { AddMembersSection, AddMembersSectionRef } from "./AddMembersSection";
import { Table, TableHead, TableBody, TableRow, TableHeader, TableCell } from "../../../components/Table/table";
import {
  useOrganizationGroup,
  useOrganizationGroupUsers,
  useOrganizationRoles,
  useUpdateGroup,
  useDeleteGroup,
  useRemoveUserFromGroup,
} from "../../../hooks/useOrganizationData";
import { SuperplaneUsersUser } from "@/api-client";
import { usePageTitle } from "@/hooks/usePageTitle";

export function GroupMembersPage() {
  const { groupName: encodedGroupName } = useParams<{ groupName: string }>();
  const groupName = encodedGroupName ? decodeURIComponent(encodedGroupName) : undefined;
  const navigate = useNavigate();
  const { organizationId } = useParams<{ organizationId: string }>();
  const orgId = organizationId;
  usePageTitle([groupName || "Group", "Members"]);
  const addMembersSectionRef = useRef<AddMembersSectionRef>(null);
  const [searchQuery, setSearchQuery] = useState("");
  const [isEditingGroupName, setIsEditingGroupName] = useState(false);
  const [isEditingGroupDescription, setIsEditingGroupDescription] = useState(false);
  const [editedGroupName, setEditedGroupName] = useState("");
  const [editedGroupDescription, setEditedGroupDescription] = useState("");
  const [sortConfig, setSortConfig] = useState<{
    key: string | null;
    direction: "asc" | "desc";
  }>({
    key: null,
    direction: "asc",
  });

  // React Query hooks
  const {
    data: group,
    isLoading: loadingGroup,
    error: groupError,
    refetch: refetchGroup,
  } = useOrganizationGroup(orgId || "", groupName || "");
  const {
    data: members = [],
    isLoading: loadingMembers,
    error: membersError,
  } = useOrganizationGroupUsers(orgId || "", groupName || "");
  const { data: roles = [], isLoading: loadingRoles, error: rolesError } = useOrganizationRoles(orgId || "");

  // Mutations
  const updateGroupMutation = useUpdateGroup(orgId || "");
  const deleteGroupMutation = useDeleteGroup(orgId || "");
  const removeUserFromGroupMutation = useRemoveUserFromGroup(orgId || "");

  const loading = loadingGroup || loadingMembers;
  const error = groupError || membersError || rolesError;

  const handleBackToGroups = () => {
    navigate(`/${orgId}/settings/groups`);
  };

  const handleEditGroupName = () => {
    if (group) {
      setEditedGroupName(group.spec?.displayName || "");
      setIsEditingGroupName(true);
    }
  };

  const handleSaveGroupName = async () => {
    if (!group || !editedGroupName.trim() || !groupName || !orgId) return;

    try {
      await updateGroupMutation.mutateAsync({
        groupName,
        organizationId: orgId,
        displayName: editedGroupName.trim(),
      });

      // Refetch group data from server to ensure consistency
      await refetchGroup();
      setIsEditingGroupName(false);
    } catch (err) {
      console.error("Error updating group name:", err);
    }
  };

  const handleCancelGroupName = () => {
    setIsEditingGroupName(false);
    setEditedGroupName("");
  };

  const handleEditGroupDescription = () => {
    if (group) {
      setEditedGroupDescription(group.spec?.description || "");
      setIsEditingGroupDescription(true);
    }
  };

  const handleSaveGroupDescription = async () => {
    if (!group || !editedGroupDescription.trim() || !groupName || !orgId) return;

    try {
      await updateGroupMutation.mutateAsync({
        groupName,
        organizationId: orgId,
        description: editedGroupDescription.trim(),
      });

      // Refetch group data from server to ensure consistency
      await refetchGroup();
      setIsEditingGroupDescription(false);
    } catch (err) {
      console.error("Error updating group description:", err);
    }
  };

  const handleCancelGroupDescription = () => {
    setIsEditingGroupDescription(false);
    setEditedGroupDescription("");
  };

  const handleSort = (key: string) => {
    setSortConfig((prevConfig) => ({
      key,
      direction: prevConfig.key === key && prevConfig.direction === "asc" ? "desc" : "asc",
    }));
  };

  const getSortedData = (data: SuperplaneUsersUser[]) => {
    if (!sortConfig.key) return data;

    return [...data].sort((a, b) => {
      const aValue = a[sortConfig.key as keyof SuperplaneUsersUser];
      const bValue = b[sortConfig.key as keyof SuperplaneUsersUser];

      if (aValue && bValue && aValue < bValue) {
        return sortConfig.direction === "asc" ? -1 : 1;
      }
      if (aValue && bValue && aValue > bValue) {
        return sortConfig.direction === "asc" ? 1 : -1;
      }
      return 0;
    });
  };

  const getSortIcon = (columnKey: string) => {
    if (sortConfig.key !== columnKey) {
      return "unfold_more";
    }
    return sortConfig.direction === "asc" ? "keyboard_arrow_up" : "keyboard_arrow_down";
  };

  const handleRemoveMember = async (userId: string) => {
    if (!groupName || !orgId) return;

    try {
      await removeUserFromGroupMutation.mutateAsync({
        groupName,
        userId,
        organizationId: orgId,
      });

      // Trigger refresh of the AddMembersSection to update the "From organization" tab
      addMembersSectionRef.current?.refreshExistingMembers();
    } catch (err) {
      console.error("Error removing member:", err);
    }
  };

  const handleMemberAdded = () => {
    // No need to manually refresh - React Query will handle cache invalidation
  };

  const handleRoleUpdate = async (newRoleName: string) => {
    if (!orgId || !group || !groupName) return;

    try {
      await updateGroupMutation.mutateAsync({
        groupName,
        organizationId: orgId,
        role: newRoleName,
      });
    } catch (err) {
      console.error("Error updating group role:", err);
    }
  };

  const handleDeleteGroup = async () => {
    if (!orgId || !groupName) return;

    const confirmed = window.confirm(
      `Are you sure you want to delete the group "${groupName}"? This action cannot be undone.`,
    );

    if (!confirmed) return;

    try {
      await deleteGroupMutation.mutateAsync({
        groupName,
        organizationId: orgId,
      });

      // Navigate back to groups list after successful deletion
      window.history.back();
    } catch (err) {
      console.error("Error deleting group:", err);
    }
  };

  const filteredMembers = members.filter(
    (member) =>
      member.spec?.displayName?.toLowerCase().includes(searchQuery.toLowerCase()) ||
      member.metadata?.email?.toLowerCase().includes(searchQuery.toLowerCase()),
  );

  if (loading) {
    return (
      <div className="flex justify-center items-center h-screen">
        <p className="text-zinc-500 dark:text-zinc-400">Loading group...</p>
      </div>
    );
  }

  if (error && !group) {
    return (
      <div className="space-y-6 pt-6">
        <div className="mb-4">
          <Breadcrumbs
            items={[
              {
                label: "Groups",
                onClick: handleBackToGroups,
              },
              {
                label: "Group",
                current: true,
              },
            ]}
            showDivider={false}
          />
        </div>
        <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">
          <p>{error instanceof Error ? error.message : "Failed to load group data"}</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6 pt-6">
      {/* Breadcrumbs navigation */}
      <div className="mb-4">
        <Breadcrumbs
          items={[
            {
              label: "Groups",
              onClick: handleBackToGroups,
            },
            {
              label: group?.spec?.displayName || groupName || "Group",
              current: true,
            },
          ]}
          showDivider={false}
        />
      </div>

      <div className="bg-zinc-100 dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 p-6 space-y-6">
        {/* Group header */}
        <div className="flex items-start justify-between">
          <div className="flex items-start gap-3">
            <Avatar
              className="w-12 bg-blue-200 dark:bg-blue-800 border border-blue-300 dark:border-blue-700"
              square
              initials={group?.spec?.displayName?.charAt(0) || "G"}
            />
            <div className="flex flex-col space-y-2">
              {/* Group Name - Inline Edit */}
              <div className="group">
                {isEditingGroupName ? (
                  <div className="flex items-center gap-2">
                    <Input
                      type="text"
                      value={editedGroupName}
                      onChange={(e) => setEditedGroupName(e.target.value)}
                      className="text-2xl font-semibold bg-white dark:bg-zinc-800 border-zinc-300 dark:border-zinc-600"
                      onKeyDown={(e) => {
                        if (e.key === "Enter") handleSaveGroupName();
                        if (e.key === "Escape") handleCancelGroupName();
                      }}
                      autoFocus
                    />
                    <Button
                      plain
                      onClick={handleSaveGroupName}
                      disabled={updateGroupMutation.isPending}
                      className="text-green-600 hover:text-green-700"
                    >
                      <MaterialSymbol name={updateGroupMutation.isPending ? "hourglass_empty" : "check"} size="sm" />
                    </Button>
                    <Button
                      plain
                      onClick={handleCancelGroupName}
                      disabled={updateGroupMutation.isPending}
                      className="text-red-600 hover:text-red-700"
                    >
                      <MaterialSymbol name="close" size="sm" />
                    </Button>
                  </div>
                ) : (
                  <div className="flex items-center gap-2">
                    <Heading level={2} className="text-2xl font-semibold text-zinc-900 dark:text-white">
                      {group?.spec?.displayName}
                    </Heading>
                    <Button
                      plain
                      onClick={handleEditGroupName}
                      className="opacity-0 group-hover:opacity-100 transition-opacity text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-300"
                    >
                      <MaterialSymbol name="edit" size="sm" />
                    </Button>
                  </div>
                )}
              </div>

              {/* Group Description - Inline Edit */}
              <div className="group">
                {isEditingGroupDescription ? (
                  <div className="flex items-center gap-2">
                    <Input
                      type="text"
                      value={editedGroupDescription}
                      onChange={(e) => setEditedGroupDescription(e.target.value)}
                      className="text-lg bg-white dark:bg-zinc-800 border-zinc-300 dark:border-zinc-600"
                      onKeyDown={(e) => {
                        if (e.key === "Enter") handleSaveGroupDescription();
                        if (e.key === "Escape") handleCancelGroupDescription();
                      }}
                      autoFocus
                    />
                    <Button
                      plain
                      onClick={handleSaveGroupDescription}
                      disabled={updateGroupMutation.isPending}
                      className="text-green-600 hover:text-green-700"
                    >
                      <MaterialSymbol name={updateGroupMutation.isPending ? "hourglass_empty" : "check"} size="sm" />
                    </Button>
                    <Button
                      plain
                      onClick={handleCancelGroupDescription}
                      disabled={updateGroupMutation.isPending}
                      className="text-red-600 hover:text-red-700"
                    >
                      <MaterialSymbol name="close" size="sm" />
                    </Button>
                  </div>
                ) : (
                  <div className="flex items-center gap-2">
                    <Subheading level={3} className="text-lg !font-normal text-zinc-600 dark:text-zinc-400">
                      {group?.spec?.description || "No description"}
                    </Subheading>
                    <Button
                      plain
                      onClick={handleEditGroupDescription}
                      className="opacity-0 group-hover:opacity-100 transition-opacity text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-300"
                    >
                      <MaterialSymbol name="edit" size="sm" />
                    </Button>
                  </div>
                )}
              </div>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Dropdown>
              <DropdownButton color="white" className="flex items-center gap-2 text-sm" disabled={loadingRoles}>
                {loadingRoles
                  ? "Loading..."
                  : roles.find((role) => role.metadata?.name === group?.spec?.role)?.spec?.displayName || "Member"}
                <MaterialSymbol name="keyboard_arrow_down" />
              </DropdownButton>
              <DropdownMenu>
                {roles.map((role) => (
                  <DropdownItem key={role.metadata?.name} onClick={() => handleRoleUpdate(role.metadata!.name!)}>
                    <DropdownLabel>{role.spec?.displayName}</DropdownLabel>
                    <DropdownDescription>{role.spec?.description}</DropdownDescription>
                  </DropdownItem>
                ))}
              </DropdownMenu>
            </Dropdown>
            <Dropdown>
              <DropdownButton plain aria-label="More options" disabled={deleteGroupMutation.isPending}>
                <MaterialSymbol name="more_vert" size="sm" />
              </DropdownButton>
              <DropdownMenu>
                <DropdownItem onClick={handleDeleteGroup} className="text-red-600 dark:text-red-400">
                  <MaterialSymbol name="delete" />
                  {deleteGroupMutation.isPending ? "Deleting..." : "Delete group"}
                </DropdownItem>
              </DropdownMenu>
            </Dropdown>
          </div>
        </div>

        {/* Add Members Section */}
        <AddMembersSection
          ref={addMembersSectionRef}
          organizationId={orgId!}
          groupName={groupName!}
          showRoleSelection={false}
          onMemberAdded={handleMemberAdded}
        />

        {/* Group members table */}
        <div className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 overflow-hidden">
          <div className="px-6 pt-6 pb-4">
            <div className="flex items-center justify-between">
              <InputGroup>
                <Input
                  name="search"
                  placeholder="Search team members…"
                  aria-label="Search"
                  className="w-xs"
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                />
              </InputGroup>
            </div>
          </div>
          <div className="px-6 pb-6">
            <Table dense>
              <TableHead>
                <TableRow>
                  <TableHeader
                    className="cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-700/50"
                    onClick={() => handleSort("name")}
                  >
                    <div className="flex items-center gap-2">
                      Name
                      <MaterialSymbol name={getSortIcon("name")} size="sm" className="text-zinc-400" />
                    </div>
                  </TableHeader>
                  <TableHeader
                    className="cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-700/50"
                    onClick={() => handleSort("email")}
                  >
                    <div className="flex items-center gap-2">
                      Email
                      <MaterialSymbol name={getSortIcon("email")} size="sm" className="text-zinc-400" />
                    </div>
                  </TableHeader>
                  <TableHeader></TableHeader>
                </TableRow>
              </TableHead>
              <TableBody>
                {getSortedData(filteredMembers).map((member) => (
                  <TableRow key={member.metadata?.id}>
                    <TableCell>
                      <div className="flex items-center gap-3">
                        <Avatar
                          src={member.spec?.accountProviders?.[0]?.avatarUrl}
                          initials={member.spec?.displayName?.charAt(0) || "U"}
                          className="size-8"
                        />
                        <div>
                          <div className="text-sm font-medium text-zinc-900 dark:text-white">
                            {member.spec?.displayName}
                          </div>
                          <div className="text-xs text-zinc-500 dark:text-zinc-400">
                            Member since {new Date().toLocaleDateString()}
                          </div>
                        </div>
                      </div>
                    </TableCell>
                    <TableCell>{member.metadata?.email}</TableCell>
                    <TableCell>
                      <div className="flex justify-end">
                        <Dropdown>
                          <DropdownButton plain className="flex items-center gap-2 text-sm">
                            <MaterialSymbol name="more_vert" size="sm" />
                          </DropdownButton>
                          <DropdownMenu>
                            <DropdownItem onClick={() => handleRemoveMember(member.metadata!.id!)}>
                              <MaterialSymbol name="person_remove" />
                              Remove from Group
                            </DropdownItem>
                          </DropdownMenu>
                        </Dropdown>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
                {filteredMembers.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={4} className="text-center h-[200px] py-6">
                      {searchQuery ? `No members found matching "${searchQuery}"` : "No group members yet"}
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </div>
        </div>
      </div>
    </div>
  );
}
