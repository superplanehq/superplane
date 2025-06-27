import { useState } from 'react'
import { LoginPage } from './components/LoginPage'
import { OrganizationPage } from './components/OrganizationPage'
import './App.css'

function App() {
  const [isLoggedIn, setIsLoggedIn] = useState(false)

  if (!isLoggedIn) {
    return <LoginPage onLogin={() => setIsLoggedIn(true)} />
  }

  return <OrganizationPage onSignOut={() => setIsLoggedIn(false)} />
}

export default App
