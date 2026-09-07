import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useContext, useEffect, useState, type ComponentType, type ReactNode } from "react";
import { MemoryRouter, Navigate, Outlet, Route, Routes, useParams } from "react-router";

import { requestCanvasAgentSidebarOpen } from "@/components/CanvasToolSidebar/canvasAgentSidebarOpenRequest";
import { writeCanvasAgentSidebarOpen } from "@/components/CanvasToolSidebar/useCanvasToolSidebarState";
import { RequireAnyPermission, RequirePermission } from "@/components/PermissionGate";
import { RequireExperimentalFeature } from "@/components/RequireExperimentalFeature";
import { AccountProvider } from "@/contexts/AccountProvider";
import { PermissionsProvider } from "@/contexts/PermissionsProvider";
import { ThemeContext } from "@/contexts/themeContextState";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import { agentChatKeys } from "@/hooks/useAgentChats";
import { FEATURE_FACTORIES, FEATURE_WORKSPACE_MODELS } from "@/lib/experimentalFeatures";
import { setAgentSuggestions } from "@/lib/agentSuggestionsContext";
import { AppPage } from "@/pages/app";
import { STORYBOOK_AGENT_MESSAGES_UPDATED_EVENT } from "@/pages/app/__fixtures__/agentChatResponses";
import { canvasAppIds, type CanvasAppFixture } from "@/pages/app/__fixtures__/handlers";
import {
  AutomationsPage,
  CreateWorkOrderComposeRedirect,
  FactoriesIndexPage,
  FactoriesLayout,
  FactoryAppCanvasPage,
  FactoryAppSplitRunPage,
  FactoryHomeRedirect,
  FactoryLineEditPage,
  FactorySettingsAutomationsPage,
  FactorySettingsGeneralPage,
  FactorySettingsLayout,
  FactorySettingsRepositoryPage,
  FactorySettingsModelsPage,
  LegacyWorkOrderDetailRedirect,
  LegacyWorkOrderPermalinkRedirect,
  LegacyWorkOrdersRedirect,
  LinesPage,
  MissionsPage,
  NewWorkspacePage,
  OrganizationSettingsOverviewPage,
  OverviewPage,
  VelocityPage,
  WikiPage,
  WorkOrderDetailPage,
  WorkOrdersPage,
} from "@/pages/factories";
import type { FactoriesFixture } from "@/pages/factories/__fixtures__/handlers";
import { createFactoryLinePath, editFactoryLinePath } from "@/pages/factories/lib/factoryPagePaths";
import {
  AccountLinkedAccountsRedirect,
  LegacyFactoryOrganizationSettingsRedirect,
  LegacyFactorySettingsIndexRedirect,
  LegacyFactorySettingsRedirect,
  LegacyOrganizationSettingsRedirect,
  WorkspaceSpendingRedirect,
} from "@/pages/factories/pages/settings/FactorySettingsRedirects";
import { MissionDetailPage } from "@/pages/factories/pages/missions/MissionDetailPage";
import { ConfigureAutomationPage } from "@/pages/factories/pages/ConfigureAutomationPage";
import { OnboardingGate } from "@/pages/factories/pages/onboarding/OnboardingGate";
import { OrganizationSettingsWorkspaceUsagePage } from "@/pages/factories/pages/organizationSettings/OrganizationSettingsWorkspaceUsagePage";
import {
  OrganizationIntegrationDetailsPage,
  OrganizationIntegrationSetupPage,
} from "@/pages/factories/pages/organizationSettings/organizationSettingsRoutePages";
import { OrganizationSettingsIntegrationsPage } from "@/pages/factories/pages/organizationSettings/OrganizationSettingsIntegrationsPage";
import {
  FactoryOrganizationApiKeyDetailPage,
  FactoryOrganizationApiKeysPage,
  FactoryOrganizationLLMModelsPage,
  FactoryOrganizationMembersPage,
  FactoryOrganizationSecretDetailPage,
  FactoryOrganizationSecretsPage,
} from "@/pages/factories/pages/settings/FactoryOrganizationSettingsPages";
import { HomePage } from "@/pages/home";
import { homePageIds, type HomePageFixture, type StorybookOrgIntegration } from "@/pages/home/__fixtures__/handlers";
import { NewAppPage } from "@/pages/home/NewAppPage";
import { OrganizationSettings } from "@/pages/organization/settings";
import type { AgentSuggestion } from "@/ui/CanvasPage";
import { TooltipProvider } from "@/ui/tooltip";

import { FactorySettingsAccountNotificationsPage } from "@/pages/factories/pages/settings/FactorySettingsAccountNotificationsPage";
import { FactorySettingsAccountProfilePage } from "@/pages/factories/pages/settings/FactorySettingsAccountProfilePage";
import { FactorySettingsAccountSecurityPage } from "@/pages/factories/pages/settings/FactorySettingsAccountSecurityPage";

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
  overview?: ComponentType;
  /** When set, mounts `/setup` and gates other factory pages while pending. */
  onboarding?: ComponentType;
  /** Storybook-only Tasks page. Live app ignores this. */
  workOrders?: ComponentType;
  /** Storybook-only Velocity page (e.g. work-order flow prototype). */
  velocity?: ComponentType;
  /** Storybook-only Organization Spending explorer. Live app ignores this. */
  organizationSpending?: ComponentType;
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
  /** Organization connections the story starts with (e.g. an installed GitHub). */
  orgIntegrations?: StorybookOrgIntegration[];
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
  orgIntegrations?: StorybookOrgIntegration[];
}

function useOrgWorkspaceFixtureFetch(options: FixtureFetchOptions) {
  const { canvasId, openAgentSidebar, homeFixture, appFixture, factoriesFixture, agentSuggestions, orgIntegrations } =
    options;
  const [fixtureFetch] = useState(() => {
    // Persist before AppPage reads the preference in useState initializers.
    writeCanvasAgentSidebarOpen(canvasId, openAgentSidebar);
    if (agentSuggestions?.length) {
      setAgentSuggestions(canvasId, agentSuggestions);
    }
    const state = fixtureFetchState();
    const impl = createOrgWorkspaceFixtureFetch(state.original, {
      homeFixture,
      appFixture,
      factoriesFixture,
      orgIntegrations,
    });
    state.delegate = impl;
    return impl;
  });

  useEffect(() => {
    writeCanvasAgentSidebarOpen(canvasId, openAgentSidebar);
    if (openAgentSidebar) {
      requestCanvasAgentSidebarOpen(canvasId);
    }
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
  const { organizationId, factoryKey } = useParams<{ organizationId: string; factoryKey: string }>();
  if (!organizationId || !factoryKey) {
    return <Navigate to="/" replace />;
  }
  return <Navigate to={createFactoryLinePath(organizationId, factoryKey)} replace />;
}

function HarnessLegacyAutomationsLineEditRedirect() {
  const { organizationId, factoryKey, lineId } = useParams<{
    organizationId: string;
    factoryKey: string;
    lineId: string;
  }>();
  if (!organizationId || !factoryKey || !lineId) {
    return <Navigate to="/" replace />;
  }
  return <Navigate to={editFactoryLinePath(organizationId, factoryKey, lineId)} replace />;
}

function OptionalOnboardingGate({ enabled }: { enabled: boolean }) {
  if (!enabled) {
    return <Outlet />;
  }
  return <OnboardingGate />;
}

function factorySettingsOrganizationSpendingRoute(OrganizationSpendingPage: ComponentType) {
  return (
    <Route
      key="factory-settings-organization-spending"
      path="organization/spending"
      element={
        <RequirePermission resource="org" action="read">
          <OrganizationSpendingPage />
        </RequirePermission>
      }
    />
  );
}

const factorySettingsStorybookRoutes = [
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
  <Route key="factory-settings-workspace-general" path="workspace/general" element={<FactorySettingsGeneralPage />} />,
  <Route
    key="factory-settings-workspace-repository"
    path="workspace/repository"
    element={<FactorySettingsRepositoryPage />}
  />,
  <Route
    key="factory-settings-workspace-automations"
    path="workspace/automations"
    element={<FactorySettingsAutomationsPage />}
  />,
  <Route
    key="factory-settings-workspace-models"
    path="workspace/models"
    element={
      <RequireExperimentalFeature featureId={FEATURE_WORKSPACE_MODELS}>
        <FactorySettingsModelsPage />
      </RequireExperimentalFeature>
    }
  />,
  <Route key="factory-settings-workspace-spending" path="workspace/spending" element={<WorkspaceSpendingRedirect />} />,
  <Route key="factory-settings-workspace-usage" path="workspace/usage" element={<WorkspaceSpendingRedirect />} />,
  <Route
    key="factory-settings-organization-general"
    path="organization/general"
    element={<OrganizationSettingsOverviewPage />}
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
        <FactoryOrganizationLLMModelsPage />
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
    element={<OrganizationIntegrationDetailsPage />}
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
];

function OrgWorkspaceRoutes({ pageOverrides }: { pageOverrides?: OrgWorkspacePageOverrides }) {
  const WikiRoutePage = pageOverrides?.wiki ?? WikiPage;
  const OverviewRoutePage = pageOverrides?.overview ?? OverviewPage;
  const WorkOrdersRoutePage = pageOverrides?.workOrders ?? WorkOrdersPage;
  const VelocityRoutePage = pageOverrides?.velocity ?? VelocityPage;
  const OnboardingRoutePage = pageOverrides?.onboarding;
  const onboardingEnabled = Boolean(OnboardingRoutePage);

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
          <Route path="new" element={factoryRoute(<NewWorkspacePage />)} />
          <Route path=":factoryKey" element={factoryRoute(<FactoriesLayout />)}>
            <Route element={<OptionalOnboardingGate enabled={onboardingEnabled} />}>
              <Route index element={<FactoryHomeRedirect />} />
              {OnboardingRoutePage ? <Route path="setup" element={<OnboardingRoutePage />} /> : null}
              <Route path="overview" element={<OverviewRoutePage />} />
              <Route path="missions" element={<MissionsPage />} />
              <Route path="missions/:missionId" element={<MissionDetailPage />} />
              <Route path="wiki" element={<WikiRoutePage />} />
              <Route path="velocity" element={<VelocityRoutePage />} />
              <Route path="tasks">
                <Route index element={<WorkOrdersRoutePage />} />
                <Route path="new" element={<CreateWorkOrderComposeRedirect />} />
                <Route path=":orderId" element={<LegacyWorkOrderDetailRedirect />} />
              </Route>
              <Route path="task/:orderNumber" element={<WorkOrderDetailPage />} />
              {/* Back-compat for bookmarks made before `work-order(s)` was renamed to `task(s)`. */}
              <Route path="work-orders/*" element={<LegacyWorkOrdersRedirect />} />
              <Route path="work-order/:orderNumber" element={<LegacyWorkOrderPermalinkRedirect />} />
              <Route path="lines">
                <Route index element={<FactoryHomeRedirect />} />
                <Route path="new" element={<FactoryLineEditPage />} />
                <Route path=":lineId" element={<LinesPage />} />
                <Route path=":lineId/edit" element={<FactoryLineEditPage />} />
                {/* Storybook design preview: factory WorkOrderCanvas node chrome */}
                <Route path=":lineId/phases/:phaseId/configure" element={<ConfigureAutomationPage />} />
              </Route>
              <Route path="automations">
                <Route index element={<AutomationsPage />} />
                <Route path="new" element={<HarnessLegacyAutomationsNewLineRedirect />} />
                <Route path=":lineId/edit" element={<HarnessLegacyAutomationsLineEditRedirect />} />
                <Route path=":appId" element={<AutomationsPage />} />
              </Route>
              <Route path="apps/:appId" element={<FactoryAppCanvasPage />} />
              <Route path="apps/:appId/split-run" element={<FactoryAppSplitRunPage />} />
            </Route>
          </Route>
          <Route path=":factoryKey/settings" element={factoryRoute(<FactorySettingsLayout />)}>
            {factorySettingsStorybookRoutes}
            {factorySettingsOrganizationSpendingRoute(
              pageOverrides?.organizationSpending ?? OrganizationSettingsWorkspaceUsagePage,
            )}
            <Route key="factory-settings-legacy" path="*" element={<LegacyFactorySettingsRedirect />} />
          </Route>
          <Route
            path=":factoryKey/organization/*"
            element={factoryRoute(<LegacyFactoryOrganizationSettingsRedirect />)}
          />
        </Route>
        <Route path="organization/*" element={factoryRoute(<LegacyOrganizationSettingsRedirect />)} />
        <Route path="settings/llm-spend" element={<LegacyOrganizationSettingsRedirect destination="llm-spend" />} />
        <Route
          path="settings/integrations/:integrationName/setup"
          element={<div data-testid="integration-setup-placeholder">Integration setup</div>}
        />
        <Route path="settings/*" element={<OrganizationSettings />} />
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
  orgIntegrations,
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
    orgIntegrations,
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
      <OptionalThemeProvider>
        <TooltipProvider delayDuration={150}>
          <div className="h-dvh w-full overflow-auto">
            <MemoryRouter initialEntries={[initialPath]}>
              <AccountProvider>
                <OrgWorkspaceRoutes pageOverrides={pageOverrides} />
              </AccountProvider>
            </MemoryRouter>
          </div>
        </TooltipProvider>
      </OptionalThemeProvider>
    </QueryClientProvider>
  );
}

/** Prefer Storybook toolbar theme when present; else mount ThemeProvider. */
function OptionalThemeProvider({ children }: { children: ReactNode }) {
  const inheritedTheme = useContext(ThemeContext);
  if (inheritedTheme) {
    return children;
  }
  return <ThemeProvider>{children}</ThemeProvider>;
}
