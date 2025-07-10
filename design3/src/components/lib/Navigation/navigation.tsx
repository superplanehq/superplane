import React, { useState } from 'react'
import { Button } from '../Button/button'
import { Avatar } from '../Avatar/avatar'
import { Badge } from '../Badge/badge'
import { Text } from '../Text/text'
import { Heading } from '../Heading/heading'
import clsx from 'clsx'

export interface User {
  id: string
  name: string
  email: string
  avatar?: string
  initials: string
}

export interface Organization {
  id: string
  name: string
  plan?: string
  avatar?: string
  initials: string
}

export interface NavigationProps {
  user: User
  organization: Organization
  onHelpClick?: () => void
  onUserMenuAction?: (action: 'profile' | 'settings' | 'signout') => void
  onOrganizationMenuAction?: (action: 'settings' | 'billing' | 'members') => void
  className?: string
}

export function Navigation({
  user,
  organization,
  onHelpClick,
  onUserMenuAction,
  onOrganizationMenuAction,
  className
}: NavigationProps) {
  const [isUserMenuOpen, setIsUserMenuOpen] = useState(false)
  const [isOrgMenuOpen, setIsOrgMenuOpen] = useState(false)

  return (
    <nav className={clsx(
      'bg-white dark:bg-zinc-900 border-b border-zinc-200 dark:border-zinc-800',
      className
    )}>
      <div className="px-4 sm:px-6 lg:px-8">
        <div className="flex justify-between items-center h-16">
          {/* Logo */}
          <div className="flex items-center">
            <div className="flex-shrink-0">
              <Heading level={1} className="text-xl font-bold text-blue-600 dark:text-blue-400 mb-0">
                SuperPlane
              </Heading>
            </div>
          </div>

          {/* Right side */}
          <div className="flex items-center space-x-4">
            {/* Help Icon */}
            <Button
              plain
              onClick={onHelpClick}
              className="text-zinc-500 hover:text-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-300"
              aria-label="Help"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8.228 9c.549-1.165 2.03-2 3.772-2 2.21 0 4 1.343 4 3 0 1.4-1.278 2.575-3.006 2.907-.542.104-.994.54-.994 1.093m0 3h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
            </Button>

            {/* Organization Dropdown */}
            <div className="relative">
              <Button
                plain
                onClick={() => setIsOrgMenuOpen(!isOrgMenuOpen)}
                className="flex items-center space-x-2 text-zinc-700 hover:text-zinc-900 dark:text-zinc-300 dark:hover:text-white"
              >
                <Avatar
                  src={organization.avatar}
                  initials={organization.initials}
                  alt={organization.name}
                  className="w-8 h-8"
                />
                <div className="hidden sm:block text-left">
                  <Text className="text-sm font-medium mb-0">{organization.name}</Text>
                  {organization.plan && (
                    <Badge color="blue" className="text-xs">{organization.plan}</Badge>
                  )}
                </div>
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                </svg>
              </Button>

              {/* Organization Dropdown Menu */}
              {isOrgMenuOpen && (
                <div className="absolute right-0 mt-2 w-56 bg-white dark:bg-zinc-800 rounded-md shadow-lg ring-1 ring-black ring-opacity-5 z-50">
                  <div className="py-1">
                    <div className="px-4 py-3 border-b border-zinc-200 dark:border-zinc-700">
                      <div className="flex items-center space-x-3">
                        <Avatar
                          src={organization.avatar}
                          initials={organization.initials}
                          alt={organization.name}
                          className="w-10 h-10"
                        />
                        <div>
                          <Text className="font-medium">{organization.name}</Text>
                          {organization.plan && (
                            <Badge color="blue" className="text-xs mt-1">{organization.plan}</Badge>
                          )}
                        </div>
                      </div>
                    </div>
                    
                    <button
                      onClick={() => {
                        onOrganizationMenuAction?.('settings')
                        setIsOrgMenuOpen(false)
                      }}
                      className="flex items-center w-full px-4 py-2 text-sm text-zinc-700 hover:bg-zinc-100 dark:text-zinc-300 dark:hover:bg-zinc-700"
                    >
                      <svg className="w-4 h-4 mr-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                      </svg>
                      Organization Settings
                    </button>
                    
                    <button
                      onClick={() => {
                        onOrganizationMenuAction?.('members')
                        setIsOrgMenuOpen(false)
                      }}
                      className="flex items-center w-full px-4 py-2 text-sm text-zinc-700 hover:bg-zinc-100 dark:text-zinc-300 dark:hover:bg-zinc-700"
                    >
                      <svg className="w-4 h-4 mr-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0" />
                      </svg>
                      Manage Members
                    </button>
                    
                    <button
                      onClick={() => {
                        onOrganizationMenuAction?.('billing')
                        setIsOrgMenuOpen(false)
                      }}
                      className="flex items-center w-full px-4 py-2 text-sm text-zinc-700 hover:bg-zinc-100 dark:text-zinc-300 dark:hover:bg-zinc-700"
                    >
                      <svg className="w-4 h-4 mr-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 10h18M7 15h1m4 0h1m-7 4h12a3 3 0 003-3V8a3 3 0 00-3-3H6a3 3 0 00-3 3v8a3 3 0 003 3z" />
                      </svg>
                      Billing & Plans
                    </button>
                  </div>
                </div>
              )}
            </div>

            {/* User Avatar Dropdown */}
            <div className="relative">
              <Button
                plain
                onClick={() => setIsUserMenuOpen(!isUserMenuOpen)}
                className="flex items-center space-x-2"
              >
                <Avatar
                  src={user.avatar}
                  initials={user.initials}
                  alt={user.name}
                  className="w-8 h-8"
                />
                
               
              </Button>

              {/* User Dropdown Menu */}
              {isUserMenuOpen && (
                <div className="absolute right-0 mt-2 w-48 bg-white dark:bg-zinc-800 rounded-md shadow-lg ring-1 ring-black ring-opacity-5 z-50">
                  <div className="py-1">
                    <div className="px-4 py-3 border-b border-zinc-200 dark:border-zinc-700">
                      <div className="flex items-center space-x-3">
                        <Avatar
                          src={user.avatar}
                          initials={user.initials}
                          alt={user.name}
                          className="w-10 h-10"
                        />
                        <div>
                          <Text className="font-medium">{user.name}</Text>
                          <Text className="text-sm text-zinc-500">{user.email}</Text>
                        </div>
                      </div>
                    </div>
                    
                    <button
                      onClick={() => {
                        onUserMenuAction?.('profile')
                        setIsUserMenuOpen(false)
                      }}
                      className="flex items-center w-full px-4 py-2 text-sm text-zinc-700 hover:bg-zinc-100 dark:text-zinc-300 dark:hover:bg-zinc-700"
                    >
                      <svg className="w-4 h-4 mr-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
                      </svg>
                      Your Profile
                    </button>
                    
                    <button
                      onClick={() => {
                        onUserMenuAction?.('settings')
                        setIsUserMenuOpen(false)
                      }}
                      className="flex items-center w-full px-4 py-2 text-sm text-zinc-700 hover:bg-zinc-100 dark:text-zinc-300 dark:hover:bg-zinc-700"
                    >
                      <svg className="w-4 h-4 mr-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                      </svg>
                      Account Settings
                    </button>
                    
                    <div className="border-t border-zinc-200 dark:border-zinc-700">
                      <button
                        onClick={() => {
                          onUserMenuAction?.('signout')
                          setIsUserMenuOpen(false)
                        }}
                        className="flex items-center w-full px-4 py-2 text-sm text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/20"
                      >
                        <svg className="w-4 h-4 mr-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
                        </svg>
                        Sign Out
                      </button>
                    </div>
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Click outside to close dropdowns */}
      {(isUserMenuOpen || isOrgMenuOpen) && (
        <div
          className="fixed inset-0 z-40"
          onClick={() => {
            setIsUserMenuOpen(false)
            setIsOrgMenuOpen(false)
          }}
        />
      )}
    </nav>
  )
}