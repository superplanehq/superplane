import { useEffect, useRef, useState } from "react";
import { useLocation, useNavigate, useParams } from "react-router";
import { Alert, AlertDescription, AlertTitle } from "@/ui/alert";
import { useAccount } from "@/contexts/useAccount";
import { Button } from "@/components/ui/button";
import { getApiErrorMessage, getResponseErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { useReportPageReady } from "@/hooks/useReportPageReady";

type AcceptStatus = "idle" | "loading" | "error";
type InvitationAcceptance =
  | { status: "accepted"; organizationSlug: string }
  | { status: "rejected"; message: string; canSignOut: boolean };

export default function InviteLinkAccept() {
  const { token } = useParams();
  const { pathname } = useLocation();
  const { account, loading } = useAccount();
  const accountID = account?.id;
  const navigate = useNavigate();
  const [status, setStatus] = useState<AcceptStatus>("idle");
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [canSignOut, setCanSignOut] = useState(false);
  const submission = useRef<{ key: string; response: Promise<Response> } | null>(null);

  useReportPageReady(!loading && status !== "loading", {
    failed: status === "error",
  });

  useEffect(() => {
    if (loading) {
      return;
    }

    if (!token) {
      setStatus("error");
      setErrorMessage(invalidInvitationMessage);
      return;
    }

    if (!accountID) {
      const redirect = encodeURIComponent(pathname);
      window.location.href = `/login?redirect=${redirect}`;
      return;
    }

    const submissionKey = `${accountID}:${pathname}:${token}`;
    if (submission.current?.key !== submissionKey) {
      const resource = pathname.startsWith("/invitations/") ? "invitations" : "invite-links";
      submission.current = {
        key: submissionKey,
        response: fetch(`/api/v1/${resource}/${encodeURIComponent(token)}/accept`, {
          method: "POST",
          credentials: "include",
        }),
      };
    }
    const responsePromise = submission.current.response;
    let active = true;

    const acceptInvite = async () => {
      try {
        setStatus("loading");
        setErrorMessage(null);
        setCanSignOut(false);
        const originalResponse = await responsePromise;
        if (!active) {
          return;
        }
        const result = await readInvitationAcceptance(originalResponse.clone());
        if (!active) {
          return;
        }
        if (result.status === "rejected") {
          setCanSignOut(result.canSignOut);
          throw new Error(result.message);
        }

        navigate(`/${result.organizationSlug}`);
      } catch (err) {
        if (!active) {
          return;
        }
        const message = getApiErrorMessage(err, acceptanceFallback);
        setStatus("error");
        setErrorMessage(message);
        showErrorToast(message);
      }
    };

    acceptInvite();
    return () => {
      active = false;
    };
  }, [accountID, loading, navigate, pathname, token]);

  if (status === "error") {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900 px-6">
        <div className="max-w-md w-full">
          <Alert variant="destructive">
            <AlertTitle>Invitation unavailable</AlertTitle>
            <AlertDescription>
              <p>{errorMessage || "Ask an organization admin for a new invitation."}</p>
              {canSignOut && (
                <Button asChild variant="outline" className="mt-4">
                  <a href={`/logout?redirect=${encodeURIComponent(pathname)}`}>Sign out</a>
                </Button>
              )}
            </AlertDescription>
          </Alert>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900 px-6">
      <div className="flex flex-col items-center space-y-4">
        <div className="animate-spin rounded-full h-8 w-8 border-b border-blue-600"></div>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          {status === "loading" ? "Opening organization..." : "Opening invitation..."}
        </p>
      </div>
    </div>
  );
}

const acceptanceFallback = "SuperPlane could not accept this invitation. Try again.";
const invalidInvitationMessage = "This invitation link is invalid. Ask an organization admin for a new invitation.";
const invitationReasonMessages: Record<string, string> = {
  NO_SEAT_AVAILABLE: "No seat is available. Ask an organization admin to free or buy a seat.",
  MEMBER_LIMIT_REACHED: "This organization reached its member limit. Ask an organization admin.",
  INVITATION_EXPIRED: "This invitation expired. Ask an organization admin for a new invitation.",
  INVITATION_CANCELED: "This invitation was canceled. Ask an organization admin for a new invitation.",
  INVITATION_ALREADY_ACCEPTED: "This invitation was already accepted. Ask an organization admin for a new invitation.",
  INVITATION_REPLACED: "A new invitation replaced this link. Open the latest invitation email.",
  INVITATION_INVITER_REVOKED: "Ask an organization admin to send this invitation again.",
};

async function readInvitationAcceptance(response: Response): Promise<InvitationAcceptance> {
  if (!response.ok) {
    return { status: "rejected", ...(await invitationRejection(response)) };
  }
  const organization = (await response.json()) as { organization_id?: string; organization_slug?: string };
  if (!organization.organization_id || !organization.organization_slug) {
    throw new Error("SuperPlane could not open the organization. Try the invitation link again.");
  }
  return { status: "accepted", organizationSlug: organization.organization_slug };
}

async function invitationRejection(response: Response): Promise<{ message: string; canSignOut: boolean }> {
  const body: unknown = await response
    .clone()
    .json()
    .catch(() => null);
  const details = isRecord(body) && Array.isArray(body.details) ? body.details : [];
  const info = details.find(
    (detail: unknown) =>
      isRecord(detail) &&
      detail["@type"] === "type.googleapis.com/google.rpc.ErrorInfo" &&
      detail.domain === "superplane.com",
  );
  if (isRecord(info) && typeof info.reason === "string") {
    if (info.reason === "INVITATION_EMAIL_MISMATCH") {
      const invitedEmail = isRecord(info.metadata) ? info.metadata.invited_email : undefined;
      if (typeof invitedEmail === "string" && invitedEmail) {
        return {
          message: `This invitation is for ${invitedEmail}. Sign out, then sign in with the invited email address.`,
          canSignOut: true,
        };
      }
    }
    const message = invitationReasonMessages[info.reason];
    if (typeof message === "string") {
      return { message, canSignOut: false };
    }
  }
  if (response.status === 404) {
    return { message: invalidInvitationMessage, canSignOut: false };
  }
  return { message: await getResponseErrorMessage(response, acceptanceFallback), canSignOut: false };
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === "object";
}
