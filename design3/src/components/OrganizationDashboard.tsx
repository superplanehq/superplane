import { useState } from 'react'
import { StackedLayout } from './lib/StackedLayout/stacked-layout'
import { 
  Navbar, 
  NavbarSection, 
  NavbarSpacer, 
  NavbarItem,
  NavbarLabel 
} from './lib/Navbar/navbar'
import { 
  Sidebar, 
  SidebarHeader, 
  SidebarBody, 
  SidebarSection as SidebarSec, 
  SidebarItem, 
  SidebarLabel
} from './lib/Sidebar/sidebar'
import { Subheading } from './lib/Heading/heading'
import { Text } from './lib/Text/text'
import { Button } from './lib/Button/button'
import { MaterialSymbol } from './lib/MaterialSymbol/material-symbol'
import { Avatar } from './lib/Avatar/avatar'
import { Dropdown, DropdownButton, DropdownMenu, DropdownItem } from './lib/Dropdown/dropdown'

interface OrganizationDashboardProps {
  onSignOut?: () => void
  onPageChange?: (page: string) => void
}

export function OrganizationDashboard({ 
  onSignOut, 
  onPageChange
}: OrganizationDashboardProps) {
  const [currentPage, setCurrentPage] = useState('dashboard')

  // Mock data
  const currentUser = {
    id: '1',
    name: 'John Doe',
    email: 'john@superplane.com',
    initials: 'JD',
    avatar: 'https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?w=64&h=64&fit=crop&crop=face',
  }

  const currentOrganization = {
    id: '1',
    name: 'Acme Corporation',
    plan: 'Pro Plan',
    initials: 'AC',
  }

  const workflows = [
    {
      id: '1',
      name: 'Customer Onboarding',
      description: 'Automated workflow for new customer registration and setup',
      status: 'active',
      lastRun: '2 hours ago',
      successRate: 98
    },
    {
      id: '2',
      name: 'Invoice Processing',
      description: 'Process and validate incoming invoices automatically',
      status: 'active',
      lastRun: '1 day ago',
      successRate: 95
    },
    {
      id: '3',
      name: 'Employee Offboarding',
      description: 'Handle employee departures and access revocation',
      status: 'draft',
      lastRun: 'Never',
      successRate: 0
    }
  ]

  const handleNavigation = (page: string) => {
    setCurrentPage(page)
    onPageChange?.(page)
  }

  const handleUserAction = (action: 'profile' | 'settings' | 'signout') => {
    if (action === 'signout') {
      onSignOut?.()
    } else {
      handleNavigation(action)
    }
  }

  // Navbar content with horizontal navigation
  const navbarContent = (
    <Navbar>
      <NavbarSection>
        {/* Logo/Brand */}
        <NavbarItem>
          <div className="flex items-center gap-2">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-blue-500 text-white text-sm font-semibold">
              SP
            </div>
            <span className="font-bold text-lg">SuperPlane</span>
          </div>
        </NavbarItem>
      </NavbarSection>

      <NavbarSpacer />

      {/* Main Navigation */}
      <NavbarSection>
        <NavbarItem 
          current={currentPage === 'dashboard'}
          onClick={() => handleNavigation('dashboard')}
        >
          <MaterialSymbol name="dashboard" data-slot="icon" />
          <NavbarLabel>Dashboard</NavbarLabel>
        </NavbarItem>
        
        <NavbarItem 
          current={currentPage === 'workflows'}
          onClick={() => handleNavigation('workflows')}
        >
          <MaterialSymbol name="account_tree" data-slot="icon" />
          <NavbarLabel>Workflows</NavbarLabel>
        </NavbarItem>

        <NavbarItem 
          current={currentPage === 'members'}
          onClick={() => handleNavigation('members')}
        >
          <MaterialSymbol name="group" data-slot="icon" />
          <NavbarLabel>Members</NavbarLabel>
        </NavbarItem>

        <NavbarItem 
          current={currentPage === 'settings'}
          onClick={() => handleNavigation('settings')}
        >
          <MaterialSymbol name="settings" data-slot="icon" />
          <NavbarLabel>Settings</NavbarLabel>
        </NavbarItem>
      </NavbarSection>

      <NavbarSpacer />

      {/* Right Section */}
      <NavbarSection>
        <NavbarItem>
          <MaterialSymbol name="search" data-slot="icon" />
        </NavbarItem>
        
        <NavbarItem>
          <MaterialSymbol name="notifications" data-slot="icon" />
        </NavbarItem>

        {/* Organization Dropdown */}
        <Dropdown>
          <DropdownButton plain>
            <div className="flex items-center gap-2">
              <div className="flex h-6 w-6 items-center justify-center rounded bg-blue-500 text-white text-xs font-semibold">
                {currentOrganization.initials}
              </div>
              <span className="hidden sm:block text-sm">{currentOrganization.name}</span>
              <MaterialSymbol name="expand_more" size="sm" />
            </div>
          </DropdownButton>
          <DropdownMenu>
            <DropdownItem>
              <MaterialSymbol name="business" />
              Organization Settings
            </DropdownItem>
            <DropdownItem>
              <MaterialSymbol name="group" />
              Manage Members
            </DropdownItem>
            <DropdownItem>
              <MaterialSymbol name="credit_card" />
              Billing & Plans
            </DropdownItem>
            <DropdownItem>
              <MaterialSymbol name="add" />
              Switch Organization
            </DropdownItem>
          </DropdownMenu>
        </Dropdown>

        {/* User Dropdown */}
        <Dropdown>
          <DropdownButton plain>
            <Avatar
              src={currentUser.avatar}
              initials={currentUser.initials}
              className="size-8"
              data-slot="avatar"
            />
          </DropdownButton>
          <DropdownMenu>
            <DropdownItem onClick={() => handleUserAction('profile')}>
              <MaterialSymbol name="person" />
              Profile
            </DropdownItem>
            <DropdownItem onClick={() => handleUserAction('settings')}>
              <MaterialSymbol name="settings" />
              Account Settings
            </DropdownItem>
            <DropdownItem onClick={() => handleUserAction('signout')}>
              <MaterialSymbol name="logout" />
              Sign out
            </DropdownItem>
          </DropdownMenu>
        </Dropdown>
      </NavbarSection>
    </Navbar>
  )

  // Sidebar content (secondary navigation or filters)
  const sidebarContent = (
    <Sidebar>
      <SidebarHeader>
        <div className="p-4">
          <Subheading level={3} className="mb-2">Quick Actions</Subheading>
        </div>
      </SidebarHeader>

      <SidebarBody>
        <SidebarSec>
          <SidebarItem>
            <MaterialSymbol name="add" data-slot="icon" />
            <SidebarLabel>New Workflow</SidebarLabel>
          </SidebarItem>
          
          <SidebarItem>
            <MaterialSymbol name="person_add" data-slot="icon" />
            <SidebarLabel>Invite Member</SidebarLabel>
          </SidebarItem>
          
          <SidebarItem>
            <MaterialSymbol name="analytics" data-slot="icon" />
            <SidebarLabel>View Analytics</SidebarLabel>
          </SidebarItem>
        </SidebarSec>
      </SidebarBody>
    </Sidebar>
  )

  const renderPageContent = () => {
    switch (currentPage) {
      case 'dashboard':
        return (
          <div className="space-y-8">
            {/* Welcome Section */}
            <div>
              <Subheading level={1} className="mb-2">Welcome to {currentOrganization.name}</Subheading>
              <Text className="text-zinc-600 dark:text-zinc-400">
                Manage your workflows, team members, and organization settings
              </Text>
            </div>

            {/* Quick Stats */}
            <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
              <div className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700">
                <div className="flex items-center gap-3">
                  <div className="p-2 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
                    <MaterialSymbol name="account_tree" className="text-blue-600 dark:text-blue-400" size="lg" />
                  </div>
                  <div>
                    <div className="text-2xl font-semibold text-zinc-900 dark:text-white">
                      {workflows.length}
                    </div>
                    <div className="text-sm text-zinc-600 dark:text-zinc-400">
                      Active Workflows
                    </div>
                  </div>
                </div>
              </div>

              <div className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700">
                <div className="flex items-center gap-3">
                  <div className="p-2 bg-green-50 dark:bg-green-900/20 rounded-lg">
                    <MaterialSymbol name="group" className="text-green-600 dark:text-green-400" size="lg" />
                  </div>
                  <div>
                    <div className="text-2xl font-semibold text-zinc-900 dark:text-white">
                      12
                    </div>
                    <div className="text-sm text-zinc-600 dark:text-zinc-400">
                      Team Members
                    </div>
                  </div>
                </div>
              </div>

              <div className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700">
                <div className="flex items-center gap-3">
                  <div className="p-2 bg-purple-50 dark:bg-purple-900/20 rounded-lg">
                    <MaterialSymbol name="trending_up" className="text-purple-600 dark:text-purple-400" size="lg" />
                  </div>
                  <div>
                    <div className="text-2xl font-semibold text-zinc-900 dark:text-white">
                      97%
                    </div>
                    <div className="text-sm text-zinc-600 dark:text-zinc-400">
                      Success Rate
                    </div>
                  </div>
                </div>
              </div>

              <div className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700">
                <div className="flex items-center gap-3">
                  <div className="p-2 bg-orange-50 dark:bg-orange-900/20 rounded-lg">
                    <MaterialSymbol name="schedule" className="text-orange-600 dark:text-orange-400" size="lg" />
                  </div>
                  <div>
                    <div className="text-2xl font-semibold text-zinc-900 dark:text-white">
                      24h
                    </div>
                    <div className="text-sm text-zinc-600 dark:text-zinc-400">
                      Avg Runtime
                    </div>
                  </div>
                </div>
              </div>
            </div>

            {/* Recent Workflows */}
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <Subheading level={2}>Recent Workflows</Subheading>
                <Button color="blue">
                  <MaterialSymbol name="add" />
                  New Workflow
                </Button>
              </div>

              <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700">
                <div className="p-6">
                  <div className="space-y-4">
                    {workflows.map((workflow) => (
                      <div key={workflow.id} className="flex items-center justify-between p-4 border border-zinc-200 dark:border-zinc-700 rounded-lg">
                        <div className="flex items-center gap-4">
                          <div className={`p-2 rounded-lg ${
                            workflow.status === 'active' 
                              ? 'bg-green-50 dark:bg-green-900/20'
                              : 'bg-zinc-50 dark:bg-zinc-900/20'
                          }`}>
                            <MaterialSymbol 
                              name="account_tree" 
                              className={
                                workflow.status === 'active'
                                  ? 'text-green-600 dark:text-green-400'
                                  : 'text-zinc-600 dark:text-zinc-400'
                              }
                            />
                          </div>
                          <div>
                            <div className="font-medium text-zinc-900 dark:text-white">
                              {workflow.name}
                            </div>
                            <div className="text-sm text-zinc-600 dark:text-zinc-400">
                              {workflow.description}
                            </div>
                            <div className="text-xs text-zinc-500 dark:text-zinc-400 mt-1">
                              Last run: {workflow.lastRun} • Success rate: {workflow.successRate}%
                            </div>
                          </div>
                        </div>
                        <div className="flex items-center gap-2">
                          <span className={`px-2 py-1 rounded-full text-xs font-medium ${
                            workflow.status === 'active'
                              ? 'bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-400'
                              : 'bg-zinc-100 text-zinc-800 dark:bg-zinc-900/20 dark:text-zinc-400'
                          }`}>
                            {workflow.status}
                          </span>
                          <Button plain>
                            <MaterialSymbol name="more_vert" />
                          </Button>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            </div>
          </div>
        )
      
      case 'workflows':
        return (
          <div className="space-y-6">
            <div className="flex items-center justify-between">
              <div>
                <Subheading level={1} className="mb-2">Workflows</Subheading>
                <Text className="text-zinc-600 dark:text-zinc-400">
                  Create and manage automated workflows for your organization
                </Text>
              </div>
              <Button color="blue">
                <MaterialSymbol name="add" />
                Create Workflow
              </Button>
            </div>

            {/* Workflow Grid */}
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
              {workflows.map((workflow) => (
                <div key={workflow.id} className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700">
                  <div className="flex items-start justify-between mb-4">
                    <div className={`p-2 rounded-lg ${
                      workflow.status === 'active' 
                        ? 'bg-green-50 dark:bg-green-900/20'
                        : 'bg-zinc-50 dark:bg-zinc-900/20'
                    }`}>
                      <MaterialSymbol 
                        name="account_tree" 
                        className={
                          workflow.status === 'active'
                            ? 'text-green-600 dark:text-green-400'
                            : 'text-zinc-600 dark:text-zinc-400'
                        }
                        size="lg" 
                      />
                    </div>
                    <Dropdown>
                      <DropdownButton plain>
                        <MaterialSymbol name="more_vert" size="sm" />
                      </DropdownButton>
                      <DropdownMenu>
                        <DropdownItem>
                          <MaterialSymbol name="edit" />
                          Edit
                        </DropdownItem>
                        <DropdownItem>
                          <MaterialSymbol name="copy" />
                          Duplicate
                        </DropdownItem>
                        <DropdownItem>
                          <MaterialSymbol name="delete" />
                          Delete
                        </DropdownItem>
                      </DropdownMenu>
                    </Dropdown>
                  </div>

                  <Subheading level={3} className="mb-2">{workflow.name}</Subheading>
                  <Text className="text-zinc-600 dark:text-zinc-400 mb-4 line-clamp-2">
                    {workflow.description}
                  </Text>

                  <div className="flex items-center justify-between">
                    <span className={`px-2 py-1 rounded-full text-xs font-medium ${
                      workflow.status === 'active'
                        ? 'bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-400'
                        : 'bg-zinc-100 text-zinc-800 dark:bg-zinc-900/20 dark:text-zinc-400'
                    }`}>
                      {workflow.status}
                    </span>
                    <Text className="text-xs text-zinc-500 dark:text-zinc-400">
                      {workflow.successRate}% success
                    </Text>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )

      case 'members':
        return (
          <div className="space-y-6">
            <div className="flex items-center justify-between">
              <div>
                <Subheading level={1} className="mb-2">Members</Subheading>
                <Text className="text-zinc-600 dark:text-zinc-400">
                  Manage team members, roles, and permissions
                </Text>
              </div>
              <Button color="blue">
                <MaterialSymbol name="person_add" />
                Invite Member
              </Button>
            </div>

            <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 p-6">
              <Text className="text-center text-zinc-500 dark:text-zinc-400">
                Member management interface would go here...
              </Text>
            </div>
          </div>
        )

      case 'settings':
        return (
          <div className="space-y-6">
            <div>
              <Subheading level={1} className="mb-2">Organization Settings</Subheading>
              <Text className="text-zinc-600 dark:text-zinc-400">
                Configure your organization preferences, security, and integrations
              </Text>
            </div>

            <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 p-6">
              <Text className="text-center text-zinc-500 dark:text-zinc-400">
                Settings interface would go here...
              </Text>
            </div>
          </div>
        )

      default:
        return (
          <div className="text-center py-12">
            <Subheading level={2} className="mb-4">Page Not Found</Subheading>
            <Text className="text-zinc-600 dark:text-zinc-400">
              The requested page could not be found.
            </Text>
          </div>
        )
    }
  }

  return (
    <StackedLayout navbar={navbarContent} sidebar={sidebarContent}>
      {renderPageContent()}
    </StackedLayout>
  )
}