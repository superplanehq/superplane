import React, { useEffect, useState } from "react";

interface ImpersonationStatus {
  active: boolean;
  user_name?: string;
  org_name?: string;
}

const ImpersonationBanner: React.FC = () => {
  const [status, setStatus] = useState<ImpersonationStatus | null>(null);

  useEffect(() => {
    const checkStatus = async () => {
      try {
        const response = await fetch("/admin/api/impersonate/status", {
          credentials: "include",
        });

        if (response.ok) {
          const data = await response.json();
          setStatus(data);
        }
      } catch {
        // Silently fail — banner just won't show
      }
    };

    checkStatus();
  }, []);

  if (!status?.active) {
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
      }
    } catch {
      // Fallback: just clear and redirect
      window.location.href = "/admin";
    }
  };

  return (
    <div className="bg-amber-400 text-amber-900 px-4 py-2 text-center text-sm font-medium flex items-center justify-center gap-3 z-50">
      <span>
        You are viewing as <strong>{status.user_name}</strong>
        {status.org_name && (
          <>
            {" "}
            in <strong>{status.org_name}</strong>
          </>
        )}
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
