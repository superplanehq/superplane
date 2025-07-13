import React, { useState } from 'react'
import { Button } from '../Button/button'
import { Avatar } from '../Avatar/avatar'
import { Badge } from '../Badge/badge'
import { Text } from '../Text/text'
import { Heading } from '../Heading/heading'
import { MaterialSymbol } from '../MaterialSymbol/material-symbol'
import { Breadcrumbs, type BreadcrumbItem } from '../Breadcrumbs/breadcrumbs'
import { 
  Dropdown, 
  DropdownButton, 
  DropdownMenu, 
  DropdownItem, 
  DropdownHeader, 
  DropdownDivider,
  DropdownSection,
  DropdownLabel
} from '../Dropdown/dropdown'
import clsx from 'clsx'
import { Link } from '../Link/link'

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

export interface NavigationOrgProps {
  user?: User
  organization?: Organization
  onHelpClick?: () => void
  onUserMenuAction?: (action: 'profile' | 'settings' | 'signout') => void
  onOrganizationMenuAction?: (action: 'settings' | 'billing' | 'members') => void
  className?: string
  breadcrumbs?: BreadcrumbItem[]
  breadcrumbsVariant?: 'default' | 'centered'
}

export function NavigationOrg({
  user,
  organization,
  onHelpClick,
  onUserMenuAction,
  onOrganizationMenuAction,
  className,
  breadcrumbs,
  breadcrumbsVariant = 'default'
}: NavigationOrgProps) {

  return (
    <nav className={clsx(
      'bg-white dark:bg-zinc-950 border-b border-zinc-200 dark:border-zinc-800',
      className
    )}>
      <div className="p-2">
        {breadcrumbsVariant === 'centered' ? (
          // Centered breadcrumbs layout
          <>
            <div className="flex justify-between items-center">
              {/* Logo */}
              <div className="flex-shrink-0">
                <Link href="/">
                  <Heading level={1} className="text-xl font-bold text-blue-600 dark:text-blue-400 mb-0 ml-3">
                    SuperPlane
                  </Heading>
                </Link>
              </div>
              {breadcrumbs && breadcrumbs.length > 0 && (
                <div className="flex items-center justify-center">
                  <Breadcrumbs 
                    items={breadcrumbs} 
                    className=""
                    showDivider={false}
                  />
                </div>
              )}
              {/* Right side */}
              <div className="flex items-center space-x-4">
                {/* Help Icon */}
                {onHelpClick && (
                  <Button
                    plain
                    onClick={onHelpClick}
                    className="text-zinc-500 hover:text-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-300 flex items-center"
                    aria-label="Help"
                  >
                    <MaterialSymbol name="help" size='lg'/>
                  </Button>
                )}

                {/* Merged Account Dropdown - only show if user and organization are provided */}
                {user && organization && (
                  <Dropdown>
                  <DropdownButton 
                    plain
                    className="flex items-center gap-x-2 rounded-md border bg-white dark:bg-zinc-950 border-zinc-200 dark:border-zinc-800 text-zinc-700 hover:text-zinc-900 dark:text-zinc-300 dark:hover:text-white"
                  >
                    {/* Organization Avatar with User Avatar Overlay */}
                    <img
                      src="https://upload.wikimedia.org/wikipedia/commons/a/ab/Confluent%2C_Inc._logo.svg"
                      alt="Confluent, Inc."
                      width={96}
                    />
                    
                    {/* User Avatar (smaller, overlapping in bottom-right) */}
                    <Avatar
                      src={user.avatar}
                      initials={user.initials}
                      alt={user.name}
                      className="w-7 h-7 bg-blue-700 dark:bg-blue-900 text-blue-100 dark:text-blue-100"
                    />
                  </DropdownButton>

                  <DropdownMenu className="w-64">
                    {/* User Section */}
                    <DropdownHeader>
                      <div className="flex items-center space-x-3">
                        <Avatar
                          src={user.avatar}
                          initials={user.initials}
                          alt={user.name}
                          className="size-8"
                        />
                        <div className="flex-1 min-w-0">
                          <Text className="font-medium truncate">{user.name}</Text>
                          <Text className="text-sm text-zinc-500 truncate">{user.email}</Text>
                        </div>
                      </div>
                    </DropdownHeader>

                    {/* User Actions */}
                    <DropdownSection>
                      <DropdownItem onClick={() => onUserMenuAction?.('profile')}>
                        <span className="flex items-center gap-x-2">
                          <MaterialSymbol name="person" data-slot="icon" size='sm'/>
                          <DropdownLabel>Your Profile</DropdownLabel>
                        </span>
                      </DropdownItem>
                      
                      <DropdownItem onClick={() => onUserMenuAction?.('settings')}>
                        <span className="flex items-center gap-x-2">
                          <MaterialSymbol name="settings" data-slot="icon" size='sm'/>
                          <DropdownLabel>Account Settings</DropdownLabel>
                        </span>
                      </DropdownItem>
                    </DropdownSection>

                    <DropdownDivider />

                    {/* Organization Section */}
                    <DropdownHeader>
                      <div className="flex items-center space-x-3">
                        <Avatar
                          src={organization.avatar}
                          alt={organization.name}
                          className="size-8"
                        />
                        <div className="flex-1 min-w-0">
                          <Text className="font-medium truncate">{organization.name}</Text>
                          {organization.plan && (
                            <Badge color="blue" className="text-xs mt-1">{organization.plan}</Badge>
                          )}
                        </div>
                      </div>
                    </DropdownHeader>

                    {/* Organization Actions */}
                    <DropdownSection>
                      <DropdownItem href='/settings'>
                        <span className="flex items-center gap-x-2">
                          <MaterialSymbol name="business" data-slot="icon" size='sm'/>
                          <DropdownLabel>Organization Settings</DropdownLabel>
                        </span>
                      </DropdownItem>
                      
                      <DropdownItem href='/settings/members'>
                        <span className="flex items-center gap-x-2">
                          <MaterialSymbol name="person" data-slot="icon" size='sm'/>
                          <DropdownLabel>Members</DropdownLabel>
                        </span>
                      </DropdownItem>
                      <DropdownItem href='/settings/groups'>
                        <span className="flex items-center gap-x-2">
                          <MaterialSymbol name="group" data-slot="icon" size='sm'/>
                          <DropdownLabel>Groups</DropdownLabel>
                        </span>
                      </DropdownItem>
                      <DropdownItem href='/settings/roles'>
                        <span className="flex items-center gap-x-2">
                          <MaterialSymbol name="shield" data-slot="icon" size='sm'/>
                          <DropdownLabel>Roles</DropdownLabel>
                        </span>
                      </DropdownItem>
                      
                      <DropdownItem href='/settings/billing'>
                        <span className="flex items-center gap-x-2">
                          <MaterialSymbol name="credit_card" data-slot="icon" size='sm'/>
                          <DropdownLabel>Billing & Plans</DropdownLabel>
                        </span>
                      </DropdownItem>
                    </DropdownSection>

                    <DropdownDivider />

                    {/* Sign Out Section */}
                    <DropdownSection>
                      <DropdownItem onClick={() => onUserMenuAction?.('signout')}>
                        <span className="flex items-center gap-x-2">
                          <MaterialSymbol name="logout" data-slot="icon" size='sm'/>
                          <DropdownLabel>Sign Out</DropdownLabel>
                        </span>
                      </DropdownItem>
                    </DropdownSection>
                  </DropdownMenu>
                </Dropdown>
                )}
              </div>
            </div>
            
            {/* Centered Breadcrumbs */}
            
          </>
        ) : (
          // Default layout with breadcrumbs next to logo
          <div className="flex justify-between items-center">
            {/* Logo and Breadcrumbs */}
            <div className="flex items-center space-x-4">
              <div className="flex-shrink-0">
                <Link href="/">
                  <Heading level={1} className="text-xl font-bold text-blue-600 dark:text-blue-400 mb-0 ml-3">
                    SuperPlane
                  </Heading>
                </Link>
              </div>
              
              {/* Breadcrumbs */}
              {breadcrumbs && breadcrumbs.length > 0 && (
                <Breadcrumbs 
                  items={breadcrumbs} 
                  className="ml-2"
                  showDivider={true}
                />
              )}
            </div>

            {/* Right side */}
            <div className="flex items-center space-x-4">
              {/* Help Icon */}
              {onHelpClick && (
                <Button
                  plain
                  onClick={onHelpClick}
                  className="text-zinc-500 hover:text-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-300 flex items-center"
                  aria-label="Help"
                >
                  <MaterialSymbol name="help" size='lg'/>
                </Button>
              )}

              {/* Merged Account Dropdown - only show if user and organization are provided */}
              {user && organization && (
                <Dropdown>
                <DropdownButton 
                  plain
                  className="flex items-center gap-x-2 bg-white dark:bg-zinc-950 rounded-md border border-zinc-200 dark:border-zinc-800 text-zinc-700 hover:text-zinc-900 dark:text-zinc-300 dark:hover:text-white"
                >
                  {/* Organization Avatar with User Avatar Overlay */}
                  <img
                    src="https://upload.wikimedia.org/wikipedia/commons/a/ab/Confluent%2C_Inc._logo.svg"
                    alt="Confluent, Inc."
                    width={96}
                  />
                  
                  {/* User Avatar (smaller, overlapping in bottom-right) */}
                  <Avatar
                    src={user.avatar}
                    initials={user.initials}
                    alt={user.name}
                    className="w-7 h-7 bg-blue-700 dark:bg-blue-900 text-blue-100 dark:text-blue-100"
                  />
                </DropdownButton>

                <DropdownMenu className="w-64">
                  {/* User Section */}
                  <DropdownHeader>
                    <div className="flex items-center space-x-3">
                      <Avatar
                        src={user.avatar}
                        initials={user.initials}
                        alt={user.name}
                        className="size-8"
                      />
                      <div className="flex-1 min-w-0">
                        <Text className="font-medium truncate">{user.name}</Text>
                        <Text className="text-sm text-zinc-500 truncate">{user.email}</Text>
                      </div>
                    </div>
                  </DropdownHeader>

                  {/* User Actions */}
                  <DropdownSection>
                    <DropdownItem onClick={() => onUserMenuAction?.('profile')}>
                      <span className="flex items-center gap-x-2">
                        <MaterialSymbol name="person" data-slot="icon" size='sm'/>
                        <DropdownLabel>Your Profile</DropdownLabel>
                      </span>
                    </DropdownItem>
                    
                    <DropdownItem onClick={() => onUserMenuAction?.('settings')}>
                      <span className="flex items-center gap-x-2">
                        <MaterialSymbol name="settings" data-slot="icon" size='sm'/>
                        <DropdownLabel>Account Settings</DropdownLabel>
                      </span>
                    </DropdownItem>
                  </DropdownSection>

                  <DropdownDivider />

                  {/* Organization Section */}
                  <DropdownHeader>
                    <div className="flex items-center space-x-3">
                      <Avatar
                        src={organization.avatar}
                        
                        alt={organization.name}
                        className="size-8"
                      />
                      <div className="flex-1 min-w-0">
                        <Text className="font-medium truncate">{organization.name}</Text>
                        {organization.plan && (
                          <Badge color="blue" className="text-xs mt-1">{organization.plan}</Badge>
                        )}
                      </div>
                    </div>
                  </DropdownHeader>

                  {/* Organization Actions */}
                  <DropdownSection>
                    <DropdownItem href='/settings'>
                      <span className="flex items-center gap-x-2">
                        <MaterialSymbol name="business" data-slot="icon" size='sm'/>
                        <DropdownLabel>Organization Settings</DropdownLabel>
                      </span>
                    </DropdownItem>
                    
                    <DropdownItem href='/settings/members'>
                      <span className="flex items-center gap-x-2">
                        <MaterialSymbol name="person" data-slot="icon" size='sm'/>
                        <DropdownLabel>Members</DropdownLabel>
                      </span>
                    </DropdownItem>
                    <DropdownItem href='/settings/groups'>
                      <span className="flex items-center gap-x-2">
                        <MaterialSymbol name="group" data-slot="icon" size='sm'/>
                        <DropdownLabel>Groups</DropdownLabel>
                      </span>
                    </DropdownItem>
                    <DropdownItem href='/settings/roles'>
                      <span className="flex items-center gap-x-2">
                        <MaterialSymbol name="shield" data-slot="icon" size='sm'/>
                        <DropdownLabel>Roles</DropdownLabel>
                      </span>
                    </DropdownItem>
                    
                    <DropdownItem href='/settings/billing'>
                      <span className="flex items-center gap-x-2">
                        <MaterialSymbol name="credit_card" data-slot="icon" size='sm'/>
                        <DropdownLabel>Billing & Plans</DropdownLabel>
                      </span>
                    </DropdownItem>
                  </DropdownSection>

                  <DropdownDivider />

                  {/* Sign Out Section */}
                  <DropdownSection>
                    <DropdownItem onClick={() => onUserMenuAction?.('signout')}>
                      <span className="flex items-center gap-x-2">
                        <MaterialSymbol name="logout" data-slot="icon" size='sm'/>
                        <DropdownLabel>Sign Out</DropdownLabel>
                      </span>
                    </DropdownItem>
                  </DropdownSection>
                </DropdownMenu>
              </Dropdown>
              )}
            </div>
          </div>
        )}
      </div>
    </nav>
  )
}