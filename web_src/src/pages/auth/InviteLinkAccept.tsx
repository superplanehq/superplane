import { useEffect, useRef, useState } from "react";
import { useLocation, useNavigate, useParams } from "react-router";
import { Alert, AlertDescription, AlertTitle } from "@/ui/alert";
import { useAccount } from "@/contexts/useAccount";
import { showErrorToast } from "@/lib/toast";
import { useReportPageReady } from "@/hooks/useReportPageReady";

type AcceptStatus = "idle" | "loading" | "error";

export default function InviteLinkAccept() {
  const { token } = useParams();
  const { pathname } = useLocation();
  const { account, loading } = useAccount();
  const navigate = useNavigate();
  const [status, setStatus] = useState<AcceptStatus>("idle");
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const hasSubmitted = useRef(false);

  useReportPageReady(!loading && status !== "loading", {
    failed: status === "error",
  });

  useEffect(() => {
    if (loading || hasSubmitted.current) {
      return;
    }

    if (!token) {
      setStatus("error");
      setErrorMessage("This invitation link is invalid. Ask an organization admin for a new invitation.");
      return;
    }

    if (!account) {
      const redirect = encodeURIComponent(pathname);
      window.location.href = `/login?redirect=${redirect}`;
      return;
    }

    hasSubmitted.current = true;

    const acceptInvite = async () => {
      try {
        setStatus("loading");
        const resource = pathname.startsWith("/invitations/") ? "invitations" : "invite-links";
        const response = await fetch(`/api/v1/${resource}/${encodeURIComponent(token)}/accept`, {
          method: "POST",
          credentials: "include",
        });

        if (!response.ok) {
          const rawResponse = await response.text();
          let responseMessage = rawResponse;

          try {
            const responseBody = JSON.parse(rawResponse) as { message?: string };
            if (typeof responseBody.message === "string" && responseBody.message.trim()) {
              responseMessage = responseBody.message;
            }
          } catch {
            responseMessage = rawResponse;
          }

          throw new Error(responseMessage || "SuperPlane could not accept this invitation. Try again.");
        }

        const data = (await response.json()) as { organization_id?: string; organization_slug?: string };
        if (!data.organization_id || !data.organization_slug) {
          throw new Error("SuperPlane could not open the organization. Try the invitation link again.");
        }

        navigate(`/${data.organization_slug}`);
      } catch (err) {
        const message = err instanceof Error ? err.message : "SuperPlane could not accept this invitation. Try again.";
        setStatus("error");
        setErrorMessage(message);
        showErrorToast(message);
      }
    };

    acceptInvite();
  }, [account, loading, navigate, pathname, token]);

  if (status === "error") {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900 px-6">
        <div className="max-w-md w-full">
          <Alert variant="destructive">
            <AlertTitle>Invitation unavailable</AlertTitle>
            <AlertDescription>{errorMessage || "Ask an organization admin for a new invitation."}</AlertDescription>
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
          {status === "loading" ? "Joining organization..." : "Opening invitation..."}
        </p>
      </div>
    </div>
  );
}
