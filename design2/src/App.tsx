import { useState, useEffect } from 'react'
import { LoginPage } from './components/LoginPage'
import { OrganizationPage } from './components/OrganizationPage'
import { OrganizationPageSidebar } from './components/OrganizationPageSidebar'
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

  if (!isLoggedIn) {
    return <LoginPage onLogin={() => setIsLoggedIn(true)} />
  }

  // Route to different organization pages based on path
  if (currentPath === '/org-sidebar') {
    return <OrganizationPageSidebar onSignOut={() => setIsLoggedIn(false)} />
  }

  // Default to original organization page
  return <OrganizationPage onSignOut={() => setIsLoggedIn(false)} />
}

export default App
