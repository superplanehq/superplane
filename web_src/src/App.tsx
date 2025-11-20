import { TooltipProvider } from "@/ui/tooltip";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";
import { BrowserRouter, Route, Routes } from "react-router-dom";
import { ToastContainer } from "react-toastify";
import "react-toastify/dist/ReactToastify.css";
import "./App.css";

// Import pages
import AuthGuard from "./components/AuthGuard";
import Navigation from "./components/Navigation";
import { AccountProvider } from "./contexts/AccountContext";
import OrganizationCreate from "./pages/auth/OrganizationCreate";
import OrganizationSelect from "./pages/auth/OrganizationSelect";
import { CustomComponent } from "./pages/custom-component";
import HomePage from "./pages/home";
import { OrganizationSettings } from "./pages/organization/settings";
import { WorkflowPageV2 } from "./pages/workflowv2";
import NodeRunPage from "./pages/node-run";

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

// Helper function to wrap components with Navigation and Auth Guard
const withAuthAndNavigation = (Component: React.ComponentType) => (
  <AuthGuard>
    <Navigation />
    <Component />
  </AuthGuard>
);

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
              <Route path=":organizationId" element={withAuthAndNavigation(HomePage)} />
              <Route path=":organizationId/custom-components/:blueprintId" element={withAuthOnly(CustomComponent)} />
              <Route path=":organizationId/workflows/:workflowId" element={withAuthOnly(WorkflowPageV2)} />
              <Route
                path=":organizationId/workflows/:workflowId/nodes/:nodeId/:executionId"
                element={withAuthOnly(NodeRunPage)}
              />
              <Route path=":organizationId/settings/*" element={withAuthAndNavigation(OrganizationSettings)} />

              {/* Organization selection and creation */}
              <Route path="create" element={<OrganizationCreate />} />
              <Route path="" element={<OrganizationSelect />} />
            </Routes>
          </BrowserRouter>
        </TooltipProvider>
        <ToastContainer
          position="bottom-center"
          autoClose={5000}
          hideProgressBar={false}
          newestOnTop={false}
          closeOnClick={true}
          rtl={false}
          pauseOnFocusLoss={true}
          draggable={true}
          pauseOnHover={true}
          closeButton={false}
          theme="auto"
        />
      </AccountProvider>
    </QueryClientProvider>
  );
}

export default App;
