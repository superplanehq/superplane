import { Button } from "@/components/ui/button";
import { useAccount } from "@/contexts/useAccount";
import { SUPPORT_SESSION_END_PATH } from "@/lib/supportSession";

export function SupportSessionBanner() {
  const { account } = useAccount();

  const supportSession = account?.support_session;
  if (!supportSession?.active) {
    return null;
  }

  const handleEndSupportSession = async () => {
    try {
      const response = await fetch(SUPPORT_SESSION_END_PATH, {
        method: "POST",
        credentials: "include",
      });

      if (response.ok) {
        const data = await response.json();
        window.location.href = data.redirect_url;
        return;
      }
    } catch {
      // Network error — fall through to redirect
    }

    window.location.href = "/admin";
  };

  return (
    <div className="shrink-0" role="status">
      <div className="flex items-center justify-center gap-3 bg-amber-400 px-4 py-2 text-center text-sm font-medium text-amber-900 shadow-sm">
        <span>
          Support session for <strong>{supportSession.user_name}</strong>. This session is read-only.
        </span>
        <Button
          type="button"
          size="xs"
          onClick={handleEndSupportSession}
          className="bg-amber-600 text-white hover:bg-amber-700"
        >
          End support session
        </Button>
      </div>
    </div>
  );
}
