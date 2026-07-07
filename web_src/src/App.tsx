import { TooltipProvider } from "@/ui/tooltip";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React, { useEffect } from "react";
import { BrowserRouter, Navigate, Outlet, Route, Routes, useLocation, useParams } from "react-router-dom";
import { appPath, appSettingsPath } from "./lib/appPaths";
import { recordLastVisitedOrganization } from "./lib/lastVisitedOrganization";
import { Toaster } from "sonner";
import "./App.css";

// Import pages
import AuthGuard from "./components/AuthGuard";
import { GlobalCommandPalette } from "./components/GlobalCommandPalette";
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
import { HomePage } from "./pages/home";
import { NewAppPage } from "./pages/home/NewAppPage";
import { InstallPage } from "./pages/install";
import { OrganizationSettings } from "./pages/organization/settings";
import { AppPage } from "./pages/app";
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
        <div className="flex-1 overflow-auto">
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
                  <Route path="new" element={withAuthAndPermission(NewAppPage, "canvases", "read")} />
                  <Route
                    path=":appId/settings"
                    element={withAuthAndPermission(CanvasSettingsPage, "canvases", "update")}
                  />
                  <Route path=":appId" element={withAuthAndPermission(AppPage, "canvases", "read")} />
                </Route>
                <Route path="canvases/:canvasId/settings" element={<LegacyCanvasRedirect settings />} />
                <Route path="canvases/:canvasId" element={<LegacyCanvasRedirect />} />
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

  useEffect(() => {
    if (account?.id && organizationId) {
      recordLastVisitedOrganization(account.id, organizationId);
    }
  }, [account?.id, organizationId]);

  return (
    <PermissionsProvider>
      <Outlet />
    </PermissionsProvider>
  );
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
