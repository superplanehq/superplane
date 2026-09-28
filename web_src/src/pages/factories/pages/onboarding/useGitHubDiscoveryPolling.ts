import type { OrganizationsBrowserAction } from "@/api-client";
import { useRecheckGitHubInstallRequest } from "@/hooks/useRecheckGitHubInstallRequest";
import { followBrowserAction } from "@/lib/browserAction";
import { useEffect, useRef } from "react";

type BrowserActionConnection = {
  id: string;
  action: OrganizationsBrowserAction;
};

const INSTALLATION_DISCOVERY_RECHECK_INTERVAL_MS = 1_000;
const INSTALL_REQUEST_RECHECK_INTERVAL_MS = 5_000;

export function useGitHubDiscoveryPolling(args: {
  organizationId: string;
  integrationId?: string;
  enabled: boolean;
  discoveryActive: boolean;
  connectScreenOpen: boolean;
  pickerOpen: boolean;
  installRequested: boolean;
  browserAction?: BrowserActionConnection;
}) {
  const recheck = useRecheckGitHubInstallRequest(
    args.organizationId,
    args.integrationId,
    args.enabled,
    args.discoveryActive ? INSTALLATION_DISCOVERY_RECHECK_INTERVAL_MS : INSTALL_REQUEST_RECHECK_INTERVAL_MS,
  );
  const observedDiscoveryIntegration = useRef("");
  const redirectedDiscovery = useRef("");

  useEffect(() => {
    if (args.discoveryActive) {
      observedDiscoveryIntegration.current = args.integrationId ?? "";
      return;
    }
    if (!args.connectScreenOpen || !args.pickerOpen || args.installRequested || !args.browserAction?.action.url) {
      return;
    }
    if (observedDiscoveryIntegration.current !== args.browserAction.id) return;
    const redirectKey = `${args.browserAction.id}:${args.browserAction.action.url}`;
    if (redirectedDiscovery.current === redirectKey) return;
    redirectedDiscovery.current = redirectKey;
    followBrowserAction(args.browserAction.action);
  }, [
    args.browserAction,
    args.connectScreenOpen,
    args.discoveryActive,
    args.installRequested,
    args.integrationId,
    args.pickerOpen,
  ]);

  return recheck;
}
