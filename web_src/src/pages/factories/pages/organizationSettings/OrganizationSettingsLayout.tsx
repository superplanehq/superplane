import { useOrganization } from "@/hooks/useOrganizationData";
import { IntegrationsBasePathProvider, organizationIntegrationsPath } from "@/lib/integrationSettingsPaths";
import { usePageTitle } from "@/hooks/usePageTitle";
import { cn } from "@/lib/utils";
import { ArrowLeft } from "lucide-react";
import { Navigate, NavLink, Outlet, useLocation, useParams } from "react-router";
import {
  organizationSettingsBackPath,
  organizationSettingsSectionPath,
  type OrganizationSettingsLocationState,
} from "../../lib/factoryPagePaths";
import { useFactoriesThemeClass } from "../../lib/useFactoriesThemeClass";
import { ORGANIZATION_SETTINGS_NAV_ITEMS, type OrganizationSettingsNavItem } from "./organizationSettingsNavItems";

export function OrganizationSettingsLayout() {
  const { organizationId } = useParams<{ organizationId: string }>();

  if (!organizationId) {
    return null;
  }

  return (
    <IntegrationsBasePathProvider basePath={organizationIntegrationsPath(organizationId)}>
      <OrganizationSettingsLayoutContent organizationId={organizationId} />
    </IntegrationsBasePathProvider>
  );
}

function OrganizationSettingsLayoutContent({ organizationId }: { organizationId: string }) {
  useFactoriesThemeClass();
  const location = useLocation();
  const fromFactoryKey = (location.state as OrganizationSettingsLocationState | null)?.fromFactoryKey;
  const { data: organization, isLoading, error } = useOrganization(organizationId);
  const organizationName = organization?.metadata?.name || "Organization";

  usePageTitle([organizationName, "Settings"], { enabled: !organization });

  if (!isLoading && error) {
    return <Navigate to={organizationSettingsBackPath(organizationId, fromFactoryKey)} replace />;
  }

  if (!organization) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background text-foreground">
        <p className="text-[13px] text-muted-foreground">Loading organization…</p>
      </div>
    );
  }

  return (
    <div className="flex min-h-screen w-full bg-background text-foreground" data-testid="organization-settings-layout">
      <aside
        className="sticky top-0 flex h-screen w-[240px] shrink-0 flex-col border-r border-sidebar-border bg-sidebar text-sidebar-foreground"
        data-testid="organization-settings-sidebar"
      >
        <div className="border-b border-sidebar-border px-3 py-3">
          <NavLink
            to={organizationSettingsBackPath(organizationId, fromFactoryKey)}
            state={location.state}
            className="inline-flex h-8 items-center gap-2 rounded-md px-2.5 text-[13px] tracking-[-0.01em] text-muted-foreground hover:bg-sidebar-accent hover:text-foreground"
            data-testid="organization-settings-back"
          >
            <ArrowLeft className="size-3.5" aria-hidden />
            {fromFactoryKey ? "Back to workspace" : "Back to workspaces"}
          </NavLink>
          <p
            className="mt-2 truncate px-2.5 text-[13px] font-medium tracking-[-0.01em] text-foreground"
            title={organizationName}
            data-testid="organization-settings-name"
          >
            {organizationName}
          </p>
        </div>
        <nav className="flex min-h-0 flex-1 flex-col overflow-y-auto px-2 py-4">
          <SettingsNavGroup organizationId={organizationId} items={ORGANIZATION_SETTINGS_NAV_ITEMS} />
        </nav>
      </aside>
      <main className="flex min-h-screen min-w-0 flex-1 flex-col bg-background">
        <Outlet />
      </main>
    </div>
  );
}

function SettingsNavGroup({ organizationId, items }: { organizationId: string; items: OrganizationSettingsNavItem[] }) {
  const location = useLocation();

  return (
    <ul className="flex flex-col gap-0.5">
      {items.map((item) => {
        const Icon = item.Icon;
        return (
          <li key={item.id}>
            <NavLink
              to={organizationSettingsSectionPath(organizationId, item.id)}
              state={location.state}
              className={({ isActive }) =>
                cn(
                  "group flex items-center gap-2 rounded-md px-2.5 py-1.5 text-[13px] tracking-[-0.01em] text-foreground/80 hover:bg-sidebar-accent hover:text-foreground",
                  isActive && "bg-sidebar-accent font-medium text-foreground",
                )
              }
              data-testid={`organization-settings-nav-${item.id}`}
            >
              <Icon className="size-[15px] shrink-0 opacity-80" strokeWidth={1.75} aria-hidden />
              <span>{item.label}</span>
            </NavLink>
          </li>
        );
      })}
    </ul>
  );
}
