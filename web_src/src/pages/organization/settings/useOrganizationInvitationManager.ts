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
import { getApiErrorMessage, getApiErrorReason } from "@/lib/errors";
import { showSuccessToast } from "@/lib/toast";

export type InvitationCapacity =
  | { state: "overCapacity"; members: number; paidSeats: number }
  | {
      state: "paymentBlocked" | "planMaximum" | "purchaseRequired" | "memberLimit" | "available" | "unlimited";
    };

const SEAT_LIMIT_REASONS = [
  "PAYMENT_PAST_DUE",
  "SEAT_PURCHASE_REQUIRED",
  "PLAN_MAXIMUM_REACHED",
  "MEMBER_LIMIT_REACHED",
];

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
  const canManageBilling = canAct("org", "update");
  const invitations = useOrganizationInvitations(organizationId, canAct("members", "read"));
  const create = useCreateOrganizationInvitation(organizationId);
  const resend = useResendOrganizationInvitation(organizationId);
  const cancel = useCancelOrganizationInvitation(organizationId);
  const [link, setLink] = useState<InvitationLink | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const busy = create.isPending || resend.isPending || cancel.isPending;
  const seats = invitations.data?.seats;
  const capacity = describeInvitationCapacity(seats);
  const sendingBlocked = !invitations.isSuccess || invitations.isFetching || blocksInvitations(capacity);
  const canPurchaseSeats =
    !!seats?.perSeat && (seats.purchasable == null || Number(seats.purchasable) > 0) && canManageBilling;
  const visibleLink = currentInvitationLink(link, invitations.data?.invitations);

  const reportActionFailure = (error: unknown, fallback: string) => {
    const reason = getApiErrorReason(error);
    if (reason === "INVITATION_RATE_LIMITED") {
      setActionError("Too many invitations were sent. Wait, then try again.");
      return;
    }
    if (reason === "PENDING_INVITATION_LIMIT_REACHED") {
      setActionError("Too many invitations are pending. Cancel a pending invitation, then try again.");
      return;
    }
    setActionError(getApiErrorMessage(error, fallback));
    // Another request changed the seat count after this page loaded it.
    if (reason && SEAT_LIMIT_REASONS.includes(reason)) void invitations.refetch();
  };

  const createInvitation = async (recipient: { email: string; role: string }) => {
    if (!canInvite || sendingBlocked || busy) return false;
    setActionError(null);
    try {
      setLink(invitationLinkFromResponse(await create.mutateAsync(recipient)));
      showSuccessToast("Invitation created.");
      return true;
    } catch (error) {
      reportActionFailure(error, "Could not create the invitation. Try again.");
      return false;
    }
  };

  const canResend = (invitation: OrganizationsOrganizationInvitation) => {
    if (!canAssignRoles && invitation.role !== "org_operator") return false;
    // The backend checks payment before it checks for an existing pending invitation.
    if (capacity?.state === "paymentBlocked") return false;
    // A pending per-seat invitation already holds its seat.
    const holdsSeat = invitation.state === "pending" && !!seats?.perSeat;
    return holdsSeat || !sendingBlocked;
  };

  const resendInvitation = async (invitation: OrganizationsOrganizationInvitation) => {
    if (!canInvite || !invitation.id || busy || !canResend(invitation)) return;
    setActionError(null);
    // Resending rotates the token. Stop displaying the previous link immediately.
    setLink(null);
    try {
      const renewedLink = invitationLinkFromResponse(await resend.mutateAsync(invitation.id));
      setLink(renewedLink);
      showSuccessToast("Invitation renewed.");
      return renewedLink;
    } catch (error) {
      reportActionFailure(error, "Could not resend the invitation. Try again.");
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
    capacity,
    canPurchaseSeats,
    canManageBilling,
    sendingBlocked,
    canResend,
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

function describeInvitationCapacity(
  seats: OrganizationsInvitationSeatSummary | undefined,
): InvitationCapacity | undefined {
  if (!seats) return undefined;
  if (seats.paymentBlocked) return { state: "paymentBlocked" };
  const occupied = Number(seats.occupied);
  if (!seats.perSeat) {
    if (seats.memberLimit == null) return { state: "unlimited" };
    return { state: occupied >= Number(seats.memberLimit) ? "memberLimit" : "available" };
  }
  if (seats.purchased != null && occupied > Number(seats.purchased)) {
    return { state: "overCapacity", members: occupied, paidSeats: Number(seats.purchased) };
  }
  if (seats.invitable != null && Number(seats.invitable) > 0) return { state: "available" };
  return { state: seats.purchasable != null && Number(seats.purchasable) === 0 ? "planMaximum" : "purchaseRequired" };
}

function blocksInvitations(capacity: InvitationCapacity | undefined): boolean {
  return !capacity || (capacity.state !== "available" && capacity.state !== "unlimited");
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
