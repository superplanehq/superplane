import React from 'react'
import { Avatar } from '../Avatar/avatar'
import { Badge } from '../Badge/badge'
import { Text } from '../Text/text'
import { MaterialSymbol } from '../MaterialSymbol/material-symbol'
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

export interface UserOrgDropdownProps {
  user: User
  organization: Organization
  onUserMenuAction?: (action: 'profile' | 'settings' | 'signout') => void
  onOrganizationMenuAction?: (action: 'settings' | 'billing' | 'members' | 'groups' | 'roles') => void
  className?: string
  plain?: boolean
}

export function UserOrgDropdown({
  user,
  organization,
  onUserMenuAction,
  onOrganizationMenuAction,
  className = '',
  plain = false
}: UserOrgDropdownProps) {
  return (
    <Dropdown>
      <DropdownButton 
        plain
        className={`flex items-center gap-4 rounded-md border border-zinc-200 dark:border-zinc-800 text-zinc-700 hover:text-zinc-900 dark:text-zinc-300 dark:hover:text-white ${className}`}
      >
        {/* Organization Avatar with User Avatar Overlay */}
        <div className='flex items-center gap-2 mr-2'>
        {organization.avatar ? (
          <img
              src={organization.avatar}
              alt={organization.name}
              className="w-8 h-8 object-contain"
            />
            
        ) : (
          <Avatar
            initials={organization.initials}
            alt={organization.name}
            className="w-8 h-8"
          />
        )}
          <span className='font-normal text-sm'>{organization.name}</span>
        </div>
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
              initials={organization.initials}
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
          <DropdownItem onClick={() => onOrganizationMenuAction?.('settings')} href='/settings'>
            <span className="flex items-center gap-x-2">
              <MaterialSymbol name="business" data-slot="icon" size='sm'/>
              <DropdownLabel>Organization Settings</DropdownLabel>
            </span>
          </DropdownItem>
          
          <DropdownItem onClick={() => onOrganizationMenuAction?.('members')} href='/settings/members'>
            <span className="flex items-center gap-x-2">
              <MaterialSymbol name="person" data-slot="icon" size='sm'/>
              <DropdownLabel>Members</DropdownLabel>
            </span>
          </DropdownItem>
          
          <DropdownItem onClick={() => onOrganizationMenuAction?.('groups')} href='/settings/groups'>
            <span className="flex items-center gap-x-2">
              <MaterialSymbol name="group" data-slot="icon" size='sm'/>
              <DropdownLabel>Groups</DropdownLabel>
            </span>
          </DropdownItem>
          
          <DropdownItem onClick={() => onOrganizationMenuAction?.('roles')} href='/settings/roles'>
            <span className="flex items-center gap-x-2">
              <MaterialSymbol name="shield" data-slot="icon" size='sm'/>
              <DropdownLabel>Roles</DropdownLabel>
            </span>
          </DropdownItem>
          
          <DropdownItem onClick={() => onOrganizationMenuAction?.('billing')} href='/settings/billing'>
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
  )
}