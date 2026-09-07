import { useEffect } from "react";
import { useLocation, useNavigate } from "react-router";

import { showErrorToast, showSuccessToast } from "@/lib/toast";

/**
 * Reads OAuth return params on Account settings pages.
 * Sign-in methods use this for SSO connect. Pull request credit uses it for GitHub link.
 */
export function useAccountSettingsAuthResults(refreshAccount: () => Promise<void>) {
  const location = useLocation();
  const navigate = useNavigate();

  useEffect(() => {
    const params = new URLSearchParams(location.search);
    const error = params.get("auth_error");
    const linkResult = params.get("auth_link_result");
    const linkedAccount = params.get("linked_account");
    if (!error && !linkResult && !linkedAccount) {
      return;
    }

    const provider = params.get("provider") === "google" ? "Google" : "GitHub";
    if (error === "signin_method_in_use") {
      showErrorToast(
        `This ${provider} identity already belongs to another SuperPlane account. Delete that account first.`,
      );
    }
    if (error === "linked_account_in_use") {
      showErrorToast("Another SuperPlane account already uses this GitHub account.");
    }
    if (linkResult === "connected") {
      showSuccessToast(`${provider} connected.`);
      void refreshAccount();
    }
    if (linkedAccount === "linked") {
      showSuccessToast("GitHub account linked.");
      void refreshAccount();
    }

    params.delete("auth_error");
    params.delete("auth_link_result");
    params.delete("linked_account");
    params.delete("provider");
    const search = params.toString();
    void navigate({ pathname: location.pathname, search: search ? `?${search}` : "" }, { replace: true });
  }, [location.pathname, location.search, navigate, refreshAccount]);

  return location;
}
