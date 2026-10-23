import { useMemo, useState } from "react";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { usePermissions } from "@/contexts/usePermissions";
import {
  useAssignRole,
  useOrganizationRoles,
  useOrganizationMembers,
  useRemoveOrganizationSubject,
  useSetUserOwner,
} from "../../../hooks/useOrganizationData";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { settingsErrorClassName, settingsTableCardClassName } from "./settingsPageStyles";
import { useMe } from "@/hooks/useMe";
import {
  InvitationCapacityNotice,
  InvitationErrors,
  InvitationLinkDialog,
  InviteMemberDialog,
} from "./OrganizationInvitations";
import { useOrganizationInvitationManager } from "./useOrganizationInvitationManager";
import { MembersTable } from "./MembersTable";
import { Button } from "@/components/ui/button";
import { Plus } from "lucide-react";
import { defaultOrganizationRoleSortIndex } from "@/lib/organizationRoles";
import { type OrganizationMember } from "./MemberRowControls";

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
  const [inviteOpen, setInviteOpen] = useState(false);
  const [linkOpen, setLinkOpen] = useState(false);
  const invitationManager = useOrganizationInvitationManager(organizationId);

  const {
    users,
    seatLimit,
    isLoading: loadingMembers,
    isFetching: refreshingMembers,
    isSuccess: membersLoaded,
    error: usersError,
  } = useOrganizationMembers(organizationId);
  const seatsFull = seatLimit != null && users.length >= seatLimit;
  const capacityLoading = loadingMembers || refreshingMembers;
  const {
    data: organizationRoles = [],
    isLoading: loadingRoles,
    error: rolesError,
  } = useOrganizationRoles(organizationId);
  const canUpdateMembers = canAct("members", "update");
  const canDeleteMembers = canAct("members", "delete");

  // Mutations for role assignment and user removal
  const assignRoleMutation = useAssignRole(organizationId);
  const setUserOwnerMutation = useSetUserOwner(organizationId);
  const removeUserMutation = useRemoveOrganizationSubject(organizationId);

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

  const showingLink = linkOpen && !!invitationManager.visibleLink;
  const memberControls = {
    currentUserId: me?.id,
    canUpdateMembers,
    canDeleteMembers,
    permissionsLoading,
    loadingRoles: loadingRoles || !!rolesError,
    assignableRoles,
    ownerCount: ownerIds.size,
    ownerChangePending: setUserOwnerMutation.isPending,
    onRoleChange: handleRoleChange,
    onOwnerChange: handleOwnerChange,
    onRemove: handleMemberRemove,
  };

  return (
    <div className="min-w-0 space-y-4 pt-6">
      {error && (
        <p role="alert" className={settingsErrorClassName}>
          {getApiErrorMessage(error, "Could not load members. Refresh the page to try again.")}
        </p>
      )}
      {removalError && (
        <p role="alert" className={settingsErrorClassName}>
          {removalError}
        </p>
      )}
      {!inviteOpen && !showingLink && <InvitationErrors manager={invitationManager} />}
      <section className={`${settingsTableCardClassName} @container/members`} aria-label="Members and invitations">
        <div className="flex flex-wrap items-center justify-between gap-3 border-b border-border p-4 sm:px-6">
          <div className="min-w-0 text-sm text-muted-foreground" aria-live="polite">
            {capacityLoading
              ? "Loading seat usage..."
              : membersLoaded &&
                (seatLimit == null
                  ? `${users.length} ${users.length === 1 ? "member" : "members"}`
                  : `${users.length} of ${seatLimit} seats used`)}
          </div>
          {invitationManager.canInvite && (
            <Button type="button" onClick={() => setInviteOpen(true)} className="h-11 sm:h-8">
              <Plus className="size-4" aria-hidden="true" />
              Invite member
            </Button>
          )}
        </div>
        {loadingMembers ? (
          <p className="p-6 text-sm text-muted-foreground" role="status">
            Loading members...
          </p>
        ) : (
          <MembersTable
            members={getSortedMembers()}
            manager={invitationManager}
            controls={memberControls}
            onShowLink={() => setLinkOpen(true)}
            sort={{ onSort: handleSort, icon: getSortIcon }}
          />
        )}
        {invitationManager.invitations.isLoading && (
          <p role="status" className="px-4 py-3 text-sm text-muted-foreground">
            Loading invitations...
          </p>
        )}
        {membersLoaded &&
          members.length === 0 &&
          !invitationManager.invitations.data?.invitations?.some(
            (invitation) => invitation.state === "pending" || invitation.state === "expired",
          ) && <p className="p-6 text-sm text-muted-foreground">No members yet. Invite a member to get started.</p>}
      </section>
      {seatsFull && (
        <p className="text-sm text-amber-700 dark:text-amber-400">
          {users.length > seatLimit! ? "Seat limit exceeded." : "Seat limit reached."}
        </p>
      )}
      {!inviteOpen && <InvitationCapacityNotice manager={invitationManager} />}
      {invitationManager.canInvite && (
        <InviteMemberDialog
          key={organizationId}
          open={inviteOpen}
          onOpenChange={setInviteOpen}
          manager={invitationManager}
          roles={assignableRoles}
          rolesLoading={loadingRoles || !!rolesError}
          onCreated={() => setLinkOpen(true)}
        />
      )}
      <InvitationLinkDialog manager={invitationManager} open={linkOpen} onOpenChange={setLinkOpen} />
    </div>
  );
}
