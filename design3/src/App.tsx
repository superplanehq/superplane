import { useState, useEffect } from 'react'
import { LoginPage } from './components/LoginPage'
import { HomePage } from './components/HomePage'
import { SettingsPage } from './components/SettingsPage'
import { MainLandingPage } from './components/MainLandingPage'
import { CanvasesPage } from './components/CanvasesPage'
import { CanvasEditorPage } from './components/CanvasEditorPage'
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

  if (currentPath === '/settings') {
    return (
      <SettingsPage 
        onSignOut={() => setIsLoggedIn(false)}
        navigationLinks={navigationLinks}
        onLinkClick={handleLinkClick}
        onConfigurationClick={handleConfigurationClick}
      />
    )
  }

  if (currentPath === '/canvases') {
    return (
      <CanvasesPage 
        onSignOut={() => setIsLoggedIn(false)}
        navigationLinks={navigationLinks}
        onLinkClick={handleLinkClick}
        onConfigurationClick={handleConfigurationClick}
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
        onConfigurationClick={handleConfigurationClick}
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

  // Default to home page after login
  return (
    <HomePage 
      onSignOut={() => setIsLoggedIn(false)}
      navigationLinks={navigationLinks}
      onLinkClick={handleLinkClick}
      onConfigurationClick={handleConfigurationClick}
    />
  )
}

export default App
