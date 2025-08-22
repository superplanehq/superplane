import React, { useState } from 'react'
import { Button } from '../Button/button'
import { Heading } from '../Heading/heading'
import { MaterialSymbol } from '../MaterialSymbol/material-symbol'
import { Breadcrumbs, type BreadcrumbItem } from '../Breadcrumbs/breadcrumbs'
import { UserOrgDropdown, type User, type Organization } from '../UserOrgDropdown'
import clsx from 'clsx'
import { Link } from '../Link/link'

export interface NavigationOrgProps {
  user?: User
  organization?: Organization
  onHelpClick?: () => void
  onUserMenuAction?: (action: 'profile' | 'settings' | 'signout') => void
  onOrganizationMenuAction?: (action: 'settings' | 'billing' | 'members' | 'groups' | 'roles') => void
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
                  <UserOrgDropdown
                    user={user}
                    organization={organization}
                    onUserMenuAction={onUserMenuAction}
                    onOrganizationMenuAction={onOrganizationMenuAction}
                    plain
                  />
                )}
              </div>
            </div>
          </>
        ) : (
          // Default breadcrumbs layout
          <>
            <div className="flex justify-between items-center">
              {/* Left side - Logo and breadcrumbs */}
              <div className="flex items-center space-x-6">
                {/* Logo */}
                <Link href="/">
                  <Heading level={1} className="text-xl font-bold text-blue-600 dark:text-blue-400 mb-0">
                    SuperPlane
                  </Heading>
                </Link>
                
                {/* Breadcrumbs */}
                {breadcrumbs && breadcrumbs.length > 0 && (
                  <Breadcrumbs 
                    items={breadcrumbs} 
                    className=""
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
                  <UserOrgDropdown
                    user={user}
                    organization={organization}
                    onUserMenuAction={onUserMenuAction}
                    onOrganizationMenuAction={onOrganizationMenuAction}
                    plain
                  />
                )}
              </div>
            </div>
          </>
        )}
      </div>
    </nav>
  )
}