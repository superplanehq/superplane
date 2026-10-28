import { useEffect, useState } from "react";

import { FIRST_RUN_COPY } from "./first-run/firstRunCopy";
import { githubOnboardingMessage } from "./useFirstRunGitHub";
import { resolvedOnboardingGithubEnterprise, writeOnboardingGithubEnterprise } from "./onboardingGithubEnterprise";
import { onboardingStepPath } from "./onboardingStepPath";
import type { FirstRunBlocking } from "./useFirstRunBlockingAction";

export function connectGitHubEnterpriseAction(args: {
  blocking: FirstRunBlocking;
  locationPath: string;
  currentHost?: string;
  integrationId?: string;
  setGitHubEnterprise: (enabled: boolean, host?: string) => void;
  setGitHubEnterpriseError: (message: string | undefined) => void;
  setShowHost: (show: boolean) => void;
  connectGitHubEnterpriseApp: (host: string, returnTo: string) => Promise<boolean>;
}) {
  return (host: string) =>
    args.blocking.run("opening-github", async () => {
      args.setGitHubEnterpriseError(undefined);
      args.setGitHubEnterprise(true, host);
      if (args.integrationId && host === args.currentHost) {
        args.setShowHost(false);
        return;
      }
      const returnPath = onboardingStepPath(args.locationPath, "repo");
      try {
        await args.connectGitHubEnterpriseApp(host, returnPath);
        args.setShowHost(false);
      } catch (error) {
        args.setGitHubEnterpriseError(githubOnboardingMessage(error, FIRST_RUN_COPY.githubEnterprise.connectError));
      }
    });
}

export function submitGitHubEnterpriseOwnerAction(args: {
  blocking: FirstRunBlocking;
  locationPath: string;
  setGitHubEnterpriseError: (message: string | undefined) => void;
  submitGitHubEnterpriseOwner: (owner: string, returnTo: string) => Promise<boolean>;
}) {
  return (owner: string) =>
    args.blocking.run("opening-github", async () => {
      args.setGitHubEnterpriseError(undefined);
      const returnPath = onboardingStepPath(args.locationPath, "repo");
      try {
        await args.submitGitHubEnterpriseOwner(owner, returnPath);
      } catch (error) {
        args.setGitHubEnterpriseError(githubOnboardingMessage(error, FIRST_RUN_COPY.githubEnterprise.connectError));
      }
    });
}

export function useGitHubEnterpriseChoice(factoryId: string, githubIntegrationId: string, githubHost?: string) {
  const remembered = resolvedOnboardingGithubEnterprise({
    factoryId,
    vcsIntegrationId: githubIntegrationId,
    hostFromIntegration: githubHost,
  });
  const [githubEnterprise, setGitHubEnterpriseEnabled] = useState(remembered.enabled);
  const [githubEnterpriseHost, setGitHubEnterpriseHost] = useState(remembered.host);
  const [githubEnterpriseError, setGitHubEnterpriseError] = useState<string>();
  const [showGitHubEnterpriseHost, setShowGitHubEnterpriseHost] = useState(false);
  const setGitHubEnterprise = (enabled: boolean, host = "") => {
    writeOnboardingGithubEnterprise(factoryId, enabled, host);
    setGitHubEnterpriseEnabled(enabled);
    if (!enabled) {
      setGitHubEnterpriseHost("");
      setGitHubEnterpriseError(undefined);
      return;
    }
    if (host) setGitHubEnterpriseHost(host);
  };
  useEffect(() => {
    const restored = resolvedOnboardingGithubEnterprise({
      factoryId,
      vcsIntegrationId: githubIntegrationId,
      hostFromIntegration: githubHost,
    });
    if (!restored.enabled || githubEnterprise) return;
    writeOnboardingGithubEnterprise(factoryId, true, restored.host, restored.integrationId);
    setGitHubEnterpriseEnabled(true);
    if (restored.host) setGitHubEnterpriseHost(restored.host);
  }, [factoryId, githubEnterprise, githubHost, githubIntegrationId]);
  return {
    githubEnterprise,
    githubEnterpriseHost,
    githubEnterpriseError,
    githubEnterpriseIntegrationId: remembered.integrationId,
    showGitHubEnterpriseHost,
    setShowGitHubEnterpriseHost,
    setGitHubEnterprise,
    setGitHubEnterpriseError,
  };
}
