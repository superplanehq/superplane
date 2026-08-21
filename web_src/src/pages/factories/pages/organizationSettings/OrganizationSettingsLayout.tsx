import { useOrganization } from "@/hooks/useOrganizationData";
import { usePageTitle } from "@/hooks/usePageTitle";
import { cn } from "@/lib/utils";
import { ArrowLeft } from "lucide-react";
import { Navigate, NavLink, Outlet, useParams } from "react-router";
import { factoryDetailPath, factoryListPath, organizationSettingsSectionPath } from "../../lib/factoryPagePaths";
import { useFactoriesThemeClass } from "../../lib/useFactoriesThemeClass";
import { ORGANIZATION_SETTINGS_NAV_ITEMS, type OrganizationSettingsNavItem } from "./organizationSettingsNavItems";

export function OrganizationSettingsLayout() {
  const { organizationId, factoryKey } = useParams<{ organizationId: string; factoryKey: string }>();

  if (!organizationId || !factoryKey) {
    return null;
  }

  return <OrganizationSettingsLayoutContent organizationId={organizationId} factoryKey={factoryKey} />;
}

function OrganizationSettingsLayoutContent({
  organizationId,
  factoryKey,
}: {
  organizationId: string;
  factoryKey: string;
}) {
  useFactoriesThemeClass();
  const { data: organization, isLoading, error } = useOrganization(organizationId);
  const organizationName = organization?.metadata?.name || "Organization";

  usePageTitle([organizationName, "Settings"], { enabled: !organization });

  if (!isLoading && error) {
    return <Navigate to={factoryListPath(organizationId)} replace />;
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
            to={factoryDetailPath(organizationId, factoryKey)}
            className="inline-flex h-8 items-center gap-2 rounded-md px-2.5 text-[13px] tracking-[-0.01em] text-muted-foreground hover:bg-sidebar-accent hover:text-foreground"
            data-testid="organization-settings-back"
          >
            <ArrowLeft className="size-3.5" aria-hidden />
            Back to workspace
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
          <SettingsNavGroup
            organizationId={organizationId}
            factoryKey={factoryKey}
            items={ORGANIZATION_SETTINGS_NAV_ITEMS}
          />
        </nav>
      </aside>
      <main className="flex min-h-screen min-w-0 flex-1 flex-col bg-background">
        <Outlet />
      </main>
    </div>
  );
}

function SettingsNavGroup({
  organizationId,
  factoryKey,
  items,
}: {
  organizationId: string;
  factoryKey: string;
  items: OrganizationSettingsNavItem[];
}) {
  return (
    <ul className="flex flex-col gap-0.5">
      {items.map((item) => {
        const Icon = item.Icon;
        return (
          <li key={item.id}>
            <NavLink
              to={organizationSettingsSectionPath(organizationId, factoryKey, item.id)}
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
