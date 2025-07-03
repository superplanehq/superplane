import { useState, useEffect } from 'react'
import { LoginPage } from './components/LoginPage'
import { MainLandingPage } from './components/MainLandingPage'
import { CanvasesPage } from './components/CanvasesPage'
import { CanvasEditorPage } from './components/CanvasEditorPage'
import { WorkspacesPage } from './components/WorkspacesPage'
import { DashboardPage } from './components/DashboardPage'
import { OrganizationPage } from './components/OrganizationPage'
import { OrganizationPageSidebar } from './components/OrganizationPageSidebar'
import { StudioPage } from './components/StudioPage'
import { AdministrationPage } from './components/AdministrationPage'
import { MaterialSymbol } from './components/lib/MaterialSymbol/material-symbol'
import type { NavigationLink } from './components/lib/Navigation/navigation-vertical'
import './App.css'

function App() {
  const [isLoggedIn, setIsLoggedIn] = useState(false)
  const [currentPath, setCurrentPath] = useState(window.location.pathname)

  // Central navigation links configuration
  const navigationLinks: NavigationLink[] = [
    {
      id: 'home',
      label: 'Home',
      icon: <MaterialSymbol size='lg' opticalSize={20} weight={400} name="home" />,
      isActive: currentPath === '/',
      tooltip: 'Home'
    },
    {
      id: 'canvases',
      label: 'Canvases',
      icon: <MaterialSymbol size='lg' opticalSize={20} weight={400} name="automation" />,
      isActive: currentPath === '/canvases',
      tooltip: 'Canvases'
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

  if (currentPath === '/canvases') {
    return (
      <CanvasesPage 
        onSignOut={() => setIsLoggedIn(false)}
        navigationLinks={navigationLinks}
        onLinkClick={handleLinkClick}
      />
    )
  }

  // Canvas editor route
  if (currentPath.startsWith('/canvas/')) {
    const canvasId = currentPath.split('/canvas/')[1]
    return (
      <CanvasEditorPage 
        canvasId={canvasId}
        onSignOut={() => setIsLoggedIn(false)}
        navigationLinks={navigationLinks}
        onLinkClick={handleLinkClick}
        onBack={() => {
          window.history.pushState(null, '', '/canvases')
          setCurrentPath('/canvases')
        }}
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

  // Default to main landing page after login
  return (
    <MainLandingPage 
      onSignOut={() => setIsLoggedIn(false)}
      navigationLinks={navigationLinks}
      onLinkClick={handleLinkClick}
    />
  )
}

export default App
