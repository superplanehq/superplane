import type { FactoriesFactory } from "@/api-client";
import { Input } from "@/components/ui/input";
import { useAccount } from "@/contexts/useAccount";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { useFactories, useFactory } from "@/hooks/useFactoryData";
import { useAvailableIntegrations } from "@/hooks/useIntegrations";
import { useOrganization } from "@/hooks/useOrganizationData";
import { usePageTitle } from "@/hooks/usePageTitle";
import { FEATURE_WORKSPACE_MODELS } from "@/lib/experimentalFeatures";
import { IntegrationsBasePathProvider } from "@/lib/integrationSettingsPaths";
import { OrganizationSettingsPathsProvider } from "@/lib/organizationSettingsPaths";
import { cn } from "@/lib/utils";
import { Search } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { Link, Navigate, NavLink, Outlet, useLocation, useParams, useSearchParams } from "react-router";
import {
  factoryRouteNeedsCanonicalRedirect,
  replaceFactoryKeySegment,
  resolveFactoryByKey,
} from "../../lib/factoryKeyResolution";
import { FactoriesSidebar } from "../../layout/FactoriesSidebar";
import { factoryListPath, factorySettingsSectionPath } from "../../lib/factoryPagePaths";
import { useFactoriesThemeClass } from "../../lib/useFactoriesThemeClass";
import { FactorySettingsLayoutContext } from "./factorySettingsLayoutContext";
import { type FactorySettingsNavGroup, type FactorySettingsNavItem } from "./settingsNavItems";
import {
  buildFactorySettingsSearchIndex,
  factorySettingsSearchResultPath,
  searchFactorySettings,
  type FactorySettingsSearchResult,
} from "./settingsSearch";
import { useFactorySettingsNavGroups } from "./useFactorySettingsNavGroups";
import { useFactorySettingsSectionScroll } from "./useFactorySettingsSectionScroll";

/** Nav item id for the in-progress workspace Models settings page, gated behind `FEATURE_WORKSPACE_MODELS`. */
const WORKSPACE_MODELS_NAV_ITEM_ID = "workspace-models";

/**
 * Drops the Models nav item when the workspace-models experimental feature is
 * off, and skips any group left with no items. The source groups stay static
 * so other consumers (e.g. route lookups) keep seeing the full, approved list.
 */
function visibleFactorySettingsNavGroups(
  groups: FactorySettingsNavGroup[],
  modelsEnabled: boolean,
): FactorySettingsNavGroup[] {
  if (modelsEnabled) {
    return groups;
  }

  return groups
    .map((group) => ({
      ...group,
      items: group.items.filter((item) => item.id !== WORKSPACE_MODELS_NAV_ITEM_ID),
    }))
    .filter((group) => group.items.length > 0);
}

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
      factories={factories}
    />
  );
}

function FactorySettingsLayoutContent({
  organizationId,
  factoryId,
  factoryKey,
  factories,
}: {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  factories: FactoriesFactory[];
}) {
  useFactoriesThemeClass();
  useFactorySettingsSectionScroll();
  const settingsNavGroups = useFactorySettingsNavGroups();
  const { data: describedFactory, isLoading, error } = useFactory(organizationId, factoryId);
  const factory = describedFactory ?? factories.find((item) => item.id === factoryId);
  const { has: hasExperimentalFeature } = useExperimentalFeature(organizationId);
  const { data: availableIntegrations = [] } = useAvailableIntegrations();
  const [navQuery, setNavQuery] = useState("");
  const navGroups = visibleFactorySettingsNavGroups(
    settingsNavGroups,
    hasExperimentalFeature(FEATURE_WORKSPACE_MODELS),
  );
  const searchIndex = useMemo(
    () =>
      buildFactorySettingsSearchIndex({
        navGroups,
        integrations: availableIntegrations,
      }),
    [availableIntegrations, navGroups],
  );
  const searchResults = useMemo(() => searchFactorySettings(searchIndex, navQuery), [navQuery, searchIndex]);
  const isSearching = navQuery.trim().length > 0;

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
          <div className="flex h-screen w-full bg-background text-foreground" data-testid="factory-settings-layout">
            <FactoriesSidebar
              organizationId={organizationId}
              factoryKey={factoryKey}
              factory={factory}
              factories={factories}
            />
            <FactorySettingsSidebar
              organizationId={organizationId}
              factoryKey={factoryKey}
              factory={factory}
              navQuery={navQuery}
              onNavQueryChange={setNavQuery}
              isSearching={isSearching}
              searchResults={searchResults}
              navGroups={navGroups}
            />
            {/*
              Gray canvas behind white panels, like the Velocity report. Dark
              mode keeps the darker page, because the factories theme paints
              the sidebar and the panels the same color.
            */}
            <main
              className="flex min-h-0 min-w-0 flex-1 flex-col overflow-y-auto bg-sidebar dark:bg-background"
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

function FactorySettingsSidebar({
  organizationId,
  factoryKey,
  factory,
  navQuery,
  onNavQueryChange,
  isSearching,
  searchResults,
  navGroups,
}: {
  organizationId: string;
  factoryKey: string;
  factory: FactoriesFactory;
  navQuery: string;
  onNavQueryChange: (query: string) => void;
  isSearching: boolean;
  searchResults: FactorySettingsSearchResult[];
  navGroups: FactorySettingsNavGroup[];
}) {
  const { account } = useAccount();
  const { data: organization } = useOrganization(organizationId);
  const accountLabel = account?.name?.trim() || "Account";
  const organizationName = organization?.metadata?.name?.trim() || "Organization";
  const workspaceName = factory.name?.trim() || "Workspace";
  const workspaceKey = factory.key ?? factoryKey;

  return (
    <aside
      className="flex h-full w-[240px] shrink-0 flex-col border-r border-sidebar-border bg-sidebar text-sidebar-foreground"
      data-testid="factory-settings-sidebar"
    >
      <div className="px-3 pt-3">
        <label className="sr-only" htmlFor="factory-settings-find">
          Find settings
        </label>
        <div className="relative">
          <Search
            className="pointer-events-none absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-muted-foreground"
            aria-hidden
          />
          <Input
            id="factory-settings-find"
            data-testid="factory-settings-find"
            type="search"
            value={navQuery}
            onChange={(event) => onNavQueryChange(event.target.value)}
            placeholder="Find settings"
            className="h-8 border-border bg-background pl-8 text-[13px] text-foreground shadow-none placeholder:text-muted-foreground focus:border-ring dark:bg-background"
          />
        </div>
      </div>
      <nav className="flex min-h-0 flex-1 flex-col gap-5 overflow-y-auto px-2 py-4">
        {isSearching ? (
          <SettingsSearchResults organizationId={organizationId} factoryKey={factoryKey} results={searchResults} />
        ) : (
          navGroups.map((group) => (
            <SettingsNavGroup
              key={group.id}
              organizationId={organizationId}
              factoryKey={factoryKey}
              accountLabel={accountLabel}
              organizationName={organizationName}
              workspaceName={workspaceName}
              workspaceKey={workspaceKey}
              group={group}
            />
          ))
        )}
      </nav>
    </aside>
  );
}

function SettingsSearchResults({
  organizationId,
  factoryKey,
  results,
}: {
  organizationId: string;
  factoryKey: string;
  results: FactorySettingsSearchResult[];
}) {
  const { pathname } = useLocation();
  const [searchParams] = useSearchParams();
  const activeSection = searchParams.get("section");

  if (results.length === 0) {
    return (
      <p className="px-2.5 text-[13px] text-muted-foreground" data-testid="factory-settings-find-empty">
        No matching settings.
      </p>
    );
  }

  return (
    <ul className="flex flex-col gap-0.5" data-testid="factory-settings-search-results">
      {results.map((result) => {
        const href = factorySettingsSearchResultPath(organizationId, factoryKey, result, factorySettingsSectionPath);
        const pathMatches = pathname.includes(`/settings/${result.scope}/${result.section}`);
        const isActive = pathMatches && (result.anchor ? activeSection === result.anchor : !activeSection);

        return (
          <li key={result.id}>
            <Link
              to={href}
              className={cn(
                "flex flex-col gap-0.5 rounded-md px-2.5 py-1.5 text-left hover:bg-sidebar-accent hover:text-foreground",
                isActive && "bg-sidebar-accent font-medium text-foreground",
              )}
              data-testid={`factory-settings-search-${result.id}`}
            >
              <span className="text-[13px] tracking-[-0.01em] text-foreground/90">{result.title}</span>
              <span className="text-[11px] text-muted-foreground">{result.breadcrumb.join(" › ")}</span>
            </Link>
          </li>
        );
      })}
    </ul>
  );
}

function SettingsNavGroup({
  organizationId,
  factoryKey,
  accountLabel,
  organizationName,
  workspaceName,
  workspaceKey,
  group,
}: {
  organizationId: string;
  factoryKey: string;
  accountLabel: string;
  organizationName: string;
  workspaceName: string;
  workspaceKey: string;
  group: FactorySettingsNavGroup;
}) {
  return (
    <section data-testid={`factory-settings-${group.id}-nav`}>
      {group.id === "workspace" ? (
        <SettingsNavGroupHeading
          name={workspaceName}
          helper={`Workspace · ${workspaceKey}`}
          testId="factory-settings-workspace-heading"
        />
      ) : group.id === "organization" ? (
        <SettingsNavGroupHeading
          name={organizationName}
          helper="Organization"
          testId="factory-settings-organization-heading"
        />
      ) : (
        <SettingsNavGroupHeading name={accountLabel} testId="factory-settings-account-heading" />
      )}
      <SettingsNavItems organizationId={organizationId} factoryKey={factoryKey} items={group.items} />
    </section>
  );
}

function SettingsNavGroupHeading({
  name,
  helper,
  testId,
}: {
  name: string;
  helper?: string;
  testId?: string;
}) {
  return (
    <div className="mb-1 px-2.5 py-1" data-testid={testId}>
      <h2 className="truncate text-[13px] font-medium tracking-[-0.01em] text-foreground">{name}</h2>
      {helper ? <p className="truncate text-[11px] text-muted-foreground">{helper}</p> : null}
    </div>
  );
}

function SettingsNavItems({
  organizationId,
  factoryKey,
  items,
}: {
  organizationId: string;
  factoryKey: string;
  items: FactorySettingsNavItem[];
}) {
  const { pathname } = useLocation();
  const activeItem = items.find((item) => pathname.includes(`/settings/${item.scope}/${item.section}`));
  const activeItemRef = useRef<HTMLAnchorElement | null>(null);

  useEffect(() => {
    activeItemRef.current?.scrollIntoView?.({ block: "nearest" });
  }, [activeItem?.id]);

  return (
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
  );
}
