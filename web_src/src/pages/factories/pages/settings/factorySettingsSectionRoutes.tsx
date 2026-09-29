import { Navigate, Route } from "react-router";

import { RequireAnyPermission, RequirePermission } from "@/components/PermissionGate";
import { RequireExperimentalFeature } from "@/components/RequireExperimentalFeature";
import { FEATURE_ORGANIZATION_BYOK, FEATURE_WORKSPACE_MCP, FEATURE_WORKSPACE_SKILLS } from "@/lib/experimentalFeatures";
import {
  FactorySettingsAccountNotificationsPage,
  FactorySettingsAccountProfilePage,
  FactorySettingsAccountSecurityPage,
  FactorySettingsGeneralPage,
  FactorySettingsMCPPage,
  FactorySettingsSkillsPage,
  FactorySettingsSkillEditorPage,
  FactorySettingsRepositoryPage,
  OrganizationSettingsOverviewPage,
} from "@/pages/factories";
import {
  OrganizationIntegrationDetailsPage,
  OrganizationIntegrationSetupPage,
} from "@/pages/factories/pages/organizationSettings/organizationSettingsRoutePages";
import { OrganizationSettingsIntegrationsPage } from "@/pages/factories/pages/organizationSettings/OrganizationSettingsIntegrationsPage";
import { OrganizationSettingsWorkspaceUsagePage } from "@/pages/factories/pages/organizationSettings/OrganizationSettingsWorkspaceUsagePage";
import { OrganizationSettingsBillingPage } from "@/pages/factories/pages/organizationSettings/OrganizationSettingsBillingPage";
import { OrganizationSettingsUsagePage } from "@/pages/factories/pages/organizationSettings/OrganizationSettingsUsagePage";
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
  OrganizationUsageRedirect,
  WorkspaceAutomationsSettingsRedirect,
  WorkspaceSpendingRedirect,
} from "@/pages/factories/pages/settings/FactorySettingsRedirects";
import { LegacyAgentResourcesRedirect } from "@/pages/factories/pages/settings/LegacyAgentResourcesRedirect";

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
  <Route
    key="factory-settings-workspace-general"
    path="workspace/general"
    element={
      <RequirePermission resource="factories" action="update">
        <FactorySettingsGeneralPage />
      </RequirePermission>
    }
  />,
  <Route
    key="factory-settings-workspace-repository"
    path="workspace/repository"
    element={
      <RequirePermission resource="factories" action="update">
        <FactorySettingsRepositoryPage />
      </RequirePermission>
    }
  />,
  <Route
    key="factory-settings-workspace-agent-resources"
    path="workspace/agent-resources"
    element={<LegacyAgentResourcesRedirect />}
  />,
  <Route
    key="factory-settings-workspace-mcp"
    path="workspace/mcp"
    element={
      <RequirePermission resource="factories" action="update">
        <RequireExperimentalFeature featureId={FEATURE_WORKSPACE_MCP}>
          <FactorySettingsMCPPage />
        </RequireExperimentalFeature>
      </RequirePermission>
    }
  />,
  <Route
    key="factory-settings-workspace-skills-new"
    path="workspace/skills/new"
    element={
      <RequirePermission resource="factories" action="update">
        <RequireExperimentalFeature featureId={FEATURE_WORKSPACE_SKILLS}>
          <FactorySettingsSkillEditorPage />
        </RequireExperimentalFeature>
      </RequirePermission>
    }
  />,
  <Route
    key="factory-settings-workspace-skills-edit"
    path="workspace/skills/:resourceId"
    element={
      <RequirePermission resource="factories" action="update">
        <RequireExperimentalFeature featureId={FEATURE_WORKSPACE_SKILLS}>
          <FactorySettingsSkillEditorPage />
        </RequireExperimentalFeature>
      </RequirePermission>
    }
  />,
  <Route
    key="factory-settings-workspace-skills"
    path="workspace/skills"
    element={
      <RequirePermission resource="factories" action="update">
        <RequireExperimentalFeature featureId={FEATURE_WORKSPACE_SKILLS}>
          <FactorySettingsSkillsPage />
        </RequireExperimentalFeature>
      </RequirePermission>
    }
  />,
  // Automations moved to the factory nav Automations tab; this URL now forwards there.
  <Route
    key="factory-settings-workspace-automations"
    path="workspace/automations"
    element={<WorkspaceAutomationsSettingsRedirect />}
  />,
  <Route key="factory-settings-workspace-spending" path="workspace/spending" element={<WorkspaceSpendingRedirect />} />,
  <Route
    key="factory-settings-workspace-usage"
    path="workspace/usage"
    element={
      <RequirePermission resource="org" action="read">
        <OrganizationSettingsUsagePage />
      </RequirePermission>
    }
  />,
  <Route
    key="factory-settings-organization-general"
    path="organization/general"
    element={
      <RequirePermission resource="org" action="read">
        <OrganizationSettingsOverviewPage />
      </RequirePermission>
    }
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
        <RequireExperimentalFeature featureId={FEATURE_ORGANIZATION_BYOK}>
          <FactoryOrganizationLLMModelsPage />
        </RequireExperimentalFeature>
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
    element={
      <RequirePermission resource="integrations" action="read">
        <OrganizationIntegrationDetailsPage />
      </RequirePermission>
    }
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
    key="factory-settings-organization-billing"
    path="organization/billing"
    element={
      <RequirePermission resource="org" action="read">
        <OrganizationSettingsBillingPage />
      </RequirePermission>
    }
  />,
  <Route key="factory-settings-organization-usage" path="organization/usage" element={<OrganizationUsageRedirect />} />,
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
