import { useFactories, useFactory } from "@/hooks/useFactoryData";
import { usePageTitle } from "@/hooks/usePageTitle";
import { cn } from "@/lib/utils";
import { ArrowLeft } from "lucide-react";
import { useEffect, useRef } from "react";
import { Navigate, NavLink, Outlet, useLocation, useParams } from "react-router";
import {
  factoryRouteNeedsCanonicalRedirect,
  replaceFactoryKeySegment,
  resolveFactoryByKey,
} from "../../lib/factoryKeyResolution";
import {
  factoryDetailPath,
  factoryListPath,
  factorySettingsSectionPath,
  type FactorySettingsScope,
} from "../../lib/factoryPagePaths";
import { IntegrationsBasePathProvider } from "@/lib/integrationSettingsPaths";
import { OrganizationSettingsPathsProvider } from "@/lib/organizationSettingsPaths";
import { useFactoriesThemeClass } from "../../lib/useFactoriesThemeClass";
import { FactorySettingsLayoutContext } from "./factorySettingsLayoutContext";
import { FACTORY_SETTINGS_NAV_GROUPS, type FactorySettingsNavItem } from "./settingsNavItems";

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
  const { data: factory, isLoading, error } = useFactory(organizationId, factoryId);

  // See the matching comment in `FactoriesLayout`: once `factory` has loaded
  // the `<Outlet/>` below mounts a leaf settings page that owns the full
  // title itself, and this baseline must stop firing so it can't clobber the
  // leaf's title on the same first-render commit.
  usePageTitle(factory?.name ? [factory.name, "Settings"] : ["Settings"], { enabled: !factory });

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

  const organizationSettingsPaths = {
    apiKeys: factorySettingsSectionPath(organizationId, factoryKey, "organization", "api-keys"),
    apiKeyDetail: (apiKeyId: string) =>
      factorySettingsSectionPath(organizationId, factoryKey, "organization", `api-keys/${apiKeyId}`),
    secrets: factorySettingsSectionPath(organizationId, factoryKey, "organization", "secrets"),
    secretDetail: (secretId: string) =>
      factorySettingsSectionPath(organizationId, factoryKey, "organization", `secrets/${secretId}`),
  };
  const integrationsPath = factorySettingsSectionPath(organizationId, factoryKey, "organization", "integrations");

  return (
    <FactorySettingsLayoutContext.Provider value={{ organizationId, factoryId, factory }}>
      <OrganizationSettingsPathsProvider paths={organizationSettingsPaths}>
        <IntegrationsBasePathProvider basePath={integrationsPath}>
          <div
            className="flex h-full min-h-0 w-full bg-background text-foreground"
            data-testid="factory-settings-layout"
          >
            <aside
              className="flex h-full w-[240px] shrink-0 flex-col border-r border-sidebar-border bg-sidebar text-sidebar-foreground"
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
              </div>
              <nav className="flex min-h-0 flex-1 flex-col gap-5 overflow-y-auto px-2 py-4">
                {FACTORY_SETTINGS_NAV_GROUPS.map((group) => (
                  <SettingsNavGroup
                    key={group.id}
                    organizationId={organizationId}
                    factoryKey={factoryKey}
                    scope={group.id}
                    title={group.label}
                    items={group.items}
                  />
                ))}
              </nav>
            </aside>
            <main
              className="flex min-h-0 min-w-0 flex-1 flex-col overflow-y-auto bg-background"
              data-testid="factory-settings-main"
            >
              <Outlet />
            </main>
          </div>
        </IntegrationsBasePathProvider>
      </OrganizationSettingsPathsProvider>
    </FactorySettingsLayoutContext.Provider>
  );
}

function SettingsNavGroup({
  organizationId,
  factoryKey,
  scope,
  title,
  items,
}: {
  organizationId: string;
  factoryKey: string;
  scope: FactorySettingsScope;
  title: string;
  items: FactorySettingsNavItem[];
}) {
  const { pathname } = useLocation();
  const activeItem = items.find((item) => pathname.includes(`/settings/${item.scope}/${item.section}`));
  const activeItemRef = useRef<HTMLAnchorElement | null>(null);

  useEffect(() => {
    activeItemRef.current?.scrollIntoView?.({ block: "nearest" });
  }, [activeItem?.id]);

  return (
    <section data-testid={`factory-settings-${scope}-nav`}>
      <h2 className="px-2.5 pb-1 text-[11px] font-medium uppercase tracking-[0.08em] text-muted-foreground">{title}</h2>
      <ul className="flex flex-col gap-0.5">
        {items.map((item) => {
          const Icon = item.Icon;
          return (
            <li key={item.id}>
              <NavLink
                ref={item.id === activeItem?.id ? activeItemRef : undefined}
                to={factorySettingsSectionPath(organizationId, factoryKey, item.scope, item.section)}
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
    </section>
  );
}
