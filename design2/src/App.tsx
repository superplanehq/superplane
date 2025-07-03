import { useState, useEffect } from 'react'
import { LoginPage } from './components/LoginPage'
import { WorkspacesPage } from './components/WorkspacesPage'
import { DashboardPage } from './components/DashboardPage'
import { OrganizationPage } from './components/OrganizationPage'
import { OrganizationPageSidebar } from './components/OrganizationPageSidebar'
import { StudioPage } from './components/StudioPage'
import { AdministrationPage } from './components/AdministrationPage'
import './App.css'

function App() {
  const [isLoggedIn, setIsLoggedIn] = useState(false)
  const [currentPath, setCurrentPath] = useState(window.location.pathname)

  useEffect(() => {
    const handlePopState = () => {
      setCurrentPath(window.location.pathname)
    }

    window.addEventListener('popstate', handlePopState)
    return () => window.removeEventListener('popstate', handlePopState)
  }, [])

  const handleWorkspaceSelect = (workspaceId: string) => {
    // Navigate to organization page when workspace is selected
    window.history.pushState(null, '', `/workspace/${workspaceId}`)
    setCurrentPath(`/workspace/${workspaceId}`)
  }


  if (!isLoggedIn) {
    return <LoginPage onLogin={() => setIsLoggedIn(true)} />
  }

  // Route based on current path
  if (currentPath === '/org-sidebar') {
    return <OrganizationPageSidebar onSignOut={() => setIsLoggedIn(false)} />
  }

  if (currentPath === '/org-tabs') {
    return <OrganizationPage onSignOut={() => setIsLoggedIn(false)} />
  }

  if (currentPath.startsWith('/workspace/')) {
    // For now, route to sidebar organization page for workspaces
    return <OrganizationPageSidebar onSignOut={() => setIsLoggedIn(false)} />
  }

  if (currentPath === '/studio') {
    return (
      <StudioPage 
        onSignOut={() => setIsLoggedIn(false)} 
      />
    )
  }

  if (currentPath === '/administration') {
    return (
      <AdministrationPage 
        onSignOut={() => setIsLoggedIn(false)} 
      />
    )
  }

  // Special route for dashboard
  if (currentPath === '/dashboard') {
    return (
      <DashboardPage 
        onSignOut={() => setIsLoggedIn(false)} 
        onWorkspaceSelect={handleWorkspaceSelect}
      />
    )
  }

  // Default to workspaces page after login
  return (
    <WorkspacesPage 
      onSignOut={() => setIsLoggedIn(false)} 
      onWorkspaceSelect={handleWorkspaceSelect}
    />
  )
}

export default App
