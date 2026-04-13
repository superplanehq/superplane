import { useAccount } from "@/contexts/AccountContext";
import React from "react";

const ImpersonationBanner: React.FC = () => {
  const { account } = useAccount();

  const impersonation = account?.impersonation;
  if (!impersonation?.active) {
    return null;
  }

  const handleEndImpersonation = async () => {
    try {
      const response = await fetch("/admin/api/impersonate/end", {
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
    <div className="shrink-0">
      <div className="flex items-center justify-center gap-3 bg-amber-400 px-4 py-2 text-center text-sm font-medium text-amber-900 shadow-sm">
        <span>
          You are viewing as <strong>{impersonation.user_name}</strong>
        </span>
        <button
          type="button"
          onClick={handleEndImpersonation}
          className="rounded bg-amber-600 px-3 py-0.5 text-xs font-medium text-white transition-colors hover:bg-amber-700"
        >
          Exit Impersonation
        </button>
      </div>
    </div>
  );
};

export default ImpersonationBanner;
