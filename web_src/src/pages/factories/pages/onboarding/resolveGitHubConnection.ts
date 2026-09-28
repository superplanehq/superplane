import {
  githubAccountPickerFromConnection,
  pendingGitHubAccountPicker,
  pendingGitHubBrowserActionConnection,
  pendingGitHubRequestConnection,
} from "@/lib/startDirectGitHubConnect";

import type { OnboardingPageModel } from "./useFirstRunSetupFlow";

export function resolveGitHubConnection(
  model: Pick<OnboardingPageModel, "githubConnections" | "selectedVcsConnectionId">,
  userId?: string,
  callbackIntegrationId?: string,
  discoveryIntegrationId?: string,
) {
  const requestConnection = pendingGitHubRequestConnection(
    model.githubConnections.allInstances,
    userId,
    callbackIntegrationId,
  );
  const preferredId = requestConnection?.id ?? callbackIntegrationId;
  const preferredConnection = model.githubConnections.allInstances.find((item) => item.metadata?.id === preferredId);
  const selectedConnection = model.githubConnections.readyInstances.find(
    (item) => item.metadata?.id === model.selectedVcsConnectionId,
  );
  const accountPicker = preferredId
    ? githubAccountPickerFromConnection(preferredConnection, userId)
    : (pendingGitHubAccountPicker(model.githubConnections.allInstances, userId) ??
      githubAccountPickerFromConnection(selectedConnection, userId));
  const browserActionConnection = pendingGitHubBrowserActionConnection(
    model.githubConnections.allInstances,
    userId,
    preferredId ?? discoveryIntegrationId,
  );
  return { requestConnection, accountPicker, browserActionConnection };
}
