import { TooltipProvider } from "@/ui/tooltip";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";
import { BrowserRouter, Navigate, Outlet, Route, Routes, useLocation } from "react-router-dom";
import { Toaster } from "sonner";
import "./App.css";

// Import pages
import AuthGuard from "./components/AuthGuard";
import { AccountProvider } from "./contexts/AccountContext";
import { useAccount } from "./contexts/AccountContext";
import { PermissionsProvider } from "./contexts/PermissionsContext";
import { isCustomComponentsEnabled } from "./lib/env";
import { Login } from "./pages/auth/Login";
import OrganizationCreate from "./pages/auth/OrganizationCreate";
import OrganizationSelect from "./pages/auth/OrganizationSelect";
import OwnerSetup from "./pages/auth/OwnerSetup";
import { CustomComponent } from "./pages/custom-component";
import { CreateCanvasPage } from "./pages/canvas/CreateCanvasPage";
import HomePage from "./pages/home";
import NodeRunPage from "./pages/node-run";
import { OrganizationSettings } from "./pages/organization/settings";
import { WorkflowPageV2 } from "./pages/workflowv2";
import InviteLinkAccept from "./pages/auth/InviteLinkAccept";

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

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <AccountProvider>
        <TooltipProvider delayDuration={150}>
          <AppRouter />
        </TooltipProvider>
        <Toaster position="bottom-center" closeButton />
      </AccountProvider>
    </QueryClientProvider>
  );
}

function AppRouter() {
  return (
    <BrowserRouter>
      <SetupGuard>
        <Routes>
          {/* public routes */}
          <Route path="login" element={<Login />} />
          <Route path="create" element={<OrganizationCreate />} />
          <Route path="setup" element={<OwnerSetup />} />

          {/* Organization selection and creation */}
          <Route path="" element={withAuthOnly(OrganizationSelect)} />

          {/* Invite link acceptance */}
          <Route path="invite/:token" element={withAuthOnly(InviteLinkAccept)} />

          {/* Organization-scoped protected routes */}
          <Route path=":organizationId" element={<OrganizationScope />}>
            <Route index element={withAuthOnly(HomePage)} />
            <Route path="canvases/new" element={withAuthOnly(CreateCanvasPage)} />
            {isCustomComponentsEnabled() && (
              <Route path="custom-components/:blueprintId" element={withAuthOnly(CustomComponent)} />
            )}
            <Route path="canvases/:canvasId" element={withAuthOnly(WorkflowPageV2)} />
            <Route path="templates/:canvasId" element={withAuthOnly(WorkflowPageV2)} />
            <Route path="canvases/:canvasId/nodes/:nodeId/:executionId" element={withAuthOnly(NodeRunPage)} />
            <Route path="settings/*" element={withAuthOnly(OrganizationSettings)} />
          </Route>

          {/* Catch-all route */}
          <Route path="*" element={<Navigate to="/" />} />
        </Routes>
      </SetupGuard>
    </BrowserRouter>
  );
}

function OrganizationScope() {
  return (
    <PermissionsProvider>
      <Outlet />
    </PermissionsProvider>
  );
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
