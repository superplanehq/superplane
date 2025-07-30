import React from 'react'
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import './App.css'

// Import pages
import HomePage from './pages/home'
import { Canvas } from './pages/canvas'
import OrganizationPage from './pages/organization'
import { OrganizationSettings } from './pages/organization/settings'
import Navigation from './components/Navigation'

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

// Get the base URL from environment or default to '/app' for production
const BASE_PATH = import.meta.env.BASE_URL || '/app'

// Helper function to wrap components with Navigation
const withNavigation = (Component: React.ComponentType) => (
  <>
    <Navigation />
    <Component />
  </>
)

// Main App component with router
function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter basename={BASE_PATH}>
        <Routes>
          <Route path="" element={withNavigation(HomePage)} />
          <Route path="organization/:orgId" element={withNavigation(OrganizationPage)} />
          <Route path="organization/:orgId/canvas/:canvasId" element={withNavigation(Canvas)} />
          <Route path="organization/:orgId/settings/*" element={withNavigation(OrganizationSettings)} />
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  )
}

export default App
