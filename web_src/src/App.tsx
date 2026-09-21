import { TooltipProvider } from "@/ui/tooltip";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React, { Suspense } from "react";
import { BrowserRouter, Navigate, Outlet, Route, Routes, useLocation, useParams, useSearchParams } from "react-router";
import { appPath, appSettingsPath } from "./lib/appPaths";
import { FEATURE_FACTORIES } from "./lib/experimentalFeatures";
import { usePersistOrganizationLastLocation } from "./hooks/usePersistOrganizationLastLocation";
import { UserNotificationsListener } from "./hooks/useUserNotificationsWebsocket";
import { resolveOrganizationUidRedirect } from "./lib/organizationPath";
import { isReservedAppPathSegment } from "./lib/reservedAppPaths";
import { useConsumeIntegrationSetupReturnOnArrival } from "./hooks/useConsumeIntegrationSetupReturnOnArrival";
import { useOrganization } from "./hooks/useOrganizationData";
import { useRedirectIntegrationSetupReturn } from "./hooks/useRedirectIntegrationSetupReturn";
import { Toaster } from "sonner";
import "./App.css";

// Import pages
import AuthGuard from "./components/AuthGuard";
import { GlobalCommandPalette } from "./components/GlobalCommandPalette";
import { RequireExperimentalFeature } from "./components/RequireExperimentalFeature";
import { AccountProvider } from "./contexts/AccountProvider";
import { ThemeProvider } from "./contexts/ThemeProvider";
import { useAccount } from "./contexts/useAccount";
import { PermissionsProvider } from "./contexts/PermissionsProvider";
import { RequirePermission } from "./components/PermissionGate";
import { isFactoryAppConfigureMode } from "./pages/factories/lib/factoryAppCanvasCopy";
import { Login } from "./pages/auth/Login";
import { OrganizationOnboardingRedirect } from "./pages/auth/OrganizationOnboardingRedirect";
import OwnerSetup from "./pages/auth/OwnerSetup";
import { RootOrganizationRedirect } from "./pages/auth/RootOrganizationRedirect";
import WelcomeSurvey from "./pages/auth/WelcomeSurvey";
import { createFactoryLinePath, editFactoryLinePath } from "./pages/factories/lib/factoryPagePaths";
import { OnboardingEntryPathProvider } from "./pages/factories/pages/onboarding/OnboardingEntryPathProvider";
import { InitialWorkspaceOnboarding } from "./pages/factories/pages/onboarding/InitialWorkspaceOnboarding";
import { OnboardingWorkspaceResolutionProvider } from "./pages/factories/pages/onboarding/OnboardingWorkspaceResolutionProvider";
import { HomePage } from "./pages/home";
import InviteLinkAccept from "./pages/auth/InviteLinkAccept";
import ImpersonationBanner from "./components/ImpersonationBanner";
import { usePageObservability } from "./hooks/usePageObservability";
import { Skeleton } from "./ui/skeleton";

// Create a client
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      staleTime: 5 * 60 * 1000, // 5 minutes
      gcTime: 10 * 60 * 1000, // 10 minutes
    },
  },
});

function lazyNamed<TModule extends Record<PropertyKey, unknown>, TName extends keyof TModule>(
  loader: () => Promise<TModule>,
  exportName: TName,
) {
  type Exported = TModule[TName];
  type Props = Exported extends React.ComponentType<infer P> ? P : never;
  return React.lazy(async () => {
    const loaded = await loader();
    return { default: loaded[exportName] as React.ComponentType<Props> };
  });
}

const CanvasSettingsPage = lazyNamed(() => import("./pages/canvas/settings"), "CanvasSettingsPage");
const AppDefaultTabGate = lazyNamed(() => import("./pages/app/AppDefaultTabGate"), "AppDefaultTabGate");
const NewAppPage = lazyNamed(() => import("./pages/home/NewAppPage"), "NewAppPage");
const GitHubInstallApprovedPage = lazyNamed(
  () => import("./pages/github/GitHubInstallApprovedPage"),
  "GitHubInstallApprovedPage",
);
const OrganizationSettings = lazyNamed(() => import("./pages/organization/settings"), "OrganizationSettings");
const AdminLayout = React.lazy(() => import("./pages/admin/AdminLayout"));
const OrganizationsListAdmin = React.lazy(() => import("./pages/admin/OrganizationsList"));
const OrganizationDetailAdmin = React.lazy(() => import("./pages/admin/OrganizationDetail"));
const AccountsListAdmin = React.lazy(() => import("./pages/admin/AccountsList"));
const InstallationSettingsAdmin = React.lazy(() => import("./pages/admin/InstallationSettings"));
const RunnerTasksAdmin = React.lazy(() => import("./pages/admin/RunnerTasks"));
const PolarWebhooksAdmin = lazyNamed(() => import("./pages/admin/PolarWebhooks"), "PolarWebhooks");
const PriceBooksAdmin = lazyNamed(() => import("./pages/admin/PriceBooks"), "PriceBooks");
const AutomationsPage = lazyNamed(() => import("./pages/factories"), "AutomationsPage");
const CreateWorkOrderComposeRedirect = lazyNamed(() => import("./pages/factories"), "CreateWorkOrderComposeRedirect");
const FactoriesIndexPage = lazyNamed(() => import("./pages/factories"), "FactoriesIndexPage");
const FactoriesLayout = lazyNamed(() => import("./pages/factories"), "FactoriesLayout");
const FactoryAppCanvasPage = lazyNamed(() => import("./pages/factories"), "FactoryAppCanvasPage");
const FactoryAppSplitRunPage = lazyNamed(() => import("./pages/factories"), "FactoryAppSplitRunPage");
const FactoryHomeRedirect = lazyNamed(() => import("./pages/factories"), "FactoryHomeRedirect");
const FactoryLineEditPage = lazyNamed(() => import("./pages/factories"), "FactoryLineEditPage");
const FactorySettingsRoutes = lazyNamed(
  () => import("./pages/factories/pages/settings/FactorySettingsRoutes"),
  "FactorySettingsRoutes",
);
const LegacyFactoryAppRedirect = lazyNamed(() => import("./pages/factories"), "LegacyFactoryAppRedirect");
const LegacyFactoryAppSplitRunRedirect = lazyNamed(
  () => import("./pages/factories"),
  "LegacyFactoryAppSplitRunRedirect",
);
const LegacyWorkOrderDetailRedirect = lazyNamed(() => import("./pages/factories"), "LegacyWorkOrderDetailRedirect");
const LegacyWorkOrderPermalinkRedirect = lazyNamed(
  () => import("./pages/factories"),
  "LegacyWorkOrderPermalinkRedirect",
);
const LegacyWorkOrdersRedirect = lazyNamed(() => import("./pages/factories"), "LegacyWorkOrdersRedirect");
const LinesPage = lazyNamed(() => import("./pages/factories"), "LinesPage");
const MissionsPage = lazyNamed(() => import("./pages/factories"), "MissionsPage");
const NewWorkspacePage = lazyNamed(() => import("./pages/factories"), "NewWorkspacePage");
const OnboardingGate = lazyNamed(() => import("./pages/factories"), "OnboardingGate");
const OnboardingPage = lazyNamed(() => import("./pages/factories"), "OnboardingPage");
const VelocityPage = lazyNamed(() => import("./pages/factories"), "VelocityPage");
const WikiPage = lazyNamed(() => import("./pages/factories"), "WikiPage");
const WorkOrderDetailPage = lazyNamed(() => import("./pages/factories"), "WorkOrderDetailPage");
const WorkOrdersPage = lazyNamed(() => import("./pages/factories"), "WorkOrdersPage");
const WorkspaceOverviewPage = lazyNamed(() => import("./pages/factories"), "WorkspaceOverviewPage");
const ChecksPRFeedbackSetupPage = lazyNamed(() => import("./pages/factories"), "ChecksPRFeedbackSetupPage");
const DiscussionPRFeedbackSetupPage = lazyNamed(() => import("./pages/factories"), "DiscussionPRFeedbackSetupPage");
const SentryIntakeSetupPage = lazyNamed(() => import("./pages/factories"), "SentryIntakeSetupPage");
const LegacyFactoryOrganizationSettingsRedirect = lazyNamed(
  () => import("./pages/factories/pages/settings/FactorySettingsRedirects"),
  "LegacyFactoryOrganizationSettingsRedirect",
);
const LegacyOrganizationSettingsRedirect = lazyNamed(
  () => import("./pages/factories/pages/settings/FactorySettingsRedirects"),
  "LegacyOrganizationSettingsRedirect",
);

function RouteFallback() {
  return (
    <div className="flex h-full items-center justify-center p-6">
      <Skeleton className="h-8 w-40" />
    </div>
  );
}

const withAuthOnly = (Component: React.ComponentType) => (
  <AuthGuard>
    <Component />
  </AuthGuard>
);

const withAuthAndPermission = (Component: React.ComponentType, resource: string, action: string) => (
  <AuthGuard>
    <RequirePermission resource={resource} action={action}>
      <Component />
    </RequirePermission>
  </AuthGuard>
);

const withAuthPermissionAndFactoriesFeature = (Component: React.ComponentType, resource: string, action: string) => (
  <AuthGuard>
    <RequirePermission resource={resource} action={action}>
      <RequireExperimentalFeature featureId={FEATURE_FACTORIES}>
        <Component />
      </RequireExperimentalFeature>
    </RequirePermission>
  </AuthGuard>
);

function organizationScopedRouteTree() {
  return (
    <Route path=":organizationId" element={<OrganizationScope />}>
      <Route index element={withAuthAndPermission(HomePage, "canvases", "read")} />
      <Route path="apps">
        <Route path="new" element={withAuthAndPermission(NewAppPage, "canvases", "create")} />
        <Route path=":appId/settings" element={withAuthAndPermission(CanvasSettingsPage, "canvases", "update")} />
        <Route path=":appId" element={withAuthAndPermission(CanvasPageConfigureGate, "canvases", "read")} />
      </Route>
      <Route path="canvases/:canvasId/settings" element={<LegacyCanvasRedirect settings />} />
      <Route path="canvases/:canvasId" element={<LegacyCanvasRedirect />} />
      <Route path="workspaces">
        <Route index element={withAuthPermissionAndFactoriesFeature(FactoriesIndexPage, "factories", "read")} />
        <Route path="new" element={withAuthPermissionAndFactoriesFeature(NewWorkspacePage, "factories", "create")} />
        <Route path=":factoryKey" element={withAuthPermissionAndFactoriesFeature(FactoriesLayout, "factories", "read")}>
          <Route element={<OnboardingGate />}>
            <Route index element={<FactoryHomeRedirect />} />
            <Route path="setup" element={<OnboardingPage />} />
            <Route path="onboarding" element={<Navigate to="../setup" replace />} />
            <Route path="overview" element={<WorkspaceOverviewPage />} />
            <Route path="missions" element={<MissionsPage />} />
            <Route path="wiki" element={<WikiPage />} />
            <Route path="velocity" element={<VelocityPage />} />
            <Route path="tasks">
              <Route index element={<WorkOrdersPage />} />
              <Route path="new" element={<CreateWorkOrderComposeGate />} />
              <Route path=":orderId" element={<LegacyWorkOrderDetailRedirect />} />
            </Route>
            <Route path="task/:orderNumber" element={<WorkOrderDetailPage />} />
            {/* Back-compat for bookmarks made before `work-order(s)` was renamed to `task(s)`. */}
            <Route path="work-orders/*" element={<LegacyWorkOrdersRedirect />} />
            <Route path="work-order/:orderNumber" element={<LegacyWorkOrderPermalinkRedirect />} />
            <Route path="lines">
              <Route index element={<FactoryHomeRedirect />} />
              <Route path="new" element={<FactoryLineEditPageGate />} />
              <Route path=":lineId" element={<LinesPage />} />
              <Route path=":lineId/edit" element={<FactoryLineEditPageGate />} />
              <Route path=":lineId/setup/comments" element={<DiscussionPRFeedbackSetupPage />} />
              <Route path=":lineId/setup/checks" element={<ChecksPRFeedbackSetupPage />} />
              <Route path=":lineId/setup/sentry" element={<SentryIntakeSetupPage />} />
            </Route>
            <Route path="automations">
              <Route index element={<AutomationsPage />} />
              <Route path="new" element={<LegacyAutomationsNewLineRedirect />} />
              <Route path=":lineId/edit" element={<LegacyAutomationsLineEditRedirect />} />
              <Route path=":appId" element={<FactoryCanvasConfigureGate />} />
              <Route path=":appId/split-run" element={<FactoryAppSplitRunPage />} />
            </Route>
            <Route path="apps/:appId" element={<LegacyFactoryAppRedirect />} />
            <Route path="apps/:appId/split-run" element={<LegacyFactoryAppSplitRunRedirect />} />
          </Route>
        </Route>
        <Route
          path=":factoryKey/settings/*"
          element={withAuthPermissionAndFactoriesFeature(FactorySettingsRoutes, "factories", "read")}
        />
        <Route
          path=":factoryKey/organization/*"
          element={withAuthPermissionAndFactoriesFeature(
            LegacyFactoryOrganizationSettingsRedirect,
            "factories",
            "read",
          )}
        />
      </Route>
      <Route
        path="organization/*"
        element={withAuthPermissionAndFactoriesFeature(LegacyOrganizationSettingsRedirect, "factories", "read")}
      />
      <Route
        path="settings/llm-spend"
        element={withAuthOnly(() => (
          <LegacyOrganizationSettingsRedirect destination="llm-spend" />
        ))}
      />
      <Route path="settings/*" element={withAuthOnly(OrganizationSettings)} />
    </Route>
  );
}

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <AccountProvider>
          <TooltipProvider delayDuration={150}>
            <AppRouter />
          </TooltipProvider>
          <Toaster position="bottom-center" closeButton />
        </AccountProvider>
      </ThemeProvider>
    </QueryClientProvider>
  );
}

function AppRouter() {
  return (
    <BrowserRouter>
      <PageObservabilityScope />
      <div className="flex h-dvh flex-col overflow-hidden">
        <ImpersonationBanner />
        <div className="relative flex-1 overflow-auto">
          <SetupGuard>
            <GlobalCommandPalette />
            <Suspense fallback={<RouteFallback />}>
              <Routes>
                <Route path="login" element={<Login />} />
                <Route path="signup" element={<Login mode="signup" />} />
                <Route path="welcome" element={withAuthOnly(WelcomeSurvey)} />
                <Route path="onboarding" element={withAuthOnly(OrganizationOnboardingRoute)} />
                <Route path="setup" element={<OwnerSetup />} />
                <Route path="admin" element={<AdminLayout />}>
                  <Route index element={<OrganizationsListAdmin />} />
                  <Route path="accounts" element={<AccountsListAdmin />} />
                  <Route path="settings" element={<InstallationSettingsAdmin />} />
                  <Route path="price-books" element={<PriceBooksAdmin />} />
                  <Route path="runner-tasks" element={<RunnerTasksAdmin />} />
                  <Route path="polar-webhooks" element={<PolarWebhooksAdmin />} />
                  <Route path="organizations/:orgId" element={<OrganizationDetailAdmin />} />
                </Route>
                <Route path="" element={withAuthOnly(RootOrganizationRedirect)} />
                <Route path="invite/:token" element={withAuthOnly(InviteLinkAccept)} />
                {/* GitHub App owners who approve an install request may not have a SuperPlane session. */}
                <Route path="github/approved" element={<GitHubInstallApprovedPage />} />
                {organizationScopedRouteTree()}
                <Route path="*" element={<Navigate to="/" />} />
              </Routes>
            </Suspense>
          </SetupGuard>
        </div>
      </div>
    </BrowserRouter>
  );
}

function OrganizationOnboardingRoute() {
  return (
    <OrganizationOnboardingRedirect
      renderWorkspace={(workspace, entryPath, reresolveWorkspace) => (
        <OnboardingEntryPathProvider path={entryPath}>
          <OnboardingWorkspaceResolutionProvider resolve={reresolveWorkspace}>
            <InitialWorkspaceOnboarding
              organizationId={workspace.organizationSlug}
              factoryKey={workspace.workspaceKey}
            />
          </OnboardingWorkspaceResolutionProvider>
        </OnboardingEntryPathProvider>
      )}
    />
  );
}

function PageObservabilityScope() {
  usePageObservability();
  return null;
}

export function OrganizationScope() {
  const { organizationId: segment } = useParams<{ organizationId: string }>();
  const { account } = useAccount();
  const accountId = account?.id;
  const location = useLocation();

  const isReserved = isReservedAppPathSegment(segment);
  // The route param accepts either the org slug or its UID, so resolve it
  // once here and self-correct any UID URL to the slug below. Every other
  // in-app link reuses this same `:organizationId` URL segment, so fixing
  // it at this single boundary keeps the rest of the app slug-only.
  const { data: organization } = useOrganization(segment ?? "", !isReserved && !!segment);
  const resolvedId = organization?.metadata?.id ?? "";
  const resolvedSlug = organization?.metadata?.slug ?? "";
  useRedirectIntegrationSetupReturn(segment, resolvedSlug);
  useConsumeIntegrationSetupReturnOnArrival(resolvedSlug || segment);

  const uidRedirectPath =
    !isReserved && segment
      ? resolveOrganizationUidRedirect({
          pathname: location.pathname,
          search: location.search,
          hash: location.hash,
          segment,
          organizationId: resolvedId,
          organizationSlug: resolvedSlug,
        })
      : null;
  usePersistOrganizationLastLocation({
    accountId,
    resolvedSlug,
    isReserved,
    uidRedirectPath,
    path: `${location.pathname}${location.search}`,
  });

  if (isReserved) {
    return <Navigate to="/" replace />;
  }

  if (uidRedirectPath) {
    return <Navigate to={uidRedirectPath} replace />;
  }

  return (
    <PermissionsProvider>
      <UserNotificationsListener organizationId={resolvedId} />
      <Outlet />
    </PermissionsProvider>
  );
}

function CreateWorkOrderComposeGate() {
  return (
    <RequirePermission resource="work_orders" action="create">
      <CreateWorkOrderComposeRedirect />
    </RequirePermission>
  );
}

function FactoryLineEditPageGate() {
  return (
    <RequirePermission resource="factories" action="update">
      <FactoryLineEditPage />
    </RequirePermission>
  );
}

function CanvasConfigureGate({ children }: { children: React.ReactNode }) {
  const [searchParams] = useSearchParams();
  if (isFactoryAppConfigureMode(searchParams)) {
    return (
      <RequirePermission resource="canvases" action="update">
        {children}
      </RequirePermission>
    );
  }
  return <>{children}</>;
}

function CanvasPageConfigureGate() {
  return (
    <CanvasConfigureGate>
      <AppDefaultTabGate />
    </CanvasConfigureGate>
  );
}

function FactoryCanvasConfigureGate() {
  return (
    <CanvasConfigureGate>
      <FactoryAppCanvasPage />
    </CanvasConfigureGate>
  );
}

function LegacyAutomationsNewLineRedirect() {
  const { organizationId, factoryKey } = useParams<{ organizationId: string; factoryKey: string }>();
  if (!organizationId || !factoryKey) return <Navigate to="/" replace />;
  return <Navigate to={createFactoryLinePath(organizationId, factoryKey)} replace />;
}

function LegacyAutomationsLineEditRedirect() {
  const { organizationId, factoryKey, lineId } = useParams<{
    organizationId: string;
    factoryKey: string;
    lineId: string;
  }>();
  if (!organizationId || !factoryKey || !lineId) return <Navigate to="/" replace />;
  return <Navigate to={editFactoryLinePath(organizationId, factoryKey, lineId)} replace />;
}

function LegacyCanvasRedirect({ settings = false }: { settings?: boolean }) {
  const { organizationId, canvasId } = useParams<{ organizationId: string; canvasId: string }>();
  const location = useLocation();

  if (!organizationId || !canvasId) return <Navigate to="/" replace />;

  const path = settings ? appSettingsPath(organizationId, canvasId) : appPath(organizationId, canvasId);
  return <Navigate to={`${path}${location.search}`} replace />;
}

function SetupGuard({ children }: { children: React.ReactNode }) {
  const { setupRequired, loading } = useAccount();
  const location = useLocation();

  if (!loading && setupRequired && location.pathname !== "/setup") {
    return <Navigate to="/setup" replace />;
  }

  return <>{children}</>;
}

export default App;
