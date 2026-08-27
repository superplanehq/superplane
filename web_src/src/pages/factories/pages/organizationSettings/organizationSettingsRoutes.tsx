import { Navigate, Route } from "react-router";

import { FactorySettingsSoonPage } from "../settings/FactorySettingsSoonPage";
import { OrganizationSettingsOverviewPage } from "./OrganizationSettingsOverviewPage";
import { OrganizationSettingsWorkspacesPage } from "./OrganizationSettingsWorkspacesPage";
import { isOrganizationSettingsComingSoon, ORGANIZATION_SETTINGS_NAV_ITEMS } from "./organizationSettingsNavItems";

export const organizationSettingsSectionRoutes = [
  <Route
    key="organization-settings-index"
    index
    element={<Navigate to={ORGANIZATION_SETTINGS_NAV_ITEMS[0].id} replace />}
  />,
  <Route key="organization-settings-general" path="general" element={<OrganizationSettingsOverviewPage />} />,
  <Route key="organization-settings-settings" path="settings" element={<Navigate to="general" replace />} />,
  <Route key="organization-settings-workspaces" path="workspaces" element={<OrganizationSettingsWorkspacesPage />} />,
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
