import { useEffect } from "react";
import { useLocation, useNavigate } from "react-router";

import { showErrorToast, showSuccessToast } from "@/lib/toast";

/**
 * Reads OAuth return params on Account settings pages for linked accounts.
 */
export function useAccountSettingsAuthResults(refreshAccount: () => Promise<void>) {
  const location = useLocation();
  const navigate = useNavigate();

  useEffect(() => {
    const params = new URLSearchParams(location.search);
    const error = params.get("auth_error");
    const linkedAccount = params.get("linked_account");
    if (!error && !linkedAccount) {
      return;
    }

    if (error === "linked_account_in_use") {
      showErrorToast("Another SuperPlane account already uses this GitHub account.");
    }
    if (linkedAccount === "linked") {
      showSuccessToast("GitHub account linked.");
      void refreshAccount();
    }

    params.delete("auth_error");
    params.delete("linked_account");
    params.delete("provider");
    const search = params.toString();
    void navigate({ pathname: location.pathname, search: search ? `?${search}` : "" }, { replace: true });
  }, [location.pathname, location.search, navigate, refreshAccount]);

  return location;
}
