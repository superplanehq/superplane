import { OrganizationsIntegration } from "@/api-client";
import { ReactNode } from "react";

export interface IntegrationMetadataRendererContext {
  integration: OrganizationsIntegration;
}

export type IntegrationMetadataRenderer = (context: IntegrationMetadataRendererContext) => ReactNode;
