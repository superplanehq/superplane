import { useState } from 'react'
import { SidebarLayout } from './lib/SidebarLayout/sidebar-layout'
import { 
  Sidebar, 
  SidebarHeader, 
  SidebarBody, 
  SidebarFooter, 
  SidebarSection, 
  SidebarItem, 
  SidebarLabel,
  SidebarDivider
} from './lib/Sidebar/sidebar'
import { 
  Navbar, 
  NavbarSection, 
  NavbarSpacer, 
  NavbarItem,
  NavbarLabel 
} from './lib/Navbar/navbar'
import { Subheading } from './lib/Heading/heading'
import { Text } from './lib/Text/text'
import { Button } from './lib/Button/button'
import { MaterialSymbol } from './lib/MaterialSymbol/material-symbol'
import { Avatar } from './lib/Avatar/avatar'
import { Dropdown, DropdownButton, DropdownMenu, DropdownItem } from './lib/Dropdown/dropdown'
import type { NavigationLink } from './lib/Navigation/navigation-vertical'

interface HomePageProps {
  onSignOut?: () => void
  navigationLinks?: NavigationLink[]
  onLinkClick?: (linkId: string) => void
  onConfigurationClick?: () => void
}

export function HomePage({ 
  onSignOut, 
  navigationLinks = [], 
  onLinkClick,
  onConfigurationClick 
}: HomePageProps) {
  const [currentPage, setCurrentPage] = useState('home')

  // Mock user and organization data
  const currentUser = {
    id: '1',
    name: 'John Doe',
    email: 'john@superplane.com',
    initials: 'JD',
    avatar: 'https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?w=64&h=64&fit=crop&crop=face',
  }

  const currentOrganization = {
    id: '1',
    name: 'Development Team',
    plan: 'Pro Plan',
    initials: 'DT',
  }

  // Navigation handlers
  const handleUserMenuAction = (action: 'profile' | 'settings' | 'signout') => {
    switch (action) {
      case 'profile':
        console.log('Navigating to user profile...')
        break
      case 'settings':
        console.log('Opening account settings...')
        break
      case 'signout':
        onSignOut?.()
        break
    }
  }

  const handleNavClick = (linkId: string) => {
    setCurrentPage(linkId)
    if (onLinkClick) {
      onLinkClick(linkId)
    } else {
      console.log(`Navigation link clicked: ${linkId}`)
    }
  }

  // Create sidebar navigation
  const sidebarContent = (
    <Sidebar>
      <SidebarHeader>
        {/* Organization Header */}
        <div className="flex items-center gap-3 p-2">
          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-blue-500 text-white text-sm font-semibold">
            {currentOrganization.initials}
          </div>
          <div className="flex-1 min-w-0">
            <div className="text-sm font-semibold text-zinc-900 dark:text-white truncate">
              {currentOrganization.name}
            </div>
            <div className="text-xs text-zinc-500 dark:text-zinc-400">
              {currentOrganization.plan}
            </div>
          </div>
        </div>
      </SidebarHeader>

      <SidebarBody>
        <SidebarSection>
          {/* Home */}
          <SidebarItem 
            current={currentPage === 'home'} 
            onClick={() => handleNavClick('home')}
          >
            <MaterialSymbol name="home" data-slot="icon" />
            <SidebarLabel>Home</SidebarLabel>
          </SidebarItem>

          {/* Dynamic Navigation Links */}
          {navigationLinks.map((link) => (
            <SidebarItem 
              key={link.id}
              current={currentPage === link.id} 
              onClick={() => handleNavClick(link.id)}
            >
              {link.icon && <MaterialSymbol name={link.icon as any} data-slot="icon" />}
              <SidebarLabel>{link.label}</SidebarLabel>
            </SidebarItem>
          ))}
        </SidebarSection>

        <SidebarDivider />

        <SidebarSection>
          <SidebarItem onClick={() => console.log('Help clicked')}>
            <MaterialSymbol name="help" data-slot="icon" />
            <SidebarLabel>Help & Support</SidebarLabel>
          </SidebarItem>
        </SidebarSection>
      </SidebarBody>

      <SidebarFooter>
        <div className="flex items-center gap-3">
          <Avatar
            src={currentUser.avatar}
            initials={currentUser.initials}
            className="size-8"
            data-slot="avatar"
          />
          <div className="flex-1 min-w-0">
            <div className="text-sm font-medium text-zinc-900 dark:text-white truncate">
              {currentUser.name}
            </div>
            <div className="text-xs text-zinc-500 dark:text-zinc-400 truncate">
              {currentUser.email}
            </div>
          </div>
          <Dropdown>
            <DropdownButton plain>
              <MaterialSymbol name="more_vert" size="sm" />
            </DropdownButton>
            <DropdownMenu>
              <DropdownItem onClick={() => handleUserMenuAction('profile')}>
                <MaterialSymbol name="person" />
                Profile
              </DropdownItem>
              <DropdownItem onClick={() => handleUserMenuAction('settings')}>
                <MaterialSymbol name="settings" />
                Settings
              </DropdownItem>
              <DropdownItem onClick={() => handleUserMenuAction('signout')}>
                <MaterialSymbol name="logout" />
                Sign out
              </DropdownItem>
            </DropdownMenu>
          </Dropdown>
        </div>
      </SidebarFooter>
    </Sidebar>
  )

  // Create navbar content
  const navbarContent = (
    <Navbar>
      <NavbarSection>
        <NavbarItem>
          <MaterialSymbol name="search" data-slot="icon" />
          <NavbarLabel>Search</NavbarLabel>
        </NavbarItem>
      </NavbarSection>
      
      <NavbarSpacer />
      
      <NavbarSection>
        <NavbarItem onClick={onConfigurationClick}>
          <MaterialSymbol name="settings" data-slot="icon" />
          <NavbarLabel>Settings</NavbarLabel>
        </NavbarItem>
        
        <NavbarItem>
          <MaterialSymbol name="notifications" data-slot="icon" />
          <NavbarLabel>Notifications</NavbarLabel>
        </NavbarItem>
      </NavbarSection>
    </Navbar>
  )

  return (
    <SidebarLayout navbar={navbarContent} sidebar={sidebarContent}>
      {/* Main Page Content */}
      <div className="space-y-8">
        {/* Welcome Section */}
        <div className="text-center py-12">
          <Subheading level={1} className="mb-4">Welcome to SuperPlane</Subheading>
          <Text className="text-lg text-zinc-600 dark:text-zinc-400 mb-8">
            Your workflow automation platform
          </Text>
          
          <Button color="blue" className="mb-8">
            <MaterialSymbol name="add" />
            Create New Canvas
          </Button>
        </div>

        {/* Quick Actions */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          <div className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700 hover:shadow-md transition-shadow">
            <div className="flex items-center gap-3 mb-4">
              <div className="p-2 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
                <MaterialSymbol name="dashboard" className="text-blue-600 dark:text-blue-400" size="lg" />
              </div>
              <Subheading level={3}>Canvases</Subheading>
            </div>
            <Text className="text-zinc-600 dark:text-zinc-400 mb-4">
              Create and manage your workflow canvases
            </Text>
            <Button plain>
              View all canvases
              <MaterialSymbol name="arrow_forward" />
            </Button>
          </div>
          
          <div className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700 hover:shadow-md transition-shadow">
            <div className="flex items-center gap-3 mb-4">
              <div className="p-2 bg-green-50 dark:bg-green-900/20 rounded-lg">
                <MaterialSymbol name="automation" className="text-green-600 dark:text-green-400" size="lg" />
              </div>
              <Subheading level={3}>Automations</Subheading>
            </div>
            <Text className="text-zinc-600 dark:text-zinc-400 mb-4">
              Set up automated processes and workflows
            </Text>
            <Button plain>
              Manage automations
              <MaterialSymbol name="arrow_forward" />
            </Button>
          </div>
          
          <div className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700 hover:shadow-md transition-shadow">
            <div className="flex items-center gap-3 mb-4">
              <div className="p-2 bg-purple-50 dark:bg-purple-900/20 rounded-lg">
                <MaterialSymbol name="analytics" className="text-purple-600 dark:text-purple-400" size="lg" />
              </div>
              <Subheading level={3}>Analytics</Subheading>
            </div>
            <Text className="text-zinc-600 dark:text-zinc-400 mb-4">
              Monitor and analyze your workflow performance
            </Text>
            <Button plain>
              View analytics
              <MaterialSymbol name="arrow_forward" />
            </Button>
          </div>
        </div>

        {/* Recent Activity */}
        <div className="space-y-4">
          <Subheading level={2}>Recent Activity</Subheading>
          <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700">
            <div className="p-6 text-center text-zinc-500 dark:text-zinc-400">
              <MaterialSymbol name="history" size="xl" className="mx-auto mb-2 opacity-50" />
              <Text>No recent activity to show</Text>
            </div>
          </div>
        </div>
      </div>
    </SidebarLayout>
  )
}