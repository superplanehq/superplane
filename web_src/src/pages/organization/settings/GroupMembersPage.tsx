import { usePageTitle } from "@/hooks/usePageTitle";
import { useRef, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { Avatar } from "../../../components/Avatar/avatar";
import { Breadcrumbs } from "../../../components/Breadcrumbs/breadcrumbs";
import { Heading } from "../../../components/Heading/heading";
import { Icon } from "../../../components/Icon";
import { Input } from "../../../components/Input/input";
import { Table, TableBody, TableCell, TableRow } from "../../../components/Table/table";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { PermissionTooltip } from "@/components/PermissionGate";
import { usePermissions } from "@/contexts/PermissionsContext";
import {
  useOrganizationGroup,
  useOrganizationGroupUsers,
  useRemoveUserFromGroup,
  useUpdateGroup,
  organizationKeys,
} from "../../../hooks/useOrganizationData";
import { Button } from "@/components/ui/button";
import { AddMembersSection, AddMembersSectionRef } from "./AddMembersSection";
import { showErrorToast } from "@/utils/toast";

export function GroupMembersPage() {
  const { groupName: encodedGroupName } = useParams<{ groupName: string }>();
  const groupName = encodedGroupName ? decodeURIComponent(encodedGroupName) : undefined;
  const navigate = useNavigate();
  const { organizationId } = useParams<{ organizationId: string }>();
  const orgId = organizationId;
  const queryClient = useQueryClient();
  usePageTitle(["Groups", groupName || "Group"]);
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const addMembersSectionRef = useRef<AddMembersSectionRef>(null);
  const [isEditingGroupName, setIsEditingGroupName] = useState(false);
  const [editedGroupName, setEditedGroupName] = useState("");

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

  // Mutations
  const updateGroupMutation = useUpdateGroup(orgId || "");
  const removeUserFromGroupMutation = useRemoveUserFromGroup(orgId || "");
  const canUpdateGroups = canAct("groups", "update");

  const loading = loadingGroup || loadingMembers;
  const error = groupError || membersError;

  const handleBackToGroups = () => {
    navigate(`/${orgId}/settings/groups`);
  };

  const handleEditGroupName = () => {
    if (!canUpdateGroups) return;
    if (group) {
      setEditedGroupName(group.spec?.displayName || "");
      setIsEditingGroupName(true);
    }
  };

  const handleSaveGroupName = async () => {
    if (!canUpdateGroups) return;
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
    } catch (_err) {
      showErrorToast("Failed to update group name");
    }
  };

  const handleCancelGroupName = () => {
    setIsEditingGroupName(false);
    setEditedGroupName("");
  };

  const handleRemoveMember = async (userId: string) => {
    if (!canUpdateGroups) return;
    if (!groupName || !orgId) return;

    try {
      await removeUserFromGroupMutation.mutateAsync({
        groupName,
        userId,
        organizationId: orgId,
      });

      // Trigger refresh of the AddMembersSection to update the "From organization" tab
      addMembersSectionRef.current?.refreshExistingMembers();
    } catch (_err) {
      showErrorToast("Failed to remove member");
    }
  };

  const handleMemberAdded = () => {
    if (!orgId || !groupName) return;
    queryClient.invalidateQueries({ queryKey: organizationKeys.groupUsers(orgId, groupName) });
    queryClient.invalidateQueries({ queryKey: organizationKeys.groups(orgId) });
  };

  if (loading) {
    return (
      <div className="flex justify-center items-center h-screen">
        <p className="text-gray-500 dark:text-gray-400">Loading group...</p>
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
        <div className="bg-white border border-red-300 text-red-500 px-4 py-2 rounded">
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

      <div className="bg-slate-50 dark:bg-gray-800 rounded-lg border border-gray-300 dark:border-gray-800 p-6 space-y-6">
        {/* Group header */}
        <div className="flex items-center justify-between">
          <div className="group">
            {isEditingGroupName ? (
              <div className="flex items-center gap-3">
                <Input
                  type="text"
                  value={editedGroupName}
                  onChange={(e) => setEditedGroupName(e.target.value)}
                  className="text-sm font-normal bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600"
                  onKeyDown={(e) => {
                    if (e.key === "Enter") handleSaveGroupName();
                    if (e.key === "Escape") handleCancelGroupName();
                  }}
                  autoFocus
                />
                <div className="flex items-center gap-2">
                  <Button onClick={handleSaveGroupName} disabled={updateGroupMutation.isPending}>
                    {updateGroupMutation.isPending ? "Saving..." : "Save"}
                  </Button>
                  <Button variant="outline" onClick={handleCancelGroupName} disabled={updateGroupMutation.isPending}>
                    Cancel
                  </Button>
                </div>
              </div>
            ) : (
              <PermissionTooltip
                allowed={canUpdateGroups || permissionsLoading}
                message="You don't have permission to update groups."
              >
                <div
                  className="flex items-center gap-2 rounded-md px-2 py-1 transition group-hover:bg-white/50 border-1 border-transparent group-hover:border-gray-300 dark:group-hover:bg-white/10 cursor-text"
                  onClick={handleEditGroupName}
                >
                  <Heading level={2} className="text-2xl font-semibold text-gray-800 dark:text-white">
                    {group?.spec?.displayName}
                  </Heading>
                  <span className="opacity-0 group-hover:opacity-100 text-xs font-medium text-gray-500 border-1 border-gray-300 rounded px-2 py-0.5">
                    Edit
                  </span>
                </div>
              </PermissionTooltip>
            )}
          </div>
          <div />
        </div>

        {/* Add Members Section */}
        <PermissionTooltip
          allowed={canUpdateGroups || permissionsLoading}
          message="You don't have permission to update group members."
          className="w-full"
        >
          <AddMembersSection
            ref={addMembersSectionRef}
            organizationId={orgId!}
            groupName={groupName!}
            showRoleSelection={false}
            onMemberAdded={handleMemberAdded}
            className="w-full"
          />
        </PermissionTooltip>

        {/* Group members table */}
        <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-300 dark:border-gray-800 overflow-hidden">
          <div className="p-6">
            <Table dense>
              <TableBody>
                {members.map((member) => (
                  <TableRow key={member.metadata?.id}>
                    <TableCell>
                      <div className="flex items-center gap-3">
                        <Avatar
                          src={member.spec?.accountProviders?.[0]?.avatarUrl}
                          initials={member.spec?.displayName?.charAt(0) || "U"}
                          className="size-8"
                        />
                        <div>
                          <div className="text-sm font-medium text-gray-800 dark:text-white">
                            {member.spec?.displayName}
                          </div>
                          <div className="text-xs text-gray-500 dark:text-gray-400">
                            Member since {new Date().toLocaleDateString()}
                          </div>
                        </div>
                      </div>
                    </TableCell>
                    <TableCell>{member.metadata?.email || "Service Account"}</TableCell>
                    <TableCell>
                      <div className="flex justify-end">
                        <PermissionTooltip
                          allowed={canUpdateGroups || permissionsLoading}
                          message="You don't have permission to update group members."
                        >
                          <TooltipProvider delayDuration={200}>
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <button
                                  type="button"
                                  onClick={() => handleRemoveMember(member.metadata!.id!)}
                                  className="p-1 rounded text-gray-800 hover:bg-gray-100 transition-colors"
                                  aria-label="Remove from Group"
                                  disabled={!canUpdateGroups}
                                >
                                  <Icon name="x" size="sm" />
                                </button>
                              </TooltipTrigger>
                              <TooltipContent side="top">Remove from Group</TooltipContent>
                            </Tooltip>
                          </TooltipProvider>
                        </PermissionTooltip>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
                {members.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={4} className="text-center text-gray-800 h-[200px] py-6 border-b-0">
                      <div className="flex flex-col items-center gap-2">
                        <Icon name="user" size="xl" className="text-gray-800" />
                        <span className="text-sm">No group members yet</span>
                      </div>
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
