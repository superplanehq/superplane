import type { FactoriesFactoryAgentResource } from "@/api-client";

import { connectionIsEstablished } from "./agentResourceDisplay";

export function canonicalMCPServerURL(raw: string): string {
  const trimmed = raw.trim();
  if (!trimmed) {
    return "";
  }
  try {
    const parsed = new URL(trimmed);
    parsed.hash = "";
    parsed.protocol = parsed.protocol.toLowerCase();
    parsed.hostname = parsed.hostname.toLowerCase();
    parsed.pathname = parsed.pathname.replace(/\/+$/, "");
    return parsed.toString().replace(/\/+$/, "");
  } catch {
    return trimmed.replace(/\/+$/, "").toLowerCase();
  }
}

export function connectedMCPResourceForURL<
  T extends Pick<FactoriesFactoryAgentResource, "id" | "url" | "auth" | "oauthStatus">,
>(resources: T[], url: string, exceptId?: string): T | undefined {
  const canonical = canonicalMCPServerURL(url);
  if (!canonical) {
    return undefined;
  }
  return resources.find((resource) => {
    if (exceptId && resource.id === exceptId) {
      return false;
    }
    if (!connectionIsEstablished(resource)) {
      return false;
    }
    return canonicalMCPServerURL(resource.url ?? "") === canonical;
  });
}
