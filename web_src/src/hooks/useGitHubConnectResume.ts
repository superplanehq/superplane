import { useEffect, useRef } from "react";
import { useLocation, useNavigate } from "react-router";

import {
  githubConnectResumeError,
  isGitHubConnectResumeReturn,
  stripGitHubConnectResumeParams,
} from "@/lib/githubIdentityLinkGate";
import { showErrorToast } from "@/lib/toast";

/**
 * Continues a hosted GitHub connect after the identity link flow returns.
 * Strips the auth result params from the URL and fires the callback once.
 * The callback runs on a failed link too, so the connect falls through to
 * the plain GitHub install flow.
 */
export function useGitHubConnectResume(onResume: () => void) {
  const location = useLocation();
  const navigate = useNavigate();
  const resumed = useRef(false);

  useEffect(() => {
    if (resumed.current || !isGitHubConnectResumeReturn(location.search)) {
      return;
    }
    resumed.current = true;

    if (githubConnectResumeError(location.search) === "linked_account_in_use") {
      showErrorToast("Another member in one of your organizations already uses this GitHub account.");
    }

    const search = stripGitHubConnectResumeParams(location.search);
    void navigate({ pathname: location.pathname, search: search ? `?${search}` : "" }, { replace: true });
    onResume();
  }, [location.pathname, location.search, navigate, onResume]);
}
