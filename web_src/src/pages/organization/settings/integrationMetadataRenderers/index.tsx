import { dash0MetadataRenderer } from "./dash0";
import { OrganizationsIntegration } from "@/api-client";
import { IntegrationMetadataRenderer } from "./types";

const integrationMetadataRenderers: Record<string, IntegrationMetadataRenderer> = {
  dash0: dash0MetadataRenderer,
};

export function renderIntegrationMetadata(
  integrationName: string | undefined,
  integration: OrganizationsIntegration | undefined,
) {
  if (!integrationName || !integration) {
    return null;
  }

  const renderer = integrationMetadataRenderers[integrationName];
  if (!renderer) {
    return null;
  }

  return renderer({ integration });
}
