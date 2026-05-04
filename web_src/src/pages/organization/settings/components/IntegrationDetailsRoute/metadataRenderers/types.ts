import type { OrganizationsIntegration } from "@/api-client";
import type { ReactNode } from "react";

export interface IntegrationMetadataRendererContext {
  integration: OrganizationsIntegration;
}

export type IntegrationMetadataRenderer = (context: IntegrationMetadataRendererContext) => ReactNode;
