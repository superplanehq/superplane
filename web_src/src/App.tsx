import React from 'react'
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { ToastContainer } from 'react-toastify'
import 'react-toastify/dist/ReactToastify.css'
import './App.css'

// Import pages
import HomePage from './pages/home'
import { Canvas } from './pages/canvas'
import { Blueprint } from './pages/blueprint'
import { OrganizationSettings } from './pages/organization/settings'
import Navigation from './components/Navigation'
import AuthGuard from './components/AuthGuard'
import OrganizationSelect from './pages/auth/OrganizationSelect'
import OrganizationCreate from './pages/auth/OrganizationCreate'
import { AccountProvider } from './contexts/AccountContext'

// Create a client
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      staleTime: 5 * 60 * 1000, // 5 minutes
      gcTime: 10 * 60 * 1000, // 10 minutes
    },
  },
})

// Helper function to wrap components with Navigation and Auth Guard
const withAuthAndNavigation = (Component: React.ComponentType) => (
  <AuthGuard>
    <Navigation />
    <Component />
  </AuthGuard>
)

const withAuthOnly = (Component: React.ComponentType) => (
  <AuthGuard>
    <Component />
  </AuthGuard>
)

// Main App component with router
function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <AccountProvider>
        <BrowserRouter>
          <Routes>

            {/* Organization-scoped protected routes */}
            <Route path=":organizationId" element={withAuthAndNavigation(HomePage)} />
            <Route path=":organizationId/canvas/:canvasId" element={withAuthOnly(Canvas)} />
            <Route path=":organizationId/blueprints/:blueprintId" element={withAuthOnly(Blueprint)} />
            <Route path=":organizationId/settings/*" element={withAuthAndNavigation(OrganizationSettings)} />

            {/* Organization selection and creation */}
            <Route path="create" element={<OrganizationCreate />} />
            <Route path="" element={<OrganizationSelect />} />
          </Routes>
        </BrowserRouter>
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
  )
}

export default App
