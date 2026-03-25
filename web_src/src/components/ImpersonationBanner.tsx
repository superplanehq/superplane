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

    // Always redirect to /admin on failure so the user isn't stuck
    window.location.href = "/admin";
  };

  return (
    <div className="bg-amber-400 text-amber-900 px-4 py-2 text-center text-sm font-medium flex items-center justify-center gap-3 z-50">
      <span>
        You are viewing as <strong>{impersonation.user_name}</strong>
      </span>
      <button
        onClick={handleEndImpersonation}
        className="bg-amber-600 hover:bg-amber-700 text-white px-3 py-0.5 rounded text-xs font-medium transition-colors"
      >
        Exit Impersonation
      </button>
    </div>
  );
};

export default ImpersonationBanner;
