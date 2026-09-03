import type { FactoriesFactory } from "@/api-client";
import { Input } from "@/components/ui/input";
import { useAccount } from "@/contexts/useAccount";
import { useAccountOrganizations } from "@/hooks/useAccountOrganizations";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { useFactories, useFactory } from "@/hooks/useFactoryData";
import { useAvailableIntegrations } from "@/hooks/useIntegrations";
import { useOrganization } from "@/hooks/useOrganizationData";
import { usePageTitle } from "@/hooks/usePageTitle";
import {
  organizationMatchesRoute,
  organizationRouteId,
  selectedOrganizationRouteId,
  type AccountOrganization,
} from "@/lib/accountOrganizations";
import { FEATURE_WORKSPACE_MODELS } from "@/lib/experimentalFeatures";
import { IntegrationsBasePathProvider } from "@/lib/integrationSettingsPaths";
import { OrganizationSettingsPathsProvider } from "@/lib/organizationSettingsPaths";
import { cn } from "@/lib/utils";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/ui/dropdownMenu";
import { ArrowLeft, Check, ChevronDown, Search } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { Link, Navigate, NavLink, Outlet, useLocation, useNavigate, useParams, useSearchParams } from "react-router";
import {
  factoryRouteNeedsCanonicalRedirect,
  replaceFactoryKeySegment,
  resolveFactoryByKey,
} from "../../lib/factoryKeyResolution";
import {
  factoryDetailPath,
  factoryListPath,
  factorySettingsSectionPath,
  replaceOrganizationSegment,
} from "../../lib/factoryPagePaths";
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
  const { data: factory, isLoading, error } = useFactory(organizationId, factoryId);
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
          <div
            className="flex h-full min-h-0 w-full bg-background text-foreground"
            data-testid="factory-settings-layout"
          >
            <FactorySettingsSidebar
              organizationId={organizationId}
              factoryKey={factoryKey}
              factory={factory}
              factories={factories}
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
  factories,
  navQuery,
  onNavQueryChange,
  isSearching,
  searchResults,
  navGroups,
}: {
  organizationId: string;
  factoryKey: string;
  factory: FactoriesFactory;
  factories: FactoriesFactory[];
  navQuery: string;
  onNavQueryChange: (query: string) => void;
  isSearching: boolean;
  searchResults: FactorySettingsSearchResult[];
  navGroups: FactorySettingsNavGroup[];
}) {
  const { account } = useAccount();
  const { data: organization } = useOrganization(organizationId);
  const { data: organizations = [] } = useAccountOrganizations();
  const accountLabel = account?.name?.trim() || "Account";
  const organizationName = organization?.metadata?.name?.trim() || "Organization";
  const workspaceName = factory.name?.trim() || "Workspace";
  const workspaceKey = factory.key ?? factoryKey;

  return (
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
              factory={factory}
              factories={factories}
              organizations={organizations}
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
  factory,
  factories,
  organizations,
  accountLabel,
  organizationName,
  workspaceName,
  workspaceKey,
  group,
}: {
  organizationId: string;
  factoryKey: string;
  factory: FactoriesFactory;
  factories: FactoriesFactory[];
  organizations: AccountOrganization[];
  accountLabel: string;
  organizationName: string;
  workspaceName: string;
  workspaceKey: string;
  group: FactorySettingsNavGroup;
}) {
  const navigate = useNavigate();
  const { pathname } = useLocation();
  const currentOrganizationRouteId = selectedOrganizationRouteId(organizations, organizationId);

  return (
    <section data-testid={`factory-settings-${group.id}-nav`}>
      {group.id === "workspace" ? (
        <SettingsEntitySwitcher
          ariaLabel={`Switch workspace, ${workspaceName}`}
          name={workspaceName}
          helper={`Workspace · ${workspaceKey}`}
          menuLabel="Switch workspace"
          selectedId={factory.id ?? ""}
          options={factories.flatMap((item) =>
            item.id && item.key ? [{ id: item.id, name: item.name?.trim() || item.key, detail: item.key }] : [],
          )}
          onChange={(nextId) => {
            const nextFactory = factories.find((item) => item.id === nextId);
            if (!nextFactory?.key || nextFactory.key === factoryKey) {
              return;
            }
            navigate(replaceFactoryKeySegment(pathname, organizationId, factoryKey, nextFactory.key));
          }}
          testId="factory-settings-workspace-switcher"
        />
      ) : group.id === "organization" ? (
        <SettingsEntitySwitcher
          ariaLabel={`Switch organization, ${organizationName}`}
          name={organizationName}
          helper="Organization"
          menuLabel="Switch organization"
          selectedId={currentOrganizationRouteId}
          options={organizations.map((item) => ({
            id: organizationRouteId(item),
            name: item.name,
            detail: "",
          }))}
          onChange={(nextRouteId) => {
            const nextOrganization = organizations.find((item) => organizationRouteId(item) === nextRouteId);
            if (!nextOrganization || organizationMatchesRoute(nextOrganization, organizationId)) {
              return;
            }
            navigate(replaceOrganizationSegment(pathname, organizationId, organizationRouteId(nextOrganization)));
          }}
          testId="factory-settings-organization-switcher"
        />
      ) : (
        <div className="px-2.5 pb-1">
          <h2 className="text-[13px] font-medium tracking-[-0.01em] text-foreground">{accountLabel}</h2>
        </div>
      )}
      <SettingsNavItems organizationId={organizationId} factoryKey={factoryKey} items={group.items} />
    </section>
  );
}

function SettingsEntitySwitcher({
  ariaLabel,
  name,
  helper,
  menuLabel,
  selectedId,
  options,
  onChange,
  testId,
}: {
  ariaLabel: string;
  name: string;
  helper: string;
  menuLabel: string;
  selectedId: string;
  options: Array<{ id: string; name: string; detail: string }>;
  onChange: (id: string) => void;
  testId: string;
}) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          aria-label={ariaLabel}
          className="mb-1 flex w-full items-start justify-between gap-2 rounded-md px-2.5 py-1 text-left hover:bg-sidebar-accent"
          data-testid={testId}
        >
          <span className="min-w-0">
            <span className="block truncate text-[13px] font-medium tracking-[-0.01em] text-foreground">{name}</span>
            <span className="block truncate text-[11px] text-muted-foreground">{helper}</span>
          </span>
          <ChevronDown className="mt-0.5 size-3.5 shrink-0 text-muted-foreground" aria-hidden />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="w-56">
        <DropdownMenuLabel>{menuLabel}</DropdownMenuLabel>
        {options.map((option) => {
          const isCurrent = option.id === selectedId;
          return (
            <DropdownMenuItem
              key={option.id}
              onClick={() => onChange(option.id)}
              aria-checked={isCurrent}
              data-testid={`${testId}-option-${option.id}`}
            >
              <span className="min-w-0 flex-1 truncate">
                {option.name}
                {option.detail ? <span className="text-muted-foreground"> · {option.detail}</span> : null}
              </span>
              {isCurrent ? <Check className="ml-auto size-3.5 shrink-0" aria-hidden /> : null}
            </DropdownMenuItem>
          );
        })}
      </DropdownMenuContent>
    </DropdownMenu>
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
