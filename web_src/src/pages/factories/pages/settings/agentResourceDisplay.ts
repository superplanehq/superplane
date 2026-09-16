import type {
  FactoriesFactoryAgentResource,
  FactoryAgentResourceAuth,
  FactoryAgentResourceOAuthStatus,
} from "@/api-client";

import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";

export function connectionAuthLabel(auth?: FactoryAgentResourceAuth): string {
  return auth === "AUTH_OAUTH" ? AGENT_RESOURCES_COPY.authSignIn : AGENT_RESOURCES_COPY.authHeader;
}

export function connectionStatusLabel(resource: Pick<FactoriesFactoryAgentResource, "auth" | "oauthStatus">): string {
  if (resource.auth !== "AUTH_OAUTH") {
    return AGENT_RESOURCES_COPY.statusConnected;
  }

  switch (resource.oauthStatus) {
    case "OAUTH_STATUS_CONNECTED":
      return AGENT_RESOURCES_COPY.statusConnected;
    case "OAUTH_STATUS_NEEDS_RECONNECT":
    case "OAUTH_STATUS_VENDOR_REJECTED":
      return AGENT_RESOURCES_COPY.statusReconnect;
    default:
      return AGENT_RESOURCES_COPY.statusNotConnected;
  }
}

export function connectionNeedsOAuthAction(
  resource: Pick<FactoriesFactoryAgentResource, "auth" | "oauthStatus">,
): boolean {
  if (resource.auth !== "AUTH_OAUTH") {
    return false;
  }
  return resource.oauthStatus !== "OAUTH_STATUS_CONNECTED";
}

export function skillSourceLabel(resource: Pick<FactoriesFactoryAgentResource, "repository" | "ref">): string {
  const repository = resource.repository?.trim() ?? "";
  if (!repository) {
    return "";
  }
  const ref = resource.ref?.trim();
  return ref ? `${repository}@${ref}` : repository;
}
