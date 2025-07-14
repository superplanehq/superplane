import React from 'react'
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import './App.css'

// Import pages
import HomePage from './pages/home'
import { Canvas } from './pages/canvas'
import OrganizationPage from './pages/organization'
import { OrganizationSettings } from './pages/organization/settings'
import Navigation from './components/Navigation'

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
    <>
      <BrowserRouter basename={BASE_PATH}>
        <Routes>
          <Route path="" element={withNavigation(HomePage)} />
          <Route path="organization/:orgId" element={withNavigation(OrganizationPage)} />
          <Route path="organization/:orgId/canvas/:canvasId" element={withNavigation(Canvas)} />
          <Route path="organization/:orgId/settings/*" element={withNavigation(OrganizationSettings)} />
        </Routes>
      </BrowserRouter>
    </>
  )
}

export default App
