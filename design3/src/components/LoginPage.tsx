import { useState } from 'react'
import { Button } from './lib/Button/button'
import { Input, InputGroup } from './lib/Input/input'
import { Checkbox } from './lib/Checkbox/checkbox'
import { Text, TextLink } from './lib/Text/text'
import { Heading } from './lib/Heading/heading'
import { Divider } from './lib/Divider/divider'
import { Label, Field } from '@headlessui/react'
  
interface LoginPageProps {
  onLogin?: () => void
}

export function LoginPage({ onLogin }: LoginPageProps = {}) {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [rememberMe, setRememberMe] = useState(false)
  const [isLoading, setIsLoading] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setIsLoading(true)
    
    // Simulate login request
    await new Promise(resolve => setTimeout(resolve, 1000))
    
    console.log('Login attempt:', { email, password, rememberMe })
    setIsLoading(false)
    
    // Call onLogin callback if provided
    if (onLogin) {
      onLogin()
    }
  }

  return (
    <div className="flex flex-col items-stretch min-h-screen px-8 py-8 dark:bg-black">
      <div className="flex-grow flex flex-col bg-zinc-100 dark:bg-zinc-950 py-12 sm:px-6 lg:px-8">
        <div className="sm:mx-auto sm:w-full sm:max-w-lg mb-8">
          <Heading level={1} className="text-center">SuperPlane</Heading>

        </div>

        <div className="sm:mx-auto sm:w-full sm:max-w-lg">
          <div className="bg-white dark:bg-zinc-900 py-10 px-4 shadow-sm rounded-xl border border-zinc-200 dark:border-zinc-800 sm:px-10">
            <form className="space-y-8" onSubmit={handleSubmit}>
              <Text className="text-zinc-600 dark:text-zinc-400 text-center">
              <span className='text-lg sm:text-lg font-bold'>Log in to SuperPlane</span>
              </Text>
              <div>
                <Button 
                  type="submit"
                  outline
                  className="flex items-center w-full text-lg px-6 py-3"
                >
                  <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" fill="currentColor" viewBox="0 0 16 16">
                    <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27s1.36.09 2 .27c1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.01 8.01 0 0 0 16 8c0-4.42-3.58-8-8-8"/>
                  </svg>
                  <span className="ml-2 text-lg">Continue with GitHub</span>
                </Button>
              </div>
            </form>

            <div className="mt-8">
              <Text className="text-center text-xs sm:text-xs leading-5 mb-0">
                By continuing, you agree to our <TextLink href="#">Terms of Service</TextLink> and <TextLink href="#">Privacy policy</TextLink>
                
              </Text>
            </div>
          </div>
        </div>
      </div>
      <Text className="absolute bottom-10 left-0 right-0 text-center text-xs sm:text-xs leading-5 mb-0">
        © 2025 SuperPlane. All rights reserved.
      </Text>
    </div>
  )
}