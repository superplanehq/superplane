import { Navigate, Route } from "react-router";

import { RequireAnyPermission, RequirePermission } from "@/components/PermissionGate";
import { FactorySettingsSoonPage } from "../settings/FactorySettingsSoonPage";
import { OrganizationSettingsIntegrationsPage } from "./OrganizationSettingsIntegrationsPage";
import { OrganizationSettingsWorkspaceUsagePage } from "./OrganizationSettingsWorkspaceUsagePage";
import { OrganizationSettingsOverviewPage } from "./OrganizationSettingsOverviewPage";
import { OrganizationSettingsWorkspacesPage } from "./OrganizationSettingsWorkspacesPage";
import {
  OrganizationIntegrationDetailsPage,
  OrganizationIntegrationSetupPage,
  PreserveStateNavigate,
} from "./organizationSettingsRoutePages";
import { isOrganizationSettingsComingSoon, ORGANIZATION_SETTINGS_NAV_ITEMS } from "./organizationSettingsNavItems";

export const organizationSettingsSectionRoutes = [
  <Route
    key="organization-settings-index"
    index
    element={<PreserveStateNavigate to={ORGANIZATION_SETTINGS_NAV_ITEMS[0].id} />}
  />,
  <Route key="organization-settings-general" path="general" element={<OrganizationSettingsOverviewPage />} />,
  <Route key="organization-settings-settings" path="settings" element={<PreserveStateNavigate to="general" />} />,
  <Route key="organization-settings-workspaces" path="workspaces" element={<OrganizationSettingsWorkspacesPage />} />,
  <Route
    key="organization-settings-workspace-usage"
    path="workspace-usage"
    element={
      <RequirePermission resource="org" action="read">
        <OrganizationSettingsWorkspaceUsagePage />
      </RequirePermission>
    }
  />,
  <Route
    key="organization-settings-llm-spend-redirect"
    path="llm-spend"
    element={<Navigate to="workspace-usage" replace />}
  />,
  <Route
    key="organization-settings-integrations"
    path="integrations"
    element={
      <RequirePermission resource="integrations" action="read">
        <OrganizationSettingsIntegrationsPage />
      </RequirePermission>
    }
  />,
  <Route
    key="organization-settings-integration-setup"
    path="integrations/:integrationName/setup"
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
    key="organization-settings-integration-detail"
    path="integrations/:integrationId"
    element={<OrganizationIntegrationDetailsPage />}
  />,
  ...ORGANIZATION_SETTINGS_NAV_ITEMS.filter(isOrganizationSettingsComingSoon).map((item) => (
    <Route
      key={item.id}
      path={item.id}
      element={
        <FactorySettingsSoonPage
          title={item.label}
          description={`${item.label} settings for this organization.`}
          Icon={item.Icon}
        />
      }
    />
  )),
];
