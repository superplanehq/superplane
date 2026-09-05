import { Navigate, Route } from "react-router";

import { RequireAnyPermission, RequirePermission } from "@/components/PermissionGate";
import { RequireExperimentalFeature } from "@/components/RequireExperimentalFeature";
import { FEATURE_WORKSPACE_MODELS } from "@/lib/experimentalFeatures";
import {
  FactorySettingsAccountNotificationsPage,
  FactorySettingsAccountProfilePage,
  FactorySettingsAccountSecurityPage,
  FactorySettingsAutomationsPage,
  FactorySettingsGeneralPage,
  FactorySettingsModelsPage,
  FactorySettingsRepositoryPage,
  OrganizationSettingsOverviewPage,
} from "@/pages/factories";
import {
  OrganizationIntegrationDetailsPage,
  OrganizationIntegrationSetupPage,
} from "@/pages/factories/pages/organizationSettings/organizationSettingsRoutePages";
import { OrganizationSettingsIntegrationsPage } from "@/pages/factories/pages/organizationSettings/OrganizationSettingsIntegrationsPage";
import { OrganizationSettingsWorkspaceUsagePage } from "@/pages/factories/pages/organizationSettings/OrganizationSettingsWorkspaceUsagePage";
import {
  FactoryOrganizationApiKeyDetailPage,
  FactoryOrganizationApiKeysPage,
  FactoryOrganizationLLMModelsPage,
  FactoryOrganizationMembersPage,
  FactoryOrganizationSecretDetailPage,
  FactoryOrganizationSecretsPage,
} from "@/pages/factories/pages/settings/FactoryOrganizationSettingsPages";
import {
  AccountLinkedAccountsRedirect,
  LegacyFactorySettingsIndexRedirect,
  LegacyFactorySettingsRedirect,
  WorkspaceSpendingRedirect,
} from "@/pages/factories/pages/settings/FactorySettingsRedirects";

export const factorySettingsSectionRoutes = [
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
    key="factory-settings-account-linked-accounts"
    path="account/linked-accounts"
    element={<AccountLinkedAccountsRedirect />}
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
  <Route key="factory-settings-workspace-general" path="workspace/general" element={<FactorySettingsGeneralPage />} />,
  <Route
    key="factory-settings-workspace-repository"
    path="workspace/repository"
    element={<FactorySettingsRepositoryPage />}
  />,
  <Route
    key="factory-settings-workspace-automations"
    path="workspace/automations"
    element={<FactorySettingsAutomationsPage />}
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
  <Route key="factory-settings-workspace-spending" path="workspace/spending" element={<WorkspaceSpendingRedirect />} />,
  <Route key="factory-settings-workspace-usage" path="workspace/usage" element={<WorkspaceSpendingRedirect />} />,
  <Route
    key="factory-settings-organization-general"
    path="organization/general"
    element={<OrganizationSettingsOverviewPage />}
  />,
  <Route
    key="factory-settings-organization-members"
    path="organization/members"
    element={
      <RequirePermission resource="members" action="read">
        <FactoryOrganizationMembersPage />
      </RequirePermission>
    }
  />,
  <Route
    key="factory-settings-organization-integrations"
    path="organization/integrations"
    element={
      <RequirePermission resource="integrations" action="read">
        <OrganizationSettingsIntegrationsPage />
      </RequirePermission>
    }
  />,
  <Route
    key="factory-settings-organization-models"
    path="organization/models"
    element={
      <RequirePermission resource="org" action="read">
        <FactoryOrganizationLLMModelsPage />
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
    key="factory-settings-organization-api-keys"
    path="organization/api-keys"
    element={
      <RequirePermission resource="api_keys" action="read">
        <FactoryOrganizationApiKeysPage />
      </RequirePermission>
    }
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
    key="factory-settings-organization-secrets"
    path="organization/secrets"
    element={
      <RequirePermission resource="secrets" action="read">
        <FactoryOrganizationSecretsPage />
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
  <Route
    key="factory-settings-organization-spending"
    path="organization/spending"
    element={
      <RequirePermission resource="org" action="read">
        <OrganizationSettingsWorkspaceUsagePage />
      </RequirePermission>
    }
  />,
  <Route key="factory-settings-legacy" path="*" element={<LegacyFactorySettingsRedirect />} />,
];
