import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useEffect, useState, type ComponentType } from "react";
import { MemoryRouter, Navigate, Outlet, Route, Routes, useParams } from "react-router-dom";

import { writeCanvasAgentSidebarOpen } from "@/components/CanvasToolSidebar/useCanvasToolSidebarState";
import { RequireExperimentalFeature } from "@/components/RequireExperimentalFeature";
import { AccountProvider } from "@/contexts/AccountProvider";
import { PermissionsProvider } from "@/contexts/PermissionsProvider";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import { agentChatKeys } from "@/hooks/useAgentChats";
import { FEATURE_FACTORIES } from "@/lib/experimentalFeatures";
import { setAgentSuggestions } from "@/lib/agentSuggestionsContext";
import { AppPage } from "@/pages/app";
import { STORYBOOK_AGENT_MESSAGES_UPDATED_EVENT } from "@/pages/app/__fixtures__/agentChatResponses";
import { canvasAppIds, type CanvasAppFixture } from "@/pages/app/__fixtures__/handlers";
import {
  AutomationsPage,
  CreateWorkOrderPage,
  FactoriesIndexPage,
  FactoriesLayout,
  FactoryAppCanvasPage,
  FactoryLineEditPage,
  FactorySettingsGeneralPage,
  FactorySettingsLayout,
  FactorySettingsSoonPage,
  FACTORY_SETTINGS_NAV_ITEMS,
  LinesPage,
  MissionsPage,
  OverviewPage,
  VelocityPage,
  WikiPage,
  WorkOrderDetailPage,
  WorkOrdersPage,
} from "@/pages/factories";
import type { FactoriesFixture } from "@/pages/factories/__fixtures__/handlers";
import { createFactoryLinePath, editFactoryLinePath } from "@/pages/factories/lib/factoryPagePaths";
import { HomePage } from "@/pages/home";
import { homePageIds, type HomePageFixture } from "@/pages/home/__fixtures__/handlers";
import { NewAppPage } from "@/pages/home/NewAppPage";
import type { AgentSuggestion } from "@/ui/CanvasPage";
import { TooltipProvider } from "@/ui/tooltip";

import { createOrgWorkspaceFixtureFetch } from "./createOrgWorkspaceFixtureFetch";

interface FixtureFetchState {
  original: typeof fetch;
  delegate: typeof fetch | null;
}

const FIXTURE_FETCH_KEY = "__orgWorkspaceFixtureFetch";

/**
 * Installs a permanent `window.fetch` wrapper (once) that delegates to the
 * currently active fixture fetch, or to the real network when no story is
 * mounted. Same race-safe delegate pattern as the page harnesses.
 */
function fixtureFetchState(): FixtureFetchState {
  const holder = window as unknown as Record<string, FixtureFetchState | undefined>;
  let state = holder[FIXTURE_FETCH_KEY];
  if (!state) {
    const created: FixtureFetchState = { original: window.fetch.bind(window), delegate: null };
    holder[FIXTURE_FETCH_KEY] = created;
    window.fetch = ((input: RequestInfo | URL, init?: RequestInit) =>
      (created.delegate ?? created.original)(input, init)) as typeof fetch;
    state = created;
  }
  return state;
}

/** Storybook-only route swaps. App routes ignore this. */
export interface OrgWorkspacePageOverrides {
  wiki?: ComponentType;
}

export interface OrgWorkspaceHarnessProps {
  /** Where to land when the story mounts. */
  startAt?: "home" | "app";
  /** Path under the org when `startAt` is `home`, e.g. `apps/new` or `workspaces/...`. */
  pathSuffix?: string;
  /** Query string for the app route (without leading `?`). */
  appQuery?: string;
  /**
   * When true, opens the canvas agent sidebar via localStorage before mount.
   * Always written (true/false) so story switches do not leak open state.
   */
  openAgentSidebar?: boolean;
  homeFixture?: HomePageFixture;
  appFixture?: CanvasAppFixture;
  /** Backing fixture for factory pages (list, detail, orders, lines, apps). */
  factoriesFixture?: FactoriesFixture;
  /** Storybook-only: seed post-install Agent improvement suggestions for the canvas. */
  agentSuggestions?: AgentSuggestion[];
  /** Storybook-only: replace selected factory page elements (e.g. wiki wireframe). */
  pageOverrides?: OrgWorkspacePageOverrides;
}

interface FixtureFetchOptions {
  canvasId: string;
  openAgentSidebar: boolean;
  homeFixture?: HomePageFixture;
  appFixture?: CanvasAppFixture;
  factoriesFixture?: FactoriesFixture;
  agentSuggestions?: AgentSuggestion[];
}

function useOrgWorkspaceFixtureFetch(options: FixtureFetchOptions) {
  const { canvasId, openAgentSidebar, homeFixture, appFixture, factoriesFixture, agentSuggestions } = options;
  const [fixtureFetch] = useState(() => {
    // Persist before AppPage reads the preference in useState initializers.
    writeCanvasAgentSidebarOpen(canvasId, openAgentSidebar);
    if (agentSuggestions?.length) {
      setAgentSuggestions(canvasId, agentSuggestions);
    }
    const state = fixtureFetchState();
    const impl = createOrgWorkspaceFixtureFetch(state.original, { homeFixture, appFixture, factoriesFixture });
    state.delegate = impl;
    return impl;
  });

  useEffect(() => {
    writeCanvasAgentSidebarOpen(canvasId, openAgentSidebar);
    const state = fixtureFetchState();
    if (state.delegate === null) {
      state.delegate = fixtureFetch;
    }
    return () => {
      if (state.delegate === fixtureFetch) {
        state.delegate = null;
      }
    };
  }, [canvasId, fixtureFetch, openAgentSidebar]);

  return fixtureFetch;
}

/**
 * Resolve org and canvas IDs from whichever fixture the story provided. Kept
 * as a plain helper so the harness component stays under the complexity budget.
 */
function resolveWorkspaceIds(
  homeFixture: HomePageFixture | undefined,
  appFixture: CanvasAppFixture | undefined,
  factoriesFixture: FactoriesFixture | undefined,
) {
  const orgId =
    homeFixture?.organizationId ??
    appFixture?.organizationId ??
    factoriesFixture?.organizationId ??
    homePageIds.organizationId;
  const canvasId = appFixture?.canvasId ?? canvasAppIds.canvasId;
  return { orgId, canvasId };
}

function factoryRoute(element: React.ReactNode) {
  return <RequireExperimentalFeature featureId={FEATURE_FACTORIES}>{element}</RequireExperimentalFeature>;
}

function HarnessLegacyAutomationsNewLineRedirect() {
  const { organizationId, factoryId } = useParams<{ organizationId: string; factoryId: string }>();
  if (!organizationId || !factoryId) {
    return <Navigate to="/" replace />;
  }
  return <Navigate to={createFactoryLinePath(organizationId, factoryId)} replace />;
}

function HarnessLegacyAutomationsLineEditRedirect() {
  const { organizationId, factoryId, lineId } = useParams<{
    organizationId: string;
    factoryId: string;
    lineId: string;
  }>();
  if (!organizationId || !factoryId || !lineId) {
    return <Navigate to="/" replace />;
  }
  return <Navigate to={editFactoryLinePath(organizationId, factoryId, lineId)} replace />;
}

function OrgWorkspaceRoutes({ pageOverrides }: { pageOverrides?: OrgWorkspacePageOverrides }) {
  const WikiRoutePage = pageOverrides?.wiki ?? WikiPage;

  return (
    <Routes>
      <Route
        path=":organizationId"
        element={
          <PermissionsProvider>
            <Outlet />
          </PermissionsProvider>
        }
      >
        <Route index element={<HomePage />} />
        <Route path="apps/new" element={<NewAppPage />} />
        <Route path="apps/:appId" element={<AppPage />} />
        <Route path="workspaces">
          <Route index element={factoryRoute(<FactoriesIndexPage />)} />
          <Route path=":factoryId" element={factoryRoute(<FactoriesLayout />)}>
            <Route index element={<Navigate to="overview" replace />} />
            <Route path="overview" element={<OverviewPage />} />
            <Route path="missions" element={<MissionsPage />} />
            <Route path="wiki" element={<WikiRoutePage />} />
            <Route path="velocity" element={<VelocityPage />} />
            <Route path="work-orders">
              <Route index element={<WorkOrdersPage />} />
              <Route path="new" element={<CreateWorkOrderPage />} />
              <Route path=":orderId" element={<WorkOrderDetailPage />} />
            </Route>
            <Route path="lines">
              <Route index element={<LinesPage />} />
              <Route path="new" element={<FactoryLineEditPage />} />
              <Route path=":lineId" element={<LinesPage />} />
              <Route path=":lineId/edit" element={<FactoryLineEditPage />} />
            </Route>
            <Route path="automations">
              <Route index element={<AutomationsPage />} />
              <Route path="new" element={<HarnessLegacyAutomationsNewLineRedirect />} />
              <Route path=":lineId/edit" element={<HarnessLegacyAutomationsLineEditRedirect />} />
              <Route path=":appId" element={<AutomationsPage />} />
            </Route>
            <Route path="apps/:appId" element={<FactoryAppCanvasPage />} />
          </Route>
          <Route path=":factoryId/settings" element={factoryRoute(<FactorySettingsLayout />)}>
            <Route index element={<Navigate to={FACTORY_SETTINGS_NAV_ITEMS[0].id} replace />} />
            <Route path="general" element={<FactorySettingsGeneralPage />} />
            {FACTORY_SETTINGS_NAV_ITEMS.filter((item) => item.id !== "general").map((item) => (
              <Route
                key={item.id}
                path={item.id}
                element={
                  <FactorySettingsSoonPage
                    title={item.label}
                    description={`${item.label} settings for this workspace.`}
                    Icon={item.Icon}
                  />
                }
              />
            ))}
          </Route>
        </Route>
        <Route
          path="settings/integrations/:integrationName/setup"
          element={<div data-testid="integration-setup-placeholder">Integration setup</div>}
        />
      </Route>
    </Routes>
  );
}

/**
 * Shared Storybook shell for org home + app editor so the real React Router
 * links work: logo/Homepage → home, Software Factory card → live canvas.
 */
export function OrgWorkspaceHarness({
  startAt = "home",
  pathSuffix = "",
  appQuery = "",
  openAgentSidebar = false,
  homeFixture,
  appFixture,
  factoriesFixture,
  agentSuggestions,
  pageOverrides,
}: OrgWorkspaceHarnessProps) {
  const { orgId, canvasId } = resolveWorkspaceIds(homeFixture, appFixture, factoriesFixture);
  useOrgWorkspaceFixtureFetch({
    canvasId,
    openAgentSidebar,
    homeFixture,
    appFixture,
    factoriesFixture,
    agentSuggestions,
  });

  const homePath = pathSuffix ? `/${orgId}/${pathSuffix}` : `/${orgId}`;
  const appPath = `/${orgId}/apps/${canvasId}${appQuery ? `?${appQuery}` : ""}`;
  const initialPath = startAt === "app" ? appPath : homePath;

  const [queryClient] = useState(() => new QueryClient({ defaultOptions: { queries: { retry: false } } }));

  useEffect(() => {
    const onMessagesUpdated = () => {
      void queryClient.invalidateQueries({ queryKey: agentChatKeys.all });
    };
    window.addEventListener(STORYBOOK_AGENT_MESSAGES_UPDATED_EVENT, onMessagesUpdated);
    return () => window.removeEventListener(STORYBOOK_AGENT_MESSAGES_UPDATED_EVENT, onMessagesUpdated);
  }, [queryClient]);

  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <TooltipProvider delayDuration={150}>
          <div className="h-dvh w-full overflow-auto">
            <MemoryRouter initialEntries={[initialPath]}>
              <AccountProvider>
                <OrgWorkspaceRoutes pageOverrides={pageOverrides} />
              </AccountProvider>
            </MemoryRouter>
          </div>
        </TooltipProvider>
      </ThemeProvider>
    </QueryClientProvider>
  );
}
