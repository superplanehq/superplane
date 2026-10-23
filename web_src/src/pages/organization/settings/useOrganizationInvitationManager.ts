import { useState } from "react";
import type {
  OrganizationsCreateInvitationResponse,
  OrganizationsInvitationSeatSummary,
  OrganizationsOrganizationInvitation,
} from "@/api-client";
import { usePermissions } from "@/contexts/usePermissions";
import {
  useCancelOrganizationInvitation,
  useCreateOrganizationInvitation,
  useOrganizationInvitations,
  useResendOrganizationInvitation,
} from "@/hooks/useOrganizationInvitations";
import { getApiErrorMessage } from "@/lib/errors";
import { showSuccessToast } from "@/lib/toast";

interface InvitationLink {
  id: string;
  email: string;
  url: string;
  sentAt?: string;
}

export function useOrganizationInvitationManager(organizationId: string) {
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const canInvite = !permissionsLoading && canAct("members", "create");
  const canAssignRoles = canAct("members", "update");
  const invitations = useOrganizationInvitations(organizationId, canAct("members", "read"));
  const create = useCreateOrganizationInvitation(organizationId);
  const resend = useResendOrganizationInvitation(organizationId);
  const cancel = useCancelOrganizationInvitation(organizationId);
  const [link, setLink] = useState<InvitationLink | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const busy = create.isPending || resend.isPending || cancel.isPending;
  const capacityFull = isInvitationCapacityFull(invitations.data?.seats);
  const sendingBlocked = !invitations.isSuccess || invitations.isFetching || capacityFull;
  const visibleLink = currentInvitationLink(link, invitations.data?.invitations);

  const createInvitation = async (recipient: { email: string; role: string }) => {
    if (!canInvite || sendingBlocked || busy) return false;
    setActionError(null);
    try {
      setLink(invitationLinkFromResponse(await create.mutateAsync(recipient)));
      showSuccessToast("Invitation created.");
      return true;
    } catch (error) {
      setActionError(getApiErrorMessage(error, "Could not create the invitation. Try again."));
      return false;
    }
  };

  const resendInvitation = async (invitation: OrganizationsOrganizationInvitation) => {
    if (!canInvite || !invitation.id || busy || (!canAssignRoles && invitation.role !== "org_operator")) return;
    setActionError(null);
    // Resending rotates the token. Stop displaying the previous link immediately.
    setLink(null);
    try {
      const renewedLink = invitationLinkFromResponse(await resend.mutateAsync(invitation.id));
      setLink(renewedLink);
      showSuccessToast("Invitation renewed.");
      return renewedLink;
    } catch (error) {
      setActionError(getApiErrorMessage(error, "Could not resend the invitation. Try again."));
    }
  };

  const cancelInvitation = async (invitationId: string) => {
    if (!canInvite || busy) return;
    setActionError(null);
    try {
      await cancel.mutateAsync(invitationId);
      if (link?.id === invitationId) setLink(null);
      showSuccessToast("Invitation canceled.");
    } catch (error) {
      setActionError(getApiErrorMessage(error, "Could not cancel the invitation. Try again."));
    }
  };

  const copyInvitationLink = async (invitationLink: InvitationLink) => {
    setActionError(null);
    try {
      await navigator.clipboard.writeText(invitationLink.url);
      showSuccessToast("Invitation link copied.");
      return true;
    } catch {
      setActionError("Could not copy the link. Select and copy it manually.");
      return false;
    }
  };

  const copyLink = async () => {
    if (!canInvite || !visibleLink) return false;
    return copyInvitationLink(visibleLink);
  };

  const resendAndCopyInvitation = async (invitation: OrganizationsOrganizationInvitation) => {
    const renewedLink = await resendInvitation(invitation);
    if (!renewedLink) return false;
    return copyInvitationLink(renewedLink);
  };

  return {
    invitations,
    canInvite,
    canAssignRoles,
    busy,
    capacityFull,
    sendingBlocked,
    visibleLink,
    actionError,
    createPending: create.isPending,
    createInvitation,
    resendInvitation,
    resendAndCopyInvitation,
    cancelInvitation,
    copyLink,
  };
}

export type InvitationManager = ReturnType<typeof useOrganizationInvitationManager>;

function invitationLinkFromResponse(response: OrganizationsCreateInvitationResponse | undefined): InvitationLink {
  if (!response?.invitation?.id || !response.invitation.email || !response.acceptanceUrl) {
    throw new Error("The invitation has no link. Resend the invitation to get a new link.");
  }
  return {
    id: response.invitation.id,
    email: response.invitation.email,
    url: response.acceptanceUrl,
    sentAt: response.invitation.sentAt,
  };
}

function isInvitationCapacityFull(seats: OrganizationsInvitationSeatSummary | undefined): boolean {
  if (!seats) return true;
  if (seats.perSeat) return seats.invitable == null || Number(seats.invitable) <= 0;
  return seats.memberLimit != null && Number(seats.occupied) >= Number(seats.memberLimit);
}

function currentInvitationLink(
  link: InvitationLink | null,
  invitations: OrganizationsOrganizationInvitation[] | undefined,
): InvitationLink | null {
  if (!link) return null;
  const invitation = invitations?.find((invitation) => invitation.id === link.id);
  if (!invitation || invitation.state !== "pending" || invitation.sentAt !== link.sentAt) return null;
  if (invitation.expiresAt && new Date(invitation.expiresAt).getTime() <= Date.now()) return null;
  return link;
}
