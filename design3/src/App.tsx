import { useState, useEffect } from 'react'
import { LoginPage } from './components/LoginPage'
import { OrganizationApp } from './components/OrganizationApp'
import { HomePage } from './components/HomePage'
import { SettingsPage } from './components/SettingsPage'
import { MainLandingPage } from './components/MainLandingPage'
import { CanvasesPage } from './components/CanvasesPage'
import { CanvasEditorPage } from './components/CanvasEditorPage'
import { CanvasEditorPage2 } from './components/CanvasEditorPage2'
import { CanvasEditorPage3 } from './components/CanvasEditorPage3'
import { CanvasEditorPage4 } from './components/CanvasEditorPage4'
import { CanvasMembersPage } from './components/CanvasMembersPage'
import { DashboardPage } from './components/DashboardPage'
import { OrganizationPage } from './components/OrganizationPage'
import { OrganizationPageSidebar } from './components/OrganizationPageSidebar'
import { StudioPage } from './components/StudioPage'
import { AdministrationPage } from './components/AdministrationPage'
import { MaterialSymbol } from './components/lib/MaterialSymbol/material-symbol'
import type { NavigationLink } from './components/lib/Navigation/navigation-vertical'
import './App.css'
import { OrganizationSettings } from './components/OrganizationSettings'
import { CreateOrganizationPage } from './components/CreateOrganizationPage'

function App() {
  // Check localStorage for existing authentication state
  const [isLoggedIn, setIsLoggedIn] = useState(() => {
    const savedAuth = localStorage.getItem('superplane_auth')
    return savedAuth === 'true'
  })
  const [currentPath, setCurrentPath] = useState(window.location.pathname)
  
  // Check if user has an organization
  const [hasOrganization, setHasOrganization] = useState(() => {
    const savedOrg = localStorage.getItem('superplane_organization')
    return savedOrg !== null
  })

  // Central navigation links configuration
  const navigationLinks: NavigationLink[] = [
    {
      id: 'home',
      label: 'Home',
      tooltip: 'Home',
      icon: <MaterialSymbol size='lg' opticalSize={20} weight={400} name="home" />,
      isActive: currentPath === '/',
    },
    {
      id: 'canvases',
      label: 'Canvases',
      tooltip: 'Canvases',
      icon: <MaterialSymbol size='lg' opticalSize={20} weight={400} name="automation" />,
      isActive: currentPath === '/canvases',
    }
  ]

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

  const handleLinkClick = (linkId: string) => {
    console.log(`Navigation link clicked: ${linkId}`)
    switch (linkId) {
      case 'home':
        window.history.pushState(null, '', '/')
        setCurrentPath('/')
        break
      case 'canvases':
        window.history.pushState(null, '', '/canvases')
        setCurrentPath('/canvases')
        break
    }
  }

  const handleConfigurationClick = () => {
    console.log('Configuration button clicked')
    window.history.pushState(null, '', '/settings')
    setCurrentPath('/settings')
  }
  

  // Handle user login
  const handleLogin = () => {
    setIsLoggedIn(true)
    localStorage.setItem('superplane_auth', 'true')
    // Optionally store additional user data
    localStorage.setItem('superplane_login_timestamp', new Date().toISOString())
    
    // Navigate to root path after login - will redirect to create org if no org exists
    window.history.pushState(null, '', '/')
    setCurrentPath('/')
  }

  // Handle user logout
  const handleLogout = () => {
    setIsLoggedIn(false)
    localStorage.removeItem('superplane_auth')
    localStorage.removeItem('superplane_login_timestamp')
    
    // Reset GitHub login state
    localStorage.removeItem('github_auth_token')
    localStorage.removeItem('github_user_data')
    localStorage.removeItem('github_login_timestamp')
    
    // Reset organization state
    localStorage.removeItem('superplane_organization')
    setHasOrganization(false)
    
    // Navigate to root path on logout
    window.history.pushState(null, '', '/')
    setCurrentPath('/')
  }

  // Handle organization creation
  const handleOrganizationCreated = (organizationData: { name: string; url: string }) => {
    // Store organization data
    localStorage.setItem('superplane_organization', JSON.stringify(organizationData))
    setHasOrganization(true)
    
    // Navigate to home page after organization creation
    window.history.pushState(null, '', '/')
    setCurrentPath('/')
  }


  if (!isLoggedIn) {
    return <LoginPage onLogin={handleLogin} />
  }

  // If logged in but no organization, show create organization page
  if (!hasOrganization) {
    return <CreateOrganizationPage onSuccess={handleOrganizationCreated} />
  }

  // Route based on current path
  if (currentPath === '/create-organization') {
    return <CreateOrganizationPage onSuccess={handleOrganizationCreated} />
  }
  
  if (currentPath === '/settings') {
    return <OrganizationSettings onSignOut={handleLogout} />
  }
 
  // Route based on current path
  if (currentPath === '/org-sidebar') {
    return <OrganizationPageSidebar onSignOut={handleLogout} />
  }
  
  if (currentPath === '/org-tabs') {
    return <OrganizationPage onSignOut={handleLogout} />
  }

  if (currentPath.startsWith('/workspace/')) {
    // For now, route to sidebar organization page for workspaces
    return <OrganizationPageSidebar onSignOut={handleLogout} />
  }

  if (currentPath === '/studio') {
    return (
      <StudioPage 
        onSignOut={handleLogout} 
      />
    )
  }

  if (currentPath === '/administration') {
    return (
      <AdministrationPage 
        onSignOut={handleLogout} 
      />
    )
  }

  if (currentPath === '/settings2') {
    return (
      <SettingsPage 
        onSignOut={handleLogout}
        navigationLinks={navigationLinks}
        onLinkClick={handleLinkClick}
        onConfigurationClick={handleConfigurationClick}
      />
    )
  }

  if (currentPath === '/canvases') {
    return (
      <CanvasesPage 
        onSignOut={handleLogout}
        
      />
    )
  }

  // Canvas members route - must come before canvas editor route
  if (currentPath.match(/^\/canvas\/[^\/]+\/members$/)) {
    const canvasId = currentPath.split('/canvas/')[1].split('/members')[0]
    return (
      <CanvasMembersPage 
        canvasId={canvasId}
        onBack={() => {
          window.history.pushState(null, '', `/canvas/${canvasId}`)
          setCurrentPath(`/canvas/${canvasId}`)
        }}
      />
    )
  }

  // Canvas editor route
  if (currentPath.startsWith('/canvas/')) {
    const canvasId = currentPath.split('/canvas/')[1]
    return (
      <CanvasEditorPage 
        canvasId={canvasId}
        onBack={() => {
          window.history.pushState(null, '', '/canvases')
          setCurrentPath('/canvases')
        }}
      />
    )
  }
  // Canvas editor route
  if (currentPath.startsWith('/canvas2/')) {
    const canvasId = currentPath.split('/canvas2/')[1]
    return (
      <CanvasEditorPage2 
        canvasId={canvasId}
        onBack={() => {
          window.history.pushState(null, '', '/canvases')
          setCurrentPath('/canvases')
        }}
      />
    )
  }
  // Canvas editor route
  if (currentPath.startsWith('/canvas3/')) {
    const canvasId = currentPath.split('/canvas3/')[1]
    return (
      <CanvasEditorPage3 
        canvasId={canvasId}
        onBack={() => {
          window.history.pushState(null, '', '/canvases')
          setCurrentPath('/canvases')
        }}
      />
    )
  }
  
  // Canvas editor route 4
  if (currentPath.startsWith('/canvas4/')) {
    const canvasId = currentPath.split('/canvas4/')[1]
    return (
      <CanvasEditorPage4 
        canvasId={canvasId}
        onBack={() => {
          window.history.pushState(null, '', '/canvases')
          setCurrentPath('/canvases')
        }}
      />
    )
  }

  // Special route for legacy dashboard
  if (currentPath === '/legacy-dashboard') {
    return (
      <DashboardPage 
        onWorkspaceSelect={handleWorkspaceSelect}
      />
    )
  }

  // Route for legacy home page
  if (currentPath === '/legacy-home') {
    return (
      <HomePage 
      />
    )
  }

  // Route for main landing page (workspaces view)
  if (currentPath === '/workspaces') {
    return (
      <MainLandingPage 
        
      />
    )
  }

  // Default to organization app after login
  return (
    <HomePage  
     
    />
  )
}

export default App
