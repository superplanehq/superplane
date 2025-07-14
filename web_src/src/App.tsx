import { BrowserRouter, Routes, Route } from 'react-router-dom'
import './App.css'

// Import pages
import HomePage from './pages/home'
import { Canvas } from './pages/canvas'
import OrganizationPage from './pages/organization'
import Navigation from './components/Navigation'

// Get the base URL from environment or default to '/app' for production
const BASE_PATH = import.meta.env.BASE_URL || '/app'

// Main App component with router
function App() {
  return (
    <>
      <BrowserRouter basename={BASE_PATH}>
        <Navigation />
        <Routes>
          <Route path="" element={<HomePage />} />
          <Route path="organization/:id" element={<OrganizationPage />} />
          <Route path="organization/:orgId/canvas/:canvasId" element={<Canvas />} />
        </Routes>
      </BrowserRouter>
    </>
  )
}

export default App
