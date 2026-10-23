import { MoreVertical } from "lucide-react";
import type { OrganizationsOrganizationInvitation } from "@/api-client";
import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import type { AssignableRoleOption } from "./MemberRowControls";
import { MemberIdentity, MemberStatus, MemberTableRow } from "./MemberTableRow";
import type { InvitationManager } from "./useOrganizationInvitationManager";

export function MemberInvitationRow({
  invitation,
  manager,
  roles,
  onShowLink,
}: {
  invitation: OrganizationsOrganizationInvitation;
  manager: InvitationManager;
  roles: AssignableRoleOption[];
  onShowLink: () => void;
}) {
  const role = roles.find((option) => option.metadata?.name === invitation.role)?.spec?.displayName || invitation.role;
  const expired = invitation.state === "expired";
  return (
    <MemberTableRow
      identity={
        <MemberIdentity
          email={invitation.email || "Unknown email"}
          initials={invitation.email?.slice(0, 2).toUpperCase() || "?"}
        />
      }
      role={
        <span className="inline-flex min-h-6 items-center text-xs text-muted-foreground @min-[44rem]/members:min-h-8 @min-[44rem]/members:text-sm">
          {role}
        </span>
      }
      status={<MemberStatus status={expired ? "expired" : "pending"} />}
      actions={
        manager.canInvite && <InvitationRowActions invitation={invitation} manager={manager} onShowLink={onShowLink} />
      }
    />
  );
}

function InvitationRowActions({
  invitation,
  manager,
  onShowLink,
}: {
  invitation: OrganizationsOrganizationInvitation;
  manager: InvitationManager;
  onShowLink: () => void;
}) {
  const knownLink = manager.visibleLink?.id === invitation.id;
  const resendBlocked =
    (!manager.canAssignRoles && invitation.role !== "org_operator") ||
    (invitation.state === "pending" && manager.invitations.data?.seats?.perSeat ? false : manager.sendingBlocked);
  const copy = async () => {
    const copied = knownLink ? await manager.copyLink() : await manager.resendAndCopyInvitation(invitation);
    if (!copied) onShowLink();
  };
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          className="size-11 shrink-0 p-0 @min-[44rem]/members:size-8"
          aria-label={`Invitation actions for ${invitation.email}`}
          disabled={manager.busy}
        >
          <MoreVertical className="size-4" aria-hidden="true" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem
          className="min-h-11 sm:min-h-8"
          disabled={manager.busy || (!knownLink && resendBlocked)}
          onClick={copy}
        >
          {knownLink ? "Copy link" : "Resend and copy link"}
        </DropdownMenuItem>
        <DropdownMenuItem
          className="min-h-11 sm:min-h-8"
          disabled={manager.busy || resendBlocked}
          onClick={() => manager.resendInvitation(invitation)}
        >
          Resend invitation
        </DropdownMenuItem>
        <DropdownMenuItem
          className="min-h-11 sm:min-h-8"
          disabled={manager.busy || !invitation.id}
          onClick={() => invitation.id && manager.cancelInvitation(invitation.id)}
        >
          Cancel invitation
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
