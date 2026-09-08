import { useMemo, useState } from "react";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { usePermissions } from "@/contexts/usePermissions";
import { Avatar } from "../../../components/Avatar/avatar";
import { Badge } from "../../../components/Badge/badge";
import { Icon } from "../../../components/Icon";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "../../../components/Table/table";
import { Text } from "../../../components/Text/text";
import { PermissionTooltip } from "@/components/PermissionGate";
import {
  useAssignRole,
  useOrganizationInviteLink,
  useOrganizationRoles,
  useOrganizationUsers,
  useRemoveOrganizationSubject,
  useResetOrganizationInviteLink,
  useSetUserOwner,
  useUpdateOrganizationInviteLink,
} from "../../../hooks/useOrganizationData";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Switch } from "@/ui/switch";
import { getApiErrorMessage } from "@/lib/errors";
import { cn } from "@/lib/utils";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { settingsCardClassName, settingsErrorClassName, settingsTableCardClassName } from "./settingsPageStyles";
import { useMe } from "@/hooks/useMe";
import { defaultOrganizationRoleSortIndex } from "@/lib/organizationRoles";
import { MemberOverflowMenu, MemberRoleSelect, type OrganizationMember } from "./MemberRowControls";

interface MembersProps {
  organizationId: string;
}

export function Members({ organizationId }: MembersProps) {
  usePageTitle(["Members"]);
  const { data: me } = useMe();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const [sortConfig, setSortConfig] = useState<{
    key: keyof OrganizationMember | null;
    direction: "asc" | "desc";
  }>({ key: null, direction: "asc" });
  const [removalError, setRemovalError] = useState<string | null>(null);

  const { data: users = [], isLoading: loadingMembers, error: usersError } = useOrganizationUsers(organizationId, true);
  const {
    data: organizationRoles = [],
    isLoading: loadingRoles,
    error: rolesError,
  } = useOrganizationRoles(organizationId);
  const canManageInviteLink = canAct("members", "create");
  const canUpdateMembers = canAct("members", "update");
  const canDeleteMembers = canAct("members", "delete");

  const {
    data: inviteLink,
    isLoading: loadingInviteLink,
    error: inviteLinkError,
  } = useOrganizationInviteLink(organizationId, canManageInviteLink);

  // Mutations for role assignment and user removal
  const assignRoleMutation = useAssignRole(organizationId);
  const setUserOwnerMutation = useSetUserOwner(organizationId);
  const removeUserMutation = useRemoveOrganizationSubject(organizationId);
  const updateInviteLinkMutation = useUpdateOrganizationInviteLink(organizationId);
  const resetInviteLinkMutation = useResetOrganizationInviteLink(organizationId);

  const error = usersError || rolesError;

  useReportPageReady(!loadingMembers && !loadingRoles && !permissionsLoading, {
    failed: !!error,
  });

  const ownerIds = useMemo(() => {
    const ids = users
      .filter((user) => user.status?.isOwner)
      .map((user) => user.metadata?.id)
      .filter((id): id is string => Boolean(id));

    return new Set(ids);
  }, [users]);

  const assignableRoles = useMemo(() => {
    return [...organizationRoles].sort((a, b) => {
      const aDefault = defaultOrganizationRoleSortIndex(a.metadata?.name);
      const bDefault = defaultOrganizationRoleSortIndex(b.metadata?.name);
      if (aDefault !== bDefault) {
        return aDefault - bDefault;
      }
      const aName = (a.spec?.displayName || a.metadata?.name || "").toLowerCase();
      const bName = (b.spec?.displayName || b.metadata?.name || "").toLowerCase();
      return aName.localeCompare(bName);
    });
  }, [organizationRoles]);

  const inviteLinkUrl = useMemo(() => {
    if (!inviteLink?.token) {
      return "";
    }

    const origin = typeof window === "undefined" ? "" : window.location.origin;
    return `${origin}/invite/${inviteLink.token}`;
  }, [inviteLink?.token]);

  const inviteLinkErrorMessage = inviteLinkError ? getApiErrorMessage(inviteLinkError) : null;
  const showInviteLinkSection = inviteLinkErrorMessage !== "Not found";
  const inviteLinkEnabled = inviteLink?.enabled ?? false;
  const inviteLinkBusy = updateInviteLinkMutation.isPending || resetInviteLinkMutation.isPending;

  // Transform users to Member interface format
  const members = useMemo(() => {
    return users.map((user): OrganizationMember => {
      // Generate initials from displayName or userId
      const name = user.spec?.displayName || "Unknown User";
      const initials = name
        .split(" ")
        .map((n) => n[0])
        .join("")
        .toUpperCase()
        .slice(0, 2);

      // Get primary role name and display name from role assignments
      const primaryRoleName = user.status?.roles?.[0]?.roleName || "Member";
      const primaryRoleDisplayName = user.status?.roles?.[0]?.roleDisplayName || primaryRoleName;

      return {
        id: user.metadata?.id || "",
        name: name,
        email: user.metadata?.email || "",
        role: primaryRoleDisplayName,
        roleName: primaryRoleName,
        initials: initials,
        avatar: user.status?.accountProviders?.[0]?.avatarUrl,
        type: "member",
        status: "active",
        isOwner: Boolean(user.status?.isOwner),
      };
    });
  }, [users]);

  const handleSort = (key: keyof OrganizationMember) => {
    setSortConfig((prevConfig) => ({
      key,
      direction: prevConfig.key === key && prevConfig.direction === "asc" ? "desc" : "asc",
    }));
  };

  const getSortIcon = (columnKey: keyof OrganizationMember) => {
    if (sortConfig.key !== columnKey) {
      return "chevrons-up-down";
    }
    return sortConfig.direction === "asc" ? "chevron-up" : "chevron-down";
  };

  const getSortedMembers = () => {
    if (!sortConfig.key) return members;

    return [...members].sort((a, b) => {
      const aValue = a[sortConfig.key!];
      const bValue = b[sortConfig.key!];

      if (aValue == null && bValue == null) return 0;
      if (aValue == null) return sortConfig.direction === "asc" ? -1 : 1;
      if (bValue == null) return sortConfig.direction === "asc" ? 1 : -1;

      if (aValue < bValue) {
        return sortConfig.direction === "asc" ? -1 : 1;
      }
      if (aValue > bValue) {
        return sortConfig.direction === "asc" ? 1 : -1;
      }
      return 0;
    });
  };

  const handleRoleChange = async (memberId: string, newRoleName: string) => {
    if (!canUpdateMembers || me?.id === memberId) return;
    try {
      await assignRoleMutation.mutateAsync({
        userId: memberId,
        roleName: newRoleName,
      });
    } catch {
      showErrorToast("Failed to update role.");
    }
  };

  const handleOwnerChange = async (member: OrganizationMember, isOwner: boolean) => {
    if (!canUpdateMembers) return;
    if (!isOwner && member.isOwner && ownerIds.size <= 1) {
      setRemovalError("The organization must keep at least one owner.");
      return;
    }

    try {
      setRemovalError(null);
      await setUserOwnerMutation.mutateAsync({
        userId: member.id,
        isOwner,
      });
    } catch (error) {
      setRemovalError(getApiErrorMessage(error, "Unable to update owner."));
    }
  };

  const handleMemberRemove = async (member: OrganizationMember) => {
    if (!canDeleteMembers) return;
    if (member.type === "member" && ownerIds.has(member.id) && ownerIds.size <= 1) {
      setRemovalError("The organization must keep at least one owner.");
      return;
    }

    try {
      setRemovalError(null);
      if (member.type === "member") {
        await removeUserMutation.mutateAsync({
          userId: member.id,
        });
      }
    } catch {
      setRemovalError("Unable to remove this member.");
    }
  };

  const handleInviteLinkToggle = async (enabled: boolean) => {
    try {
      await updateInviteLinkMutation.mutateAsync(enabled);
    } catch {
      showErrorToast("Failed to update invite link.");
    }
  };

  const handleInviteLinkReset = async () => {
    try {
      await resetInviteLinkMutation.mutateAsync();
      showSuccessToast("Invite link reset.");
    } catch {
      showErrorToast("Failed to reset invite link.");
    }
  };

  const handleCopyInviteLink = async () => {
    if (!inviteLinkUrl) return;

    try {
      await navigator.clipboard.writeText(inviteLinkUrl);
      showSuccessToast("Invite link copied.");
    } catch {
      showErrorToast("Failed to copy invite link.");
    }
  };

  return (
    <div className="space-y-6 pt-6">
      {error && (
        <div className={settingsErrorClassName}>
          <p>{error instanceof Error ? error.message : "Failed to fetch data"}</p>
        </div>
      )}

      {showInviteLinkSection ? (
        <PermissionTooltip
          allowed={canManageInviteLink || permissionsLoading}
          message="You don't have permission to invite members."
          className="w-full"
        >
          <div className={settingsCardClassName}>
            <div className="flex items-start justify-between gap-6">
              <div>
                <Text className="text-left font-semibold text-gray-800 dark:text-white mb-1">
                  Invite link to add members
                </Text>
                <Text className="text-sm text-gray-500 dark:text-gray-400">
                  {canManageInviteLink
                    ? "Only people with required roles can see this."
                    : "You don't have permission to manage invite links."}
                  {inviteLinkEnabled && (
                    <>
                      {" "}
                      You can also{" "}
                      <button
                        type="button"
                        className="text-blue-600 hover:underline hover:text-blue-700 disabled:text-gray-400 dark:text-indigo-300 dark:hover:text-indigo-200 dark:disabled:text-gray-500"
                        onClick={handleInviteLinkReset}
                        disabled={loadingInviteLink || inviteLinkBusy || !canManageInviteLink}
                      >
                        generate a new link
                      </button>
                      .
                    </>
                  )}
                </Text>
              </div>
              <div className="flex items-center gap-3">
                <Switch
                  checked={inviteLinkEnabled}
                  onCheckedChange={handleInviteLinkToggle}
                  disabled={loadingInviteLink || inviteLinkBusy || !canManageInviteLink}
                  aria-label="Toggle invite link"
                />
              </div>
            </div>

            {inviteLinkErrorMessage && inviteLinkErrorMessage !== "Not found" && (
              <div className={cn(settingsErrorClassName, "mt-4")}>
                <p className="text-sm">{inviteLinkErrorMessage}</p>
              </div>
            )}

            {!inviteLinkEnabled && !loadingInviteLink && (
              <div className="mt-4 text-sm text-gray-500 dark:text-gray-400">Invite link is currently disabled.</div>
            )}

            {inviteLinkEnabled && inviteLinkUrl && (
              <div className="mt-4 flex flex-wrap items-center gap-3">
                <Input
                  readOnly
                  value={inviteLinkUrl}
                  className="flex-1 bg-gray-50 dark:bg-gray-900 text-gray-600 dark:text-gray-300"
                />
                <Button
                  variant="outline"
                  onClick={handleCopyInviteLink}
                  disabled={!inviteLinkUrl || loadingInviteLink || inviteLinkBusy || !canManageInviteLink}
                >
                  <Icon name="copy" />
                  Copy link
                </Button>
              </div>
            )}
          </div>
        </PermissionTooltip>
      ) : (
        <div className={settingsCardClassName}>
          <Text className="text-left font-semibold text-gray-800 dark:text-white mb-1">Invite link to add members</Text>
          <Text className="text-sm text-gray-500 dark:text-gray-400">
            Reach out to an organization owner or admin to invite new members.
          </Text>
        </div>
      )}

      {/* Members List */}
      <div className={settingsTableCardClassName}>
        <div className="px-6 pt-6 pb-4">
          <div className="flex items-center justify-between mb-4">
            <Text className="text-sm font-medium text-gray-600 dark:text-gray-300">Members ({members.length})</Text>
          </div>
        </div>

        <div className="px-6 pb-6">
          {removalError && (
            <div className={cn(settingsErrorClassName, "mb-4")}>
              <p>{removalError}</p>
            </div>
          )}
          {loadingMembers ? (
            <div className="flex justify-center items-center h-32">
              <p className="text-gray-500 dark:text-gray-400">Loading...</p>
            </div>
          ) : (
            <Table dense className="!overflow-x-hidden !whitespace-normal">
              <TableHead>
                <TableRow>
                  <TableHeader
                    className="cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-700/50"
                    onClick={() => handleSort("name")}
                  >
                    <div className="flex items-center gap-2">
                      Name
                      <Icon name={getSortIcon("name")} size="sm" className="text-gray-400 dark:text-gray-500" />
                    </div>
                  </TableHeader>
                  <TableHeader
                    className="cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-700/50"
                    onClick={() => handleSort("email")}
                  >
                    <div className="flex items-center gap-2">
                      Email
                      <Icon name={getSortIcon("email")} size="sm" className="text-gray-400 dark:text-gray-500" />
                    </div>
                  </TableHeader>
                  <TableHeader
                    className="cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-700/50"
                    onClick={() => handleSort("role")}
                  >
                    <div className="flex items-center gap-2">
                      Role
                      <Icon name={getSortIcon("role")} size="sm" className="text-gray-400 dark:text-gray-500" />
                    </div>
                  </TableHeader>
                  <TableHeader
                    className="cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-700/50"
                    onClick={() => handleSort("status")}
                  >
                    <div className="flex items-center gap-2">
                      Status
                      <Icon name={getSortIcon("status")} size="sm" className="text-gray-400 dark:text-gray-500" />
                    </div>
                  </TableHeader>
                  <TableHeader></TableHeader>
                </TableRow>
              </TableHead>
              <TableBody>
                {getSortedMembers().map((member) => (
                  <TableRow key={member.id} className="last:[&>td]:border-b-0">
                    <TableCell>
                      <div className="flex items-center gap-3">
                        <Avatar src={member.avatar} initials={member.initials} className="size-8" />
                        <div>
                          <div className="flex items-center gap-2">
                            <div className="text-sm font-medium text-gray-800 dark:text-white">{member.name}</div>
                            {member.isOwner && <Badge color="yellow">Owner</Badge>}
                          </div>
                        </div>
                      </div>
                    </TableCell>
                    <TableCell className="min-w-0">
                      <div className="max-w-[26rem] truncate" title={member.email}>
                        {member.email}
                      </div>
                    </TableCell>
                    <TableCell>
                      <MemberRoleSelect
                        member={member}
                        currentUserId={me?.id}
                        canUpdateMembers={canUpdateMembers}
                        permissionsLoading={permissionsLoading}
                        loadingRoles={loadingRoles}
                        assignableRoles={assignableRoles}
                        onRoleChange={handleRoleChange}
                      />
                    </TableCell>
                    <TableCell>
                      <Badge color="green">Active</Badge>
                    </TableCell>
                    <TableCell>
                      <div className="flex justify-end">
                        <MemberOverflowMenu
                          member={member}
                          ownerCount={ownerIds.size}
                          canUpdateMembers={canUpdateMembers}
                          canDeleteMembers={canDeleteMembers}
                          permissionsLoading={permissionsLoading}
                          ownerChangePending={setUserOwnerMutation.isPending}
                          onOwnerChange={handleOwnerChange}
                          onRemove={handleMemberRemove}
                        />
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
                {getSortedMembers().length === 0 && (
                  <TableRow>
                    <TableCell colSpan={5} className="text-center py-8">
                      <div className="text-gray-500 dark:text-gray-400">
                        <Icon name="search" className="mx-auto mb-4 h-12 w-12 text-gray-300 dark:text-gray-600" />
                        <p className="text-lg font-medium text-gray-800 dark:text-white mb-2">No members yet</p>
                        <p className="text-sm">Add members to get started</p>
                      </div>
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          )}
        </div>
      </div>
    </div>
  );
}
