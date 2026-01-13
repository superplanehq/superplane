import { useEffect, useRef, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useAccount } from "@/contexts/AccountContext";
import { showErrorToast } from "@/utils/toast";

type AcceptStatus = "idle" | "loading" | "error";

export default function InviteLinkAccept() {
  const { token } = useParams();
  const { account, loading } = useAccount();
  const navigate = useNavigate();
  const [status, setStatus] = useState<AcceptStatus>("idle");
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const hasSubmitted = useRef(false);

  useEffect(() => {
    if (loading || hasSubmitted.current) {
      return;
    }

    if (!token) {
      setStatus("error");
      setErrorMessage("Invalid invite link.");
      return;
    }

    if (!account) {
      const redirect = encodeURIComponent(`/invite/${token}`);
      window.location.href = `/login?redirect=${redirect}`;
      return;
    }

    hasSubmitted.current = true;

    const acceptInvite = async () => {
      try {
        setStatus("loading");
        const response = await fetch(`/api/v1/invite-links/${token}/accept`, {
          method: "POST",
          credentials: "include",
        });

        if (!response.ok) {
          const errorText = await response.text();
          throw new Error(errorText || "Unable to accept invite link.");
        }

        const data = (await response.json()) as { organization_id?: string };
        if (!data.organization_id) {
          throw new Error("Invite link response was missing organization details.");
        }

        navigate(`/${data.organization_id}`);
      } catch (err) {
        const message = err instanceof Error ? err.message : "Unable to accept invite link.";
        setStatus("error");
        setErrorMessage(message);
        showErrorToast(message);
      }
    };

    acceptInvite();
  }, [account, loading, navigate, token]);

  if (status === "error") {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900 px-6">
        <div className="max-w-md text-center">
          <h1 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Invite link not available</h1>
          <p className="mt-2 text-sm text-gray-600 dark:text-gray-400">
            {errorMessage || "This invite link is invalid or has been disabled."}
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900 px-6">
      <div className="flex flex-col items-center space-y-4">
        <div className="animate-spin rounded-full h-8 w-8 border-b border-blue-600"></div>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          {status === "loading" ? "Joining organization..." : "Preparing invite..."}
        </p>
      </div>
    </div>
  );
}
