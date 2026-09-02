import { Navigate, Route } from "react-router";

import { RequireAnyPermission, RequirePermission } from "@/components/PermissionGate";
import { RequireExperimentalFeature } from "@/components/RequireExperimentalFeature";
import { FEATURE_WORKSPACE_MODELS } from "@/lib/experimentalFeatures";

import { OrganizationSettingsWorkspaceUsagePage } from "../../organizationSettings/OrganizationSettingsWorkspaceUsagePage";
import { FactorySettingsAccountNotificationsPage } from "../FactorySettingsAccountNotificationsPage";
import { FactorySettingsAccountProfilePage } from "../FactorySettingsAccountProfilePage";
import { FactorySettingsAccountSecurityPage } from "../FactorySettingsAccountSecurityPage";
import { FactorySettingsModelsPage } from "../FactorySettingsModelsPage";
import { LegacyFactorySettingsIndexRedirect, LegacyFactorySettingsRedirect } from "../FactorySettingsRedirects";
import {
  OrganizationIntegrationDetailsPage,
  OrganizationIntegrationSetupPage,
} from "../../organizationSettings/organizationSettingsRoutePages";
import {
  FactoryOrganizationApiKeyDetailPage,
  FactoryOrganizationSecretDetailPage,
} from "../FactoryOrganizationSettingsPages";
import { SettingsRedesignAutomationsPage } from "./SettingsRedesignAutomationsPage";
import { SettingsRedesignApiKeysPage } from "./SettingsRedesignApiKeysPage";
import { SettingsRedesignIntegrationsPage } from "./SettingsRedesignIntegrationsPage";
import { SettingsRedesignOrganizationGeneralPage } from "./SettingsRedesignOrganizationGeneral";
import { SettingsRedesignOrganizationMembersPage } from "./SettingsRedesignOrganizationPages";
import { SettingsRedesignSecretsPage } from "./SettingsRedesignSecretsPage";
import {
  SettingsRedesignWorkspaceGeneralPage,
  SettingsRedesignWorkspaceRepositoryPage,
  SettingsRedesignWorkspaceSpendingPage,
} from "./SettingsRedesignWorkspacePages";

/** Factory settings routes. Account and Models stay on the existing pages. */
export const factorySettingsRedesignRoutes = [
  <Route key="factory-settings-index" index element={<LegacyFactorySettingsIndexRedirect />} />,
  <Route
    key="factory-settings-account-general"
    path="account/general"
    element={<Navigate to="../profile" replace />}
  />,
  <Route
    key="factory-settings-account-profile"
    path="account/profile"
    element={<FactorySettingsAccountProfilePage />}
  />,
  <Route
    key="factory-settings-account-security"
    path="account/security"
    element={<FactorySettingsAccountSecurityPage />}
  />,
  <Route
    key="factory-settings-account-notifications"
    path="account/notifications"
    element={<FactorySettingsAccountNotificationsPage />}
  />,
  <Route
    key="factory-settings-workspace-general"
    path="workspace/general"
    element={<SettingsRedesignWorkspaceGeneralPage />}
  />,
  <Route
    key="factory-settings-workspace-repository"
    path="workspace/repository"
    element={<SettingsRedesignWorkspaceRepositoryPage />}
  />,
  <Route
    key="factory-settings-workspace-automations"
    path="workspace/automations"
    element={<SettingsRedesignAutomationsPage />}
  />,
  <Route
    key="factory-settings-workspace-models"
    path="workspace/models"
    element={
      <RequireExperimentalFeature featureId={FEATURE_WORKSPACE_MODELS}>
        <FactorySettingsModelsPage />
      </RequireExperimentalFeature>
    }
  />,
  <Route
    key="factory-settings-workspace-spending"
    path="workspace/spending"
    element={<SettingsRedesignWorkspaceSpendingPage />}
  />,
  <Route
    key="factory-settings-organization-general"
    path="organization/general"
    element={<SettingsRedesignOrganizationGeneralPage />}
  />,
  <Route
    key="factory-settings-organization-members"
    path="organization/members"
    element={
      <RequirePermission resource="members" action="read">
        <SettingsRedesignOrganizationMembersPage />
      </RequirePermission>
    }
  />,
  <Route
    key="factory-settings-organization-integrations"
    path="organization/integrations"
    element={
      <RequirePermission resource="integrations" action="read">
        <SettingsRedesignIntegrationsPage />
      </RequirePermission>
    }
  />,
  <Route
    key="factory-settings-organization-api-keys"
    path="organization/api-keys"
    element={
      <RequirePermission resource="api_keys" action="read">
        <SettingsRedesignApiKeysPage />
      </RequirePermission>
    }
  />,
  <Route
    key="factory-settings-organization-secrets"
    path="organization/secrets"
    element={
      <RequirePermission resource="secrets" action="read">
        <SettingsRedesignSecretsPage />
      </RequirePermission>
    }
  />,
  <Route
    key="factory-settings-organization-spending"
    path="organization/spending"
    element={
      <RequirePermission resource="org" action="read">
        <OrganizationSettingsWorkspaceUsagePage />
      </RequirePermission>
    }
  />,
  <Route
    key="factory-settings-organization-integration-setup"
    path="organization/integrations/:integrationName/setup"
    element={
      <RequireAnyPermission
        checks={[
          { resource: "integrations", action: "create" },
          { resource: "integrations", action: "update" },
        ]}
      >
        <OrganizationIntegrationSetupPage />
      </RequireAnyPermission>
    }
  />,
  <Route
    key="factory-settings-organization-integration-detail"
    path="organization/integrations/:integrationId"
    element={<OrganizationIntegrationDetailsPage />}
  />,
  <Route
    key="factory-settings-organization-api-key-detail"
    path="organization/api-keys/:id"
    element={
      <RequirePermission resource="api_keys" action="read">
        <FactoryOrganizationApiKeyDetailPage />
      </RequirePermission>
    }
  />,
  <Route
    key="factory-settings-organization-secret-detail"
    path="organization/secrets/:secretId"
    element={
      <RequirePermission resource="secrets" action="read">
        <FactoryOrganizationSecretDetailPage />
      </RequirePermission>
    }
  />,
  <Route key="factory-settings-legacy" path="*" element={<LegacyFactorySettingsRedirect />} />,
];
