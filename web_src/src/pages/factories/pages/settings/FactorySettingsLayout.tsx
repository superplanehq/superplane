import { useAccount } from "@/contexts/useAccount";
import { useFactories, useFactory } from "@/hooks/useFactoryData";
import { useOrganization } from "@/hooks/useOrganizationData";
import { usePageTitle } from "@/hooks/usePageTitle";
import { cn } from "@/lib/utils";
import { ArrowLeft } from "lucide-react";
import { Navigate, NavLink, Outlet, useLocation, useParams } from "react-router";
import {
  factoryRouteNeedsCanonicalRedirect,
  replaceFactoryKeySegment,
  resolveFactoryByKey,
} from "../../lib/factoryKeyResolution";
import { factoryDetailPath, factoryListPath, factorySettingsSectionPath } from "../../lib/factoryPagePaths";
import { SidebarUserMenu } from "../../layout/SidebarUserMenu";
import { useFactoriesThemeClass } from "../../lib/useFactoriesThemeClass";
import { FactorySettingsLayoutContext } from "./factorySettingsLayoutContext";
import { FACTORY_SETTINGS_NAV_ITEMS } from "./settingsNavItems";

export function FactorySettingsLayout() {
  const { organizationId, factoryKey } = useParams<{ organizationId: string; factoryKey: string }>();

  if (!organizationId || !factoryKey) {
    return null;
  }

  return <FactorySettingsLayoutResolver organizationId={organizationId} factoryKey={factoryKey} />;
}

/** Mirrors `FactoriesLayoutResolver` — resolves `:factoryKey` before rendering the settings shell. */
function FactorySettingsLayoutResolver({ organizationId, factoryKey }: { organizationId: string; factoryKey: string }) {
  const location = useLocation();
  const {
    data: factories = [],
    isLoading: factoriesLoading,
    isFetching: factoriesFetching,
  } = useFactories(organizationId);
  const resolution = resolveFactoryByKey(factories, factoryKey, factoriesLoading || factoriesFetching);

  if (factoryRouteNeedsCanonicalRedirect(resolution, factoryKey)) {
    const target = replaceFactoryKeySegment(location.pathname, organizationId, factoryKey, resolution.factory!.key!);
    return <Navigate to={`${target}${location.search}`} replace />;
  }

  if (resolution.status === "not-found") {
    return <Navigate to={factoryListPath(organizationId)} replace />;
  }

  if (resolution.status === "loading" || !resolution.factory?.id) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background text-foreground">
        <p className="text-[13px] text-muted-foreground">Loading settings…</p>
      </div>
    );
  }

  return (
    <FactorySettingsLayoutContent
      organizationId={organizationId}
      factoryId={resolution.factory.id}
      factoryKey={resolution.factory.key ?? factoryKey}
    />
  );
}

function FactorySettingsLayoutContent({
  organizationId,
  factoryId,
  factoryKey,
}: {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
}) {
  useFactoriesThemeClass();
  const { account } = useAccount();
  const { data: organization } = useOrganization(organizationId);
  const { data: factory, isLoading, error } = useFactory(organizationId, factoryId);

  usePageTitle(factory?.name ? [factory.name, "Settings"] : ["Settings"]);

  if (!isLoading && error) {
    return <Navigate to={factoryListPath(organizationId)} replace />;
  }

  if (!factory) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background text-foreground">
        <p className="text-[13px] text-muted-foreground">Loading settings…</p>
      </div>
    );
  }

  const workspaceGroup = FACTORY_SETTINGS_NAV_ITEMS.filter((item) => item.group === "workspace");
  const governanceGroup = FACTORY_SETTINGS_NAV_ITEMS.filter((item) => item.group === "governance");

  return (
    <FactorySettingsLayoutContext.Provider value={{ organizationId, factoryId, factory }}>
      <div className="flex min-h-screen w-full bg-background text-foreground" data-testid="factory-settings-layout">
        <aside
          className="sticky top-0 flex h-screen w-[240px] shrink-0 flex-col border-r border-sidebar-border bg-sidebar text-sidebar-foreground"
          data-testid="factory-settings-sidebar"
        >
          <div className="border-b border-sidebar-border px-3 py-3">
            <NavLink
              to={factoryDetailPath(organizationId, factoryKey)}
              className="inline-flex h-8 items-center gap-2 rounded-md px-2.5 text-[13px] tracking-[-0.01em] text-muted-foreground hover:bg-sidebar-accent hover:text-foreground"
              data-testid="factory-settings-back"
            >
              <ArrowLeft className="size-3.5" aria-hidden />
              Back to workspace
            </NavLink>
            <p
              className="mt-2 truncate px-2.5 text-[13px] font-medium tracking-[-0.01em] text-foreground"
              title={factory.name ?? undefined}
              data-testid="factory-settings-workspace-name"
            >
              {factory.name}
            </p>
          </div>
          <nav className="flex flex-1 flex-col gap-4 px-2 py-4">
            <SettingsNavGroup organizationId={organizationId} factoryKey={factoryKey} items={workspaceGroup} />
            <SettingsNavGroup organizationId={organizationId} factoryKey={factoryKey} items={governanceGroup} />
          </nav>
          <SidebarUserMenu
            organizationId={organizationId}
            userName={account?.name ?? "You"}
            userEmail={account?.email}
            userAvatarUrl={account?.avatar_url}
            organizationName={organization?.metadata?.name ?? "Organization"}
          />
        </aside>
        <main className="flex min-h-screen min-w-0 flex-1 flex-col bg-background">
          <Outlet />
        </main>
      </div>
    </FactorySettingsLayoutContext.Provider>
  );
}

function SettingsNavGroup({
  organizationId,
  factoryKey,
  items,
}: {
  organizationId: string;
  factoryKey: string;
  items: typeof FACTORY_SETTINGS_NAV_ITEMS;
}) {
  return (
    <ul className="flex flex-col gap-0.5">
      {items.map((item) => {
        const Icon = item.Icon;
        return (
          <li key={item.id}>
            <NavLink
              to={factorySettingsSectionPath(organizationId, factoryKey, item.id)}
              className={({ isActive }) =>
                cn(
                  "group flex items-center gap-2 rounded-md px-2.5 py-1.5 text-[13px] tracking-[-0.01em] text-foreground/80 hover:bg-sidebar-accent hover:text-foreground",
                  isActive && "bg-sidebar-accent font-medium text-foreground",
                )
              }
              data-testid={`factory-settings-nav-${item.id}`}
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
