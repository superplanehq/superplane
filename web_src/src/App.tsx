import { TooltipProvider } from "@/ui/tooltip";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";
import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { Toaster } from "sonner";
import "./App.css";

// Import pages
import AuthGuard from "./components/AuthGuard";
import { AccountProvider } from "./contexts/AccountContext";
import { isCustomComponentsEnabled } from "./lib/env";
import EmailLogin from "./pages/auth/EmailLogin";
import { Login } from "./pages/auth/Login";
import OrganizationCreate from "./pages/auth/OrganizationCreate";
import OrganizationSelect from "./pages/auth/OrganizationSelect";
import OwnerSetup from "./pages/auth/OwnerSetup";
import { CustomComponent } from "./pages/custom-component";
import HomePage from "./pages/home";
import NodeRunPage from "./pages/node-run";
import { OrganizationSettings } from "./pages/organization/settings";
import { WorkflowPageV2 } from "./pages/workflowv2";

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

// Main App component with router
function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <AccountProvider>
        <TooltipProvider delayDuration={150}>
          <BrowserRouter>
            <Routes>
              {/* Organization-scoped protected routes */}
              <Route path=":organizationId" element={withAuthOnly(HomePage)} />
              {isCustomComponentsEnabled() && (
                <Route path=":organizationId/custom-components/:blueprintId" element={withAuthOnly(CustomComponent)} />
              )}
              <Route path=":organizationId/workflows/:workflowId" element={withAuthOnly(WorkflowPageV2)} />
              <Route
                path=":organizationId/workflows/:workflowId/nodes/:nodeId/:executionId"
                element={withAuthOnly(NodeRunPage)}
              />
              <Route path=":organizationId/settings/*" element={withAuthOnly(OrganizationSettings)} />
              {/* Organization selection and creation */}
              <Route path="login" element={<Login />} />
              <Route path="login/email" element={<EmailLogin />} />
              <Route path="create" element={<OrganizationCreate />} />
              <Route path="setup" element={<OwnerSetup />} />
              <Route path="" element={<OrganizationSelect />} />
              <Route path="*" element={<Navigate to="/" />} />
            </Routes>
          </BrowserRouter>
        </TooltipProvider>
        <Toaster position="bottom-center" closeButton />
      </AccountProvider>
    </QueryClientProvider>
  );
}

export default App;
