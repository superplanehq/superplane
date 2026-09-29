import { createContext, createElement, useContext, type ReactNode } from "react";

export function legacySettingsIntegrationsPath(organizationId: string) {
  return `/${organizationId}/settings/integrations`;
}

export function organizationIntegrationsPath(organizationId: string) {
  return `/${organizationId}/organization/integrations`;
}

export function integrationSetupPath(basePath: string, integrationName: string) {
  return `${basePath}/${integrationName}/setup`;
}

export function integrationDetailPath(basePath: string, integrationId: string) {
  return `${basePath}/${integrationId}`;
}

const IntegrationsBasePathContext = createContext<string | undefined>(undefined);

export function IntegrationsBasePathProvider({ basePath, children }: { basePath: string; children: ReactNode }) {
  return createElement(IntegrationsBasePathContext.Provider, { value: basePath }, children);
}

export function useIntegrationsBasePath(organizationId: string) {
  return useContext(IntegrationsBasePathContext) ?? legacySettingsIntegrationsPath(organizationId);
}
