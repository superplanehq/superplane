import { useMemo } from "react";

import { usePermissions } from "@/contexts/usePermissions";
import { useMe } from "@/hooks/useMe";
import {
  useAssignRole,
  useOrganizationInviteLink,
  useOrganizationRoles,
  useOrganizationUsers,
  useRemoveOrganizationSubject,
  useResetOrganizationInviteLink,
  useUpdateOrganizationInviteLink,
} from "@/hooks/useOrganizationData";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";

import { useFactorySettingsLayout } from "../factorySettingsLayoutContext";

export interface SettingsRedesignMember {
  id: string;
  name: string;
  email: string;
  roleName: string;
  roleLabel: string;
}

export interface SettingsRedesignRoleOption {
  name: string;
  label: string;
}

export function useSettingsRedesignOrganizationMembers() {
  const { organizationId } = useFactorySettingsLayout();
  const { data: me } = useMe();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const { data: users = [], isLoading: usersLoading, error: usersError } = useOrganizationUsers(organizationId, true);
  const { data: organizationRoles = [], isLoading: rolesLoading } = useOrganizationRoles(organizationId);
  const canManageInvite = canAct("members", "create");
  const canUpdateMembers = canAct("members", "update");
  const canDeleteMembers = canAct("members", "delete");
  const { data: inviteLink, isLoading: inviteLoading } = useOrganizationInviteLink(organizationId, canManageInvite);
  const assignRole = useAssignRole(organizationId);
  const removeUser = useRemoveOrganizationSubject(organizationId);
  const updateInvite = useUpdateOrganizationInviteLink(organizationId);
  const resetInvite = useResetOrganizationInviteLink(organizationId);

  const members = useMemo(() => users.map(toSettingsMember), [users]);
  const roles = useMemo(() => organizationRoles.map(toRoleOption).filter((role) => role.name), [organizationRoles]);
  const ownerIds = useMemo(
    () => new Set(members.filter((member) => member.roleName === "org_owner").map((member) => member.id)),
    [members],
  );
  const inviteUrl = inviteLink?.token
    ? `${typeof window === "undefined" ? "" : window.location.origin}/invite/${inviteLink.token}`
    : "";

  return {
    meId: me?.id,
    members,
    roles,
    ownerIds,
    usersLoading,
    rolesLoading,
    usersError: usersError ? getApiErrorMessage(usersError, "Failed to load members.") : null,
    permissionsLoading,
    canManageInvite,
    canUpdateMembers,
    canDeleteMembers,
    inviteEnabled: inviteLink?.enabled ?? false,
    inviteUrl,
    inviteLoading,
    inviteBusy: updateInvite.isPending || resetInvite.isPending,
    changeRole: async (memberId: string, roleName: string) => {
      if (!canUpdateMembers || memberId === me?.id) return;
      try {
        await assignRole.mutateAsync({ userId: memberId, roleName });
        showSuccessToast("Role updated.");
      } catch {
        showErrorToast("Failed to update role.");
      }
    },
    removeMember: async (member: SettingsRedesignMember) => {
      if (!canDeleteMembers || member.id === me?.id) return;
      if (member.roleName === "org_owner" && ownerIds.size <= 1) {
        showErrorToast("You must keep at least one organization owner.");
        return;
      }
      try {
        await removeUser.mutateAsync({ userId: member.id });
        showSuccessToast("Member removed.");
      } catch {
        showErrorToast("Unable to remove this member.");
      }
    },
    toggleInvite: async (enabled: boolean) => {
      try {
        await updateInvite.mutateAsync(enabled);
      } catch {
        showErrorToast("Failed to update invite link.");
      }
    },
    resetInviteLink: async () => {
      try {
        await resetInvite.mutateAsync();
        showSuccessToast("Invite link reset.");
      } catch {
        showErrorToast("Failed to reset invite link.");
      }
    },
    copyInvite: async () => {
      if (!inviteUrl) return;
      try {
        await navigator.clipboard.writeText(inviteUrl);
        showSuccessToast("Invite link copied.");
      } catch {
        showErrorToast("Failed to copy invite link.");
      }
    },
  };
}

function toSettingsMember(user: {
  metadata?: { id?: string; email?: string };
  spec?: { displayName?: string };
  status?: { roles?: Array<{ roleName?: string; roleDisplayName?: string }> };
}): SettingsRedesignMember {
  const role = user.status?.roles?.[0];
  return {
    id: user.metadata?.id || "",
    name: user.spec?.displayName || "Unknown user",
    email: user.metadata?.email || "",
    roleName: role?.roleName || "org_member",
    roleLabel: role?.roleDisplayName || role?.roleName || "Member",
  };
}

function toRoleOption(role: {
  metadata?: { name?: string };
  spec?: { displayName?: string };
}): SettingsRedesignRoleOption {
  return {
    name: role.metadata?.name || "",
    label: role.spec?.displayName || role.metadata?.name || "",
  };
}
