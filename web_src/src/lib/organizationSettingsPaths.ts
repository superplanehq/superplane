import { createContext, createElement, useContext, type ReactNode } from "react";

export interface OrganizationSettingsPaths {
  apiKeys: string;
  apiKeyDetail: (apiKeyId: string) => string;
  secrets: string;
  secretDetail: (secretId: string) => string;
}

function legacyOrganizationSettingsPaths(organizationId: string): OrganizationSettingsPaths {
  const basePath = `/${organizationId}/settings`;
  return {
    apiKeys: `${basePath}/api-keys`,
    apiKeyDetail: (apiKeyId) => `${basePath}/api-keys/${apiKeyId}`,
    secrets: `${basePath}/secrets`,
    secretDetail: (secretId) => `${basePath}/secrets/${secretId}`,
  };
}

const OrganizationSettingsPathsContext = createContext<OrganizationSettingsPaths | undefined>(undefined);

export function OrganizationSettingsPathsProvider({
  paths,
  children,
}: {
  paths: OrganizationSettingsPaths;
  children: ReactNode;
}) {
  return createElement(OrganizationSettingsPathsContext.Provider, { value: paths }, children);
}

export function useOrganizationSettingsPaths(organizationId: string) {
  return useContext(OrganizationSettingsPathsContext) ?? legacyOrganizationSettingsPaths(organizationId);
}
