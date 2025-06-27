import { useState } from 'react'
import { LoginPage } from './components/LoginPage'
import { Button } from './lib/components/Button/button'
import './App.css'

function App() {
  const [isLoggedIn, setIsLoggedIn] = useState(false)

  if (!isLoggedIn) {
    return <LoginPage onLogin={() => setIsLoggedIn(true)} />
  }

  return (
    <div className="min-h-screen bg-zinc-50 dark:bg-zinc-900 flex flex-col items-center justify-center p-8">
      <div className="max-w-md w-full text-center space-y-8">
        <div>
          <h1 className="text-4xl font-bold text-zinc-900 dark:text-white mb-4">
            Welcome to the Dashboard
          </h1>
          <p className="text-lg text-zinc-600 dark:text-zinc-400">
            You have successfully logged in!
          </p>
        </div>
        
        <div className="space-y-4">
          <Button 
            onClick={() => setIsLoggedIn(false)}
            color="red"
            className="w-full"
          >
            Sign Out
          </Button>
          
          <Button 
            onClick={() => window.open('http://localhost:6006', '_blank')}
            outline
            className="w-full"
          >
            View Component Library
          </Button>
        </div>
        
        <div className="text-sm text-zinc-500 dark:text-zinc-400">
          <p>This is a demo dashboard page.</p>
          <p>Check out Storybook for component documentation.</p>
        </div>
      </div>
    </div>
  )
}

export default App
