import { TooltipProvider } from "@/ui/tooltip";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React, { useEffect } from "react";
import { BrowserRouter, Navigate, Outlet, Route, Routes, useLocation, useParams } from "react-router";
import { appPath, appSettingsPath } from "./lib/appPaths";
import { FEATURE_FACTORIES } from "./lib/experimentalFeatures";
import { recordLastVisitedOrganization } from "./lib/lastVisitedOrganization";
import { isReservedAppPathSegment } from "./lib/reservedAppPaths";
import { useConsumeIntegrationSetupReturnOnArrival } from "./hooks/useConsumeIntegrationSetupReturnOnArrival";
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
import { Login } from "./pages/auth/Login";
import OrganizationCreate from "./pages/auth/OrganizationCreate";
import OrganizationSelect from "./pages/auth/OrganizationSelect";
import OwnerSetup from "./pages/auth/OwnerSetup";
import WelcomeSurvey from "./pages/auth/WelcomeSurvey";
import { CanvasSettingsPage } from "./pages/canvas/settings";
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
  FactorySettingsNotificationsPage,
  FactorySettingsProfilePage,
  FactorySettingsSoonPage,
  FactorySettingsUsagePage,
  FACTORY_SETTINGS_NAV_ITEMS,
  isFactorySettingsComingSoon,
  LegacyWorkOrderDetailRedirect,
  LinesPage,
  MissionsPage,
  NewWorkspacePage,
  OnboardingGate,
  OnboardingPage,
  OrganizationSettingsLayout,
  organizationSettingsSectionRoutes,
  VelocityPage,
  WikiPage,
  WorkOrderDetailPage,
  WorkOrdersPage,
  WorkspaceOverviewPage,
} from "./pages/factories";
import { createFactoryLinePath, editFactoryLinePath } from "./pages/factories/lib/factoryPagePaths";
import { HomePage } from "./pages/home";
import { NewAppPage } from "./pages/home/NewAppPage";
import { InstallPage } from "./pages/install";
import { OrganizationSettings } from "./pages/organization/settings";
import { AppDefaultTabGate } from "./pages/app/AppDefaultTabGate";
import InviteLinkAccept from "./pages/auth/InviteLinkAccept";
import AdminLayout from "./pages/admin/AdminLayout";
import OrganizationsListAdmin from "./pages/admin/OrganizationsList";
import OrganizationDetailAdmin from "./pages/admin/OrganizationDetail";
import AccountsListAdmin from "./pages/admin/AccountsList";
import InstallationSettingsAdmin from "./pages/admin/InstallationSettings";
import RunnerTasksAdmin from "./pages/admin/RunnerTasks";
import ImpersonationBanner from "./components/ImpersonationBanner";
import { usePageObservability } from "./hooks/usePageObservability";

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
            <Routes>
              {/* public routes */}
              <Route path="login" element={<Login />} />
              <Route path="signup" element={<Login mode="signup" />} />
              <Route path="welcome" element={withAuthOnly(WelcomeSurvey)} />
              <Route path="create" element={<OrganizationCreate />} />
              <Route path="setup" element={<OwnerSetup />} />

              {/* Admin dashboard routes */}
              <Route path="admin" element={<AdminLayout />}>
                <Route index element={<OrganizationsListAdmin />} />
                <Route path="accounts" element={<AccountsListAdmin />} />
                <Route path="settings" element={<InstallationSettingsAdmin />} />
                <Route path="runner-tasks" element={<RunnerTasksAdmin />} />
                <Route path="organizations/:orgId" element={<OrganizationDetailAdmin />} />
              </Route>

              {/* Organization selection and creation */}
              <Route path="" element={withAuthOnly(OrganizationSelect)} />

              {/* Invite link acceptance */}
              <Route path="invite/:token" element={withAuthOnly(InviteLinkAccept)} />

              {/* GitHub app installation */}
              <Route path="install" element={withAuthOnly(InstallPage)} />

              {/* Organization-scoped protected routes */}
              <Route path=":organizationId" element={<OrganizationScope />}>
                <Route index element={withAuthAndPermission(HomePage, "canvases", "read")} />
                <Route path="apps">
                  <Route path="new" element={withAuthAndPermission(NewAppPage, "canvases", "create")} />
                  <Route
                    path=":appId/settings"
                    element={withAuthAndPermission(CanvasSettingsPage, "canvases", "update")}
                  />
                  <Route path=":appId" element={withAuthAndPermission(AppDefaultTabGate, "canvases", "read")} />
                </Route>
                <Route path="canvases/:canvasId/settings" element={<LegacyCanvasRedirect settings />} />
                <Route path="canvases/:canvasId" element={<LegacyCanvasRedirect />} />
                <Route path="workspaces">
                  <Route
                    index
                    element={withAuthPermissionAndFactoriesFeature(FactoriesIndexPage, "factories", "read")}
                  />
                  <Route
                    path="new"
                    element={withAuthPermissionAndFactoriesFeature(NewWorkspacePage, "factories", "create")}
                  />
                  <Route
                    path=":factoryKey"
                    element={withAuthPermissionAndFactoriesFeature(FactoriesLayout, "factories", "read")}
                  >
                    <Route element={<OnboardingGate />}>
                      <Route index element={<FactoryHomeRedirect />} />
                      <Route path="setup" element={<OnboardingPage />} />
                      <Route path="onboarding" element={<Navigate to="../setup" replace />} />
                      <Route path="overview" element={<WorkspaceOverviewPage />} />
                      <Route path="missions" element={<MissionsPage />} />
                      <Route path="wiki" element={<WikiPage />} />
                      <Route path="velocity" element={<VelocityPage />} />
                      <Route path="work-orders">
                        <Route index element={<WorkOrdersPage />} />
                        <Route path="new" element={<CreateWorkOrderComposeGate />} />
                        {/* Legacy `/work-orders/:orderId` bookmark shape — redirects to the line board. */}
                        <Route path=":orderId" element={<LegacyWorkOrderDetailRedirect />} />
                      </Route>
                      <Route path="work-order/:orderNumber" element={<WorkOrderDetailPage />} />
                      <Route path="lines">
                        <Route index element={<FactoryHomeRedirect />} />
                        <Route path="new" element={<FactoryLineEditPageGate />} />
                        <Route path=":lineId" element={<LinesPage />} />
                        <Route path=":lineId/edit" element={<FactoryLineEditPageGate />} />
                      </Route>
                      <Route path="automations">
                        <Route index element={<AutomationsPage />} />
                        <Route path="new" element={<LegacyAutomationsNewLineRedirect />} />
                        <Route path=":lineId/edit" element={<LegacyAutomationsLineEditRedirect />} />
                        <Route path=":appId" element={<AutomationsPage />} />
                      </Route>
                      <Route path="apps/:appId" element={<FactoryAppCanvasPage />} />
                      <Route path="apps/:appId/split-run" element={<FactoryAppSplitRunPage />} />
                    </Route>
                    <Route path="settings/*" element={<Navigate to={FACTORY_SETTINGS_NAV_ITEMS[0].id} replace />} />
                  </Route>
                  <Route
                    path=":factoryKey/settings"
                    element={withAuthPermissionAndFactoriesFeature(FactorySettingsLayout, "factories", "read")}
                  >
                    {factorySettingsSectionRoutes}
                  </Route>
                  <Route
                    path=":factoryKey/organization"
                    element={withAuthPermissionAndFactoriesFeature(OrganizationSettingsLayout, "factories", "read")}
                  >
                    {organizationSettingsSectionRoutes}
                  </Route>
                </Route>
                <Route path="settings/*" element={withAuthOnly(OrganizationSettings)} />
              </Route>

              {/* Catch-all route */}
              <Route path="*" element={<Navigate to="/" />} />
            </Routes>
          </SetupGuard>
        </div>
      </div>
    </BrowserRouter>
  );
}

function PageObservabilityScope() {
  usePageObservability();
  return null;
}

function OrganizationScope() {
  const { organizationId } = useParams<{ organizationId: string }>();
  const { account } = useAccount();
  useConsumeIntegrationSetupReturnOnArrival(organizationId);

  useEffect(() => {
    if (account?.id && organizationId && !isReservedAppPathSegment(organizationId)) {
      recordLastVisitedOrganization(account.id, organizationId);
    }
  }, [account?.id, organizationId]);

  if (isReservedAppPathSegment(organizationId)) {
    return <Navigate to="/" replace />;
  }

  return (
    <PermissionsProvider>
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

const factorySettingsSectionRoutes = [
  <Route key="factory-settings-index" index element={<Navigate to={FACTORY_SETTINGS_NAV_ITEMS[0].id} replace />} />,
  <Route key="factory-settings-general" path="general" element={<FactorySettingsGeneralPage />} />,
  <Route key="factory-settings-automations" path="automations" element={<FactorySettingsAutomationsPage />} />,
  <Route key="factory-settings-usage" path="usage" element={<FactorySettingsUsagePage />} />,
  <Route key="factory-settings-profile" path="profile" element={<FactorySettingsProfilePage />} />,
  <Route key="factory-settings-notifications" path="notifications" element={<FactorySettingsNotificationsPage />} />,
  ...FACTORY_SETTINGS_NAV_ITEMS.filter(isFactorySettingsComingSoon).map((item) => (
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
  )),
];

function LegacyAutomationsNewLineRedirect() {
  const { organizationId, factoryKey } = useParams<{ organizationId: string; factoryKey: string }>();
  if (!organizationId || !factoryKey) {
    return <Navigate to="/" replace />;
  }
  return <Navigate to={createFactoryLinePath(organizationId, factoryKey)} replace />;
}

function LegacyAutomationsLineEditRedirect() {
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

function LegacyCanvasRedirect({ settings = false }: { settings?: boolean }) {
  const { organizationId, canvasId } = useParams<{ organizationId: string; canvasId: string }>();
  const location = useLocation();

  if (!organizationId || !canvasId) {
    return <Navigate to="/" replace />;
  }

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
