import { useEffect } from "react";
import { useLocation, useNavigate } from "react-router";

import { showErrorToast, showSuccessToast } from "@/lib/toast";

const AUTH_ERROR_MESSAGES: Record<string, string> = {
  linked_account_in_use: "Another member in one of your organizations already uses this GitHub account.",
  github_pr_authorization_denied:
    "GitHub authorization was canceled. The SuperPlane app continues to open your pull requests.",
  github_pr_authorization_failed: "SuperPlane could not complete the GitHub authorization. Try again.",
};

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
    const githubPrAuthorization = params.get("github_pr_authorization");
    if (!error && !linkedAccount && !githubPrAuthorization) {
      return;
    }

    const errorMessage = error ? AUTH_ERROR_MESSAGES[error] : undefined;
    if (errorMessage) {
      showErrorToast(errorMessage);
    }
    if (linkedAccount === "linked") {
      showSuccessToast("GitHub account linked.");
      void refreshAccount();
    }
    if (githubPrAuthorization === "enabled") {
      showSuccessToast("SuperPlane now opens your pull requests with your GitHub account.");
      void refreshAccount();
    }

    params.delete("auth_error");
    params.delete("linked_account");
    params.delete("github_pr_authorization");
    params.delete("provider");
    const search = params.toString();
    void navigate({ pathname: location.pathname, search: search ? `?${search}` : "" }, { replace: true });
  }, [location.pathname, location.search, navigate, refreshAccount]);

  return location;
}
