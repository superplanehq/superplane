import React, { useState } from 'react'
import { Button } from '../Button/button'
import { Avatar } from '../Avatar/avatar'
import { Badge } from '../Badge/badge'
import { Text } from '../Text/text'
import { Heading } from '../Heading/heading'
import clsx from 'clsx'
import { MaterialSymbol } from '../MaterialSymbol/material-symbol'

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

export interface NavigationLink {
  id: string
  label: string
  icon: React.ReactNode
  href?: string
  onClick?: () => void
  isActive?: boolean
  tooltip?: string
}

export interface NavigationVerticalProps {
  user: User
  organization?: Organization
  showOrganization?: boolean
  links?: NavigationLink[]
  onHelpClick?: () => void
  onConfigurationClick?: () => void
  onUserMenuAction?: (action: 'profile' | 'settings' | 'signout') => void
  onOrganizationMenuAction?: (action: 'settings' | 'billing' | 'members') => void
  onLinkClick?: (linkId: string) => void
  className?: string
}

export function NavigationVertical({
  user,
  organization,
  showOrganization = true,
  links = [],
  onHelpClick,
  onConfigurationClick,
  onUserMenuAction,
  onOrganizationMenuAction,
  onLinkClick,
  className
}: NavigationVerticalProps) {
  const [isUserMenuOpen, setIsUserMenuOpen] = useState(false)
  const [isOrgMenuOpen, setIsOrgMenuOpen] = useState(false)

  return (
    <nav className={clsx(
      'w-16 bg-white dark:bg-zinc-900 border-r border-zinc-200 dark:border-zinc-800 flex flex-col',
      className
    )}>
      {/* Top Section - Logo */}
      <div className="flex-shrink-0 flex items-center justify-center py-4">
        <div className="w-8 h-8 flex flex-col items-center justify-center">
          <span className="block text-slate-600 dark:text-teal-200 font-extrabold text-xl tracking-wider">SP</span>
        </div>
      </div>

      {/* Navigation Links */}
      {links.length > 0 && (
        <div className="flex-shrink-0 flex flex-col items-center space-y-4 px-2">
          {links.map((link) => (
            <NavigationLinkItem
              key={link.id}
              link={link}
              onLinkClick={onLinkClick}
            />
          ))}
        </div>
      )}

      {/* Middle Section - Spacer */}
      <div className="flex-1" />
      {/* Configuration Icon */}
      <div className="flex-shrink-0 flex items-center justify-center pb-4">
        <Button
          plain
          onClick={onConfigurationClick}
          className="w-8 h-8 flex items-center justify-center text-zinc-500 hover:text-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-300 hover:bg-zinc-100 dark:hover:bg-zinc-800 rounded-md"
          aria-label="Configuration"
        >
          <MaterialSymbol name="settings" size='lg'/>
        </Button>
      </div>

      {/* Help Icon */}
      <div className="flex-shrink-0 flex items-center justify-center pb-4">
        <Button
          plain
          onClick={onHelpClick}
          className="w-8 h-8 flex items-center justify-center text-zinc-500 hover:text-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-300 hover:bg-zinc-100 dark:hover:bg-zinc-800 rounded-md"
          aria-label="Help"
        >
          <MaterialSymbol name="help" size='lg'/>
        </Button>
      </div>

      {/* Bottom Section - Avatars */}
      <div className="flex-shrink-0 flex flex-col items-center space-y-3 pb-4">
        {/* Organization Avatar */}
        {showOrganization && organization && (
          <div className="relative">
            <Button
              plain
              onClick={() => setIsOrgMenuOpen(!isOrgMenuOpen)}
              className="relative"
            >
              <Avatar
                src={organization.avatar}
                initials={organization.initials}
                alt={organization.name}
                className="w-8 h-8"
              />
              {organization.plan && (
                <div className="absolute -top-1 -right-1 w-3 h-3 bg-blue-500 rounded-full flex items-center justify-center">
                  <div className="w-1 h-1 bg-white rounded-full" />
                </div>
              )}
            </Button>

            {/* Organization Dropdown Menu */}
            {isOrgMenuOpen && (
              <div className="absolute left-full bottom-0 ml-2 w-56 bg-white dark:bg-zinc-800 rounded-md shadow-lg ring-1 ring-black ring-opacity-5 z-50">
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
        )}

        {/* User Avatar */}
        <div className="relative">
          <Button
            plain
            onClick={() => setIsUserMenuOpen(!isUserMenuOpen)}
            className="relative"
          >
            <Avatar
              src={user.avatar}
              initials={user.initials}
              alt={user.name}
              className="w-8 h-8"
            />
            <div className="absolute -bottom-0.5 -right-0.5 w-3 h-3 bg-green-500 border-2 border-white dark:border-zinc-900 rounded-full" />
          </Button>

          {/* User Dropdown Menu */}
          {isUserMenuOpen && (
            <div className="absolute left-full bottom-0 ml-2 w-48 bg-white dark:bg-zinc-800 rounded-md shadow-lg ring-1 ring-black ring-opacity-5 z-50">
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

// Individual navigation link component
function NavigationLinkItem({ 
  link, 
  onLinkClick 
}: { 
  link: NavigationLink
  onLinkClick?: (linkId: string) => void 
}) {
  const handleClick = () => {
    if (link.onClick) {
      link.onClick()
    }
    if (onLinkClick) {
      onLinkClick(link.id)
    }
  }

  const buttonClasses = clsx(
    'relative p-1 flex flex-col rounded-md transition-all duration-200 ease-in-out gap-1 group leading-1',
    'text-zinc-500 hover:text-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-300'
  )

  const buttonContent = (
    <div className='flex flex-col items-center'>
      <div className={(
        link.isActive
        ? 'bg-blue-100 text-blue-600 dark:bg-blue-900/20 dark:text-blue-400' 
        : 'text-zinc-500 hover:text-zinc-700 hover:bg-zinc-100 dark:text-zinc-400 dark:hover:text-zinc-300 dark:hover:bg-zinc-800'
        ) + " w-7 h-7 block rounded-md"}>
        {link.icon}
      </div>
      <span className="text-tiny block mt-2 leading-0">{link.label}</span>
      
      {link.tooltip && (
        <div className="absolute left-full ml-2 px-2 py-1 bg-zinc-900 text-white text-xs rounded opacity-0 group-hover:opacity-100 transition-opacity duration-200 pointer-events-none z-50 whitespace-nowrap">
          {link.tooltip}
        </div>
      )}
    </div>
  )

  if (link.href) {
    return (
      <a
        href={link.href}
        className={buttonClasses}
        aria-label={link.label}
        title={link.tooltip || link.label}
        onClick={handleClick}
      >
        {buttonContent}
      </a>
    )
  }

  return (
    <button
      type="button"
      className={buttonClasses}
      onClick={handleClick}
      aria-label={link.label}
      title={link.tooltip || link.label}
    >
      {buttonContent}
      
    </button>
  )
}