import { useMemo, useState } from "react";
import { useAccount } from "@/contexts/AccountContext";
import { Avatar } from "../../../components/Avatar/avatar";
import { Badge } from "../../../components/Badge/badge";
import {
  Dropdown,
  DropdownButton,
  DropdownDescription,
  DropdownItem,
  DropdownLabel,
  DropdownMenu,
} from "../../../components/Dropdown/dropdown";
import { Icon } from "../../../components/Icon";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "../../../components/Table/table";
import { Text } from "../../../components/Text/text";
import {
  useAssignRole,
  useOrganizationInviteLink,
  useOrganizationRoles,
  useOrganizationUsers,
  useRemoveOrganizationSubject,
  useResetOrganizationInviteLink,
  useUpdateOrganizationInviteLink,
} from "../../../hooks/useOrganizationData";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { isRBACEnabled } from "@/lib/env";
import { Switch } from "@/ui/switch";
import { getApiErrorMessage } from "@/utils/errors";
import { showErrorToast, showSuccessToast } from "@/utils/toast";

interface Member {
  id: string;
  name: string;
  email: string;
  role: string;
  roleName: string;
  initials: string;
  avatar?: string;
  type: "member";
  status: "active";
}

interface MembersProps {
  organizationId: string;
}

export function Members({ organizationId }: MembersProps) {
  const { account } = useAccount();
  const [sortConfig, setSortConfig] = useState<{
    key: keyof Member | null;
    direction: "asc" | "desc";
  }>({ key: null, direction: "asc" });
  const [removalError, setRemovalError] = useState<string | null>(null);

  // Use React Query hooks for data fetching
  const { data: users = [], isLoading: loadingMembers, error: usersError } = useOrganizationUsers(organizationId);
  const {
    data: organizationRoles = [],
    isLoading: loadingRoles,
    error: rolesError,
  } = useOrganizationRoles(organizationId);
  const currentUserRoleNames = useMemo(() => {
    if (!account?.email) {
      return [];
    }

    const matchedUser = users.find((user) => user.metadata?.email?.toLowerCase() === account.email.toLowerCase());
    return matchedUser?.status?.roleAssignments?.map((role) => role.roleName) ?? [];
  }, [account?.email, users]);

  const canManageInviteLink = currentUserRoleNames.some(
    (roleName) => roleName === "org_owner" || roleName === "org_admin",
  );

  const {
    data: inviteLink,
    isLoading: loadingInviteLink,
    error: inviteLinkError,
  } = useOrganizationInviteLink(organizationId, canManageInviteLink);

  // Mutations for role assignment and user removal
  const assignRoleMutation = useAssignRole(organizationId);
  const removeUserMutation = useRemoveOrganizationSubject(organizationId);
  const updateInviteLinkMutation = useUpdateOrganizationInviteLink(organizationId);
  const resetInviteLinkMutation = useResetOrganizationInviteLink(organizationId);

  const error = usersError || rolesError;
  const ownerIds = useMemo(() => {
    const ids = users
      .filter((user) => user.status?.roleAssignments?.some((role) => role.roleName === "org_owner"))
      .map((user) => user.metadata?.id)
      .filter((id): id is string => Boolean(id));

    return new Set(ids);
  }, [users]);

  const inviteLinkUrl = useMemo(() => {
    if (!inviteLink?.token) {
      return "";
    }

    const origin = typeof window === "undefined" ? "" : window.location.origin;
    return `${origin}/invite/${inviteLink.token}`;
  }, [inviteLink?.token]);

  const inviteLinkErrorMessage = inviteLinkError ? getApiErrorMessage(inviteLinkError) : null;
  const showInviteLinkSection = canManageInviteLink && inviteLinkErrorMessage !== "Not found";
  const inviteLinkEnabled = inviteLink?.enabled ?? false;
  const inviteLinkBusy = updateInviteLinkMutation.isPending || resetInviteLinkMutation.isPending;

  // Transform users to Member interface format
  const members = useMemo(() => {
    return users.map((user): Member => {
      // Generate initials from displayName or userId
      const name = user.spec?.displayName || "Unknown User";
      const initials = name
        .split(" ")
        .map((n) => n[0])
        .join("")
        .toUpperCase()
        .slice(0, 2);

      // Get primary role name and display name from role assignments
      const primaryRoleName = user.status?.roleAssignments?.[0]?.roleName || "Member";
      const primaryRoleDisplayName = user.status?.roleAssignments?.[0]?.roleDisplayName || primaryRoleName;

      return {
        id: user.metadata?.id || "",
        name: name,
        email: user.metadata?.email || "",
        role: primaryRoleDisplayName,
        roleName: primaryRoleName,
        initials: initials,
        avatar: user.spec?.accountProviders?.[0]?.avatarUrl,
        type: "member",
        status: "active",
      };
    });
  }, [users]);

  const handleSort = (key: keyof Member) => {
    setSortConfig((prevConfig) => ({
      key,
      direction: prevConfig.key === key && prevConfig.direction === "asc" ? "desc" : "asc",
    }));
  };

  const getSortIcon = (columnKey: keyof Member) => {
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
    try {
      await assignRoleMutation.mutateAsync({
        userId: memberId,
        roleName: newRoleName,
      });
    } catch (err) {
      console.error("Error updating role:", err);
    }
  };

  const handleMemberRemove = async (member: Member) => {
    if (member.type === "member" && ownerIds.has(member.id) && ownerIds.size <= 1) {
      setRemovalError("You must have at least one organization owner.");
      return;
    }

    try {
      setRemovalError(null);
      if (member.type === "member") {
        await removeUserMutation.mutateAsync({
          userId: member.id,
        });
      }
    } catch (err) {
      setRemovalError("Unable to remove this member.");
      console.error("Error removing member:", err);
    }
  };

  const handleInviteLinkToggle = async (enabled: boolean) => {
    try {
      await updateInviteLinkMutation.mutateAsync(enabled);
    } catch (err) {
      console.error("Error updating invite link:", err);
      showErrorToast("Failed to update invite link.");
    }
  };

  const handleInviteLinkReset = async () => {
    try {
      await resetInviteLinkMutation.mutateAsync();
      showSuccessToast("Invite link reset.");
    } catch (err) {
      console.error("Error resetting invite link:", err);
      showErrorToast("Failed to reset invite link.");
    }
  };

  const handleCopyInviteLink = async () => {
    if (!inviteLinkUrl) return;

    try {
      await navigator.clipboard.writeText(inviteLinkUrl);
      showSuccessToast("Invite link copied.");
    } catch (err) {
      console.error("Error copying invite link:", err);
      showErrorToast("Failed to copy invite link.");
    }
  };

  return (
    <div className="space-y-6 pt-6">
      {error && (
        <div className="bg-white border border-red-300 text-red-500 px-4 py-2 rounded">
          <p>{error instanceof Error ? error.message : "Failed to fetch data"}</p>
        </div>
      )}

      {showInviteLinkSection ? (
        <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-300 dark:border-gray-800 p-6">
          <div className="flex items-start justify-between gap-6">
            <div>
              <Text className="text-left font-semibold text-gray-800 dark:text-white mb-1">
                Invite link to add members
              </Text>
              <Text className="text-sm text-gray-500 dark:text-gray-400">
                Only people with owner and admin roles can see this.
                {inviteLinkEnabled && (
                  <>
                    {" "}
                    You can also{" "}
                    <button
                      type="button"
                      className="text-blue-600 hover:underline disabled:text-gray-400"
                      onClick={handleInviteLinkReset}
                      disabled={loadingInviteLink || inviteLinkBusy}
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
                disabled={loadingInviteLink || inviteLinkBusy}
                aria-label="Toggle invite link"
              />
            </div>
          </div>

          {inviteLinkErrorMessage && inviteLinkErrorMessage !== "Not found" && (
            <div className="bg-white border border-red-300 text-red-500 px-4 py-2 rounded mt-4">
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
                disabled={!inviteLinkUrl || loadingInviteLink || inviteLinkBusy}
              >
                <Icon name="copy" />
                Copy link
              </Button>
            </div>
          )}
        </div>
      ) : (
        <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-300 dark:border-gray-800 p-6">
          <Text className="text-left font-semibold text-gray-800 dark:text-white mb-1">Invite link to add members</Text>
          <Text className="text-sm text-gray-500 dark:text-gray-400">
            Reach out to an organization owner or admin to invite new members.
          </Text>
        </div>
      )}

      {/* Members List */}
      <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-300 dark:border-gray-800 overflow-hidden">
        <div className="px-6 pt-6 pb-4">
          <div className="flex items-center justify-between mb-4">
            <Text className="text-sm font-medium text-gray-600 dark:text-gray-300">Members ({members.length})</Text>
          </div>
        </div>

        <div className="px-6 pb-6">
          {removalError && (
            <div className="bg-white border border-red-300 text-red-500 px-4 py-2 rounded mb-4">
              <p>{removalError}</p>
            </div>
          )}
          {loadingMembers ? (
            <div className="flex justify-center items-center h-32">
              <p className="text-gray-500 dark:text-gray-400">Loading...</p>
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
                      Name
                      <Icon name={getSortIcon("name")} size="sm" className="text-gray-400" />
                    </div>
                  </TableHeader>
                  <TableHeader
                    className="cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-700/50"
                    onClick={() => handleSort("email")}
                  >
                    <div className="flex items-center gap-2">
                      Email
                      <Icon name={getSortIcon("email")} size="sm" className="text-gray-400" />
                    </div>
                  </TableHeader>
                  {isRBACEnabled() && (
                    <TableHeader
                      className="cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-700/50"
                      onClick={() => handleSort("role")}
                    >
                      <div className="flex items-center gap-2">
                        Role
                        <Icon name={getSortIcon("role")} size="sm" className="text-gray-400" />
                      </div>
                    </TableHeader>
                  )}
                  <TableHeader
                    className="cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-700/50"
                    onClick={() => handleSort("status")}
                  >
                    <div className="flex items-center gap-2">
                      Status
                      <Icon name={getSortIcon("status")} size="sm" className="text-gray-400" />
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
                          <div className="text-sm font-medium text-gray-800 dark:text-white">{member.name}</div>
                        </div>
                      </div>
                    </TableCell>
                    <TableCell>{member.email}</TableCell>
                    {isRBACEnabled() && (
                      <TableCell>
                        <Dropdown>
                          <DropdownButton className="flex items-center gap-2 text-sm">
                            {member.role}
                            <Icon name="chevron-down" />
                          </DropdownButton>
                          <DropdownMenu>
                            {organizationRoles.map((role) => (
                              <DropdownItem
                                key={role.metadata?.name}
                                onClick={() => handleRoleChange(member.id, role.metadata?.name || "")}
                                disabled={loadingRoles}
                              >
                                <DropdownLabel>{role.spec?.displayName || role.metadata?.name}</DropdownLabel>
                                {role.spec?.description && (
                                  <DropdownDescription>{role.spec?.description}</DropdownDescription>
                                )}
                              </DropdownItem>
                            ))}
                            {loadingRoles && (
                              <DropdownItem disabled>
                                <DropdownLabel>Loading roles...</DropdownLabel>
                              </DropdownItem>
                            )}
                          </DropdownMenu>
                        </Dropdown>
                      </TableCell>
                    )}
                    <TableCell>
                      <Badge color="green">Active</Badge>
                    </TableCell>
                    <TableCell>
                      <div className="flex justify-end">
                        {ownerIds.has(member.id) && ownerIds.size <= 1 ? (
                          <Dropdown>
                            <DropdownButton className="flex items-center gap-2 text-sm">
                              <Icon name="ellipsis-vertical" size="sm" />
                            </DropdownButton>
                            <DropdownMenu>
                              <DropdownItem disabled>
                                <Icon name="x" size="sm" />
                                <span className="ml-1">Cannot remove last owner</span>
                              </DropdownItem>
                            </DropdownMenu>
                          </Dropdown>
                        ) : (
                          <Dropdown>
                            <DropdownButton className="flex items-center gap-2 text-sm">
                              <Icon name="ellipsis-vertical" size="sm" />
                            </DropdownButton>
                            <DropdownMenu>
                              <DropdownItem
                                className="flex items-center gap-1"
                                onClick={() => handleMemberRemove(member)}
                              >
                                <Icon name="x" size="sm" />
                                Remove
                              </DropdownItem>
                            </DropdownMenu>
                          </Dropdown>
                        )}
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
                {getSortedMembers().length === 0 && (
                  <TableRow>
                    <TableCell colSpan={5} className="text-center py-8">
                      <div className="text-gray-500 dark:text-gray-400">
                        <Icon name="search" className="h-12 w-12 mx-auto mb-4 text-gray-300" />
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
