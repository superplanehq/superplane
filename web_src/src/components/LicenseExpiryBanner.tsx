import { useAccount } from "@/contexts/useAccount";
import { licenseExpiryWarning } from "@/lib/license";
import { X } from "lucide-react";
import React, { useState } from "react";
import { Link } from "react-router";

const DISMISSED_KEY = "superplane:license-expiry-banner-dismissed";

const LicenseExpiryBanner: React.FC = () => {
  const { account } = useAccount();
  const license = account?.license;
  const dismissalKey = license?.expires_at ? `${license.state}:${license.expires_at}` : "";
  const [dismissed, setDismissed] = useState(() => sessionStorage.getItem(DISMISSED_KEY));

  if (!account?.installation_admin || license?.hide_expiry_banner || dismissed === dismissalKey) {
    return null;
  }

  const warning = licenseExpiryWarning(license);
  if (!warning) {
    return null;
  }

  const message = warning.expired
    ? "The Enterprise license for this installation has expired. Enterprise features are not available."
    : `The Enterprise license for this installation expires in ${warning.daysLeft} ${warning.daysLeft === 1 ? "day" : "days"}.`;

  const dismiss = () => {
    sessionStorage.setItem(DISMISSED_KEY, dismissalKey);
    setDismissed(dismissalKey);
  };

  return (
    <div className="shrink-0" data-testid="license-expiry-banner">
      <div className="flex items-center justify-center gap-3 bg-amber-100 px-4 py-2 text-center text-sm text-amber-900 dark:bg-amber-950/60 dark:text-amber-100">
        <span>{message}</span>
        <Link to="/admin/license" className="font-medium underline underline-offset-2">
          Manage license
        </Link>
        <button
          type="button"
          onClick={dismiss}
          aria-label="Dismiss license notice"
          className="rounded p-0.5 transition-colors hover:bg-amber-200 dark:hover:bg-amber-900"
        >
          <X size={14} />
        </button>
      </div>
    </div>
  );
};

export default LicenseExpiryBanner;
