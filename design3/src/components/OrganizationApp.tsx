import { useState } from 'react'
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
  SidebarSection, 
  SidebarItem, 
  SidebarLabel
} from './lib/Sidebar/sidebar'
import { 
  Dropdown, 
  DropdownButton, 
  DropdownDivider,
  DropdownMenu, 
  DropdownItem,
  DropdownLabel 
} from './lib/Dropdown/dropdown'
import { MaterialSymbol } from './lib/MaterialSymbol/material-symbol'
import { Avatar } from './lib/Avatar/avatar'
import { MembersPage } from './MembersPage'
import { WorkflowPage } from './WorkflowPage'
import { WorkflowEditor } from './WorkflowEditor'
import { OrganizationSettings } from './OrganizationSettings'
import { Subheading } from './lib/Heading/heading'
import { Text } from './lib/Text/text'
import { Button } from './lib/Button/button'

interface OrganizationAppProps {
  onSignOut?: () => void
  onSwitchOrganization?: () => void
}

export function OrganizationApp({ 
  onSignOut, 
  onSwitchOrganization
}: OrganizationAppProps) {
  const [currentPage, setCurrentPage] = useState('dashboard')
  const [currentWorkflow, setCurrentWorkflow] = useState<{ id: string; name: string } | null>(null)

  // Mock data
  const currentUser = {
    id: '1',
    name: 'John Doe',
    email: 'john@acme.com',
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
      successRate: 98,
      executions: 342
    },
    {
      id: '2',
      name: 'Invoice Processing',
      description: 'Process and validate incoming invoices automatically',
      status: 'active',
      lastRun: '1 day ago',
      successRate: 95,
      executions: 156
    },
    {
      id: '3',
      name: 'Employee Offboarding',
      description: 'Handle employee departures and access revocation',
      status: 'draft',
      lastRun: 'Never',
      successRate: 0,
      executions: 0
    }
  ]

  // Navigation items
  const navItems = [
    { label: 'Dashboard', id: 'dashboard', icon: 'dashboard' },
    { label: 'Workflows', id: 'workflows', icon: 'account_tree' }
  ]

  const handleNavigation = (page: string) => {
    setCurrentPage(page)
    setCurrentWorkflow(null) // Clear workflow when navigating to other pages
  }

  const handleOpenWorkflow = (workflowId: string, workflowName: string) => {
    setCurrentWorkflow({ id: workflowId, name: workflowName })
    setCurrentPage('workflow-editor')
  }

  const handleBackToWorkflows = () => {
    setCurrentWorkflow(null)
    setCurrentPage('workflows')
  }

  const handleUserAction = (action: 'profile' | 'account-settings' | 'signout') => {
    if (action === 'signout') {
      onSignOut?.()
    } else {
      console.log(`User action: ${action}`)
    }
  }

  const handleOrgAction = (action: 'settings' | 'billing' | 'members' | 'switch') => {
    if (action === 'switch') {
      onSwitchOrganization?.()
    } else if (action === 'members') {
      setCurrentPage('members')
    } else if (action === 'settings') {
      setCurrentPage('organization-settings')
      setCurrentWorkflow(null)
    } else {
      setCurrentPage(action)
    }
  }

  // Organization dropdown menu component
  function OrganizationDropdownMenu() {
    return (
      <DropdownMenu className="min-w-64" anchor="bottom start">
        <DropdownItem onClick={() => handleOrgAction('settings')}>
          <MaterialSymbol name="settings" />
          <DropdownLabel>Organization Settings</DropdownLabel>
        </DropdownItem>
        <DropdownDivider />
        <DropdownItem onClick={() => handleOrgAction('members')}>
          <Avatar 
            initials={currentOrganization.initials} 
            className="bg-blue-500 text-white" 
            data-slot="icon" 
          />
          <DropdownLabel>{currentOrganization.name}</DropdownLabel>
        </DropdownItem>
        <DropdownItem>
          <Avatar initials="TC" className="bg-purple-500 text-white" data-slot="icon" />
          <DropdownLabel>Tech Corp</DropdownLabel>
        </DropdownItem>
        <DropdownDivider />
        <DropdownItem onClick={() => handleOrgAction('switch')}>
          <MaterialSymbol name="add" />
          <DropdownLabel>New organization&hellip;</DropdownLabel>
        </DropdownItem>
      </DropdownMenu>
    )
  }

  // User dropdown menu component  
  function UserDropdownMenu() {
    return (
      <DropdownMenu className="min-w-64" anchor="bottom end">
        <DropdownItem onClick={() => handleUserAction('profile')}>
          <MaterialSymbol name="person" />
          <DropdownLabel>My profile</DropdownLabel>
        </DropdownItem>
        <DropdownItem onClick={() => handleUserAction('account-settings')}>
          <MaterialSymbol name="settings" />
          <DropdownLabel>Account Settings</DropdownLabel>
        </DropdownItem>
        <DropdownDivider />
        <DropdownItem>
          <MaterialSymbol name="shield" />
          <DropdownLabel>Privacy policy</DropdownLabel>
        </DropdownItem>
        <DropdownItem>
          <MaterialSymbol name="lightbulb" />
          <DropdownLabel>Share feedback</DropdownLabel>
        </DropdownItem>
        <DropdownDivider />
        <DropdownItem onClick={() => handleUserAction('signout')}>
          <MaterialSymbol name="logout" />
          <DropdownLabel>Sign out</DropdownLabel>
        </DropdownItem>
      </DropdownMenu>
    )
  }

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
                      {workflows.filter(w => w.status === 'active').length}
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
                    <MaterialSymbol name="play_arrow" className="text-green-600 dark:text-green-400" size="lg" />
                  </div>
                  <div>
                    <div className="text-2xl font-semibold text-zinc-900 dark:text-white">
                      {workflows.reduce((sum, w) => sum + w.executions, 0)}
                    </div>
                    <div className="text-sm text-zinc-600 dark:text-zinc-400">
                      Total Executions
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
                      Avg Success Rate
                    </div>
                  </div>
                </div>
              </div>

              <div className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700">
                <div className="flex items-center gap-3">
                  <div className="p-2 bg-orange-50 dark:bg-orange-900/20 rounded-lg">
                    <MaterialSymbol name="group" className="text-orange-600 dark:text-orange-400" size="lg" />
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
            </div>

            {/* Recent Workflows */}
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <Subheading level={2}>Recent Workflows</Subheading>
                <Button color="blue" onClick={() => setCurrentPage('workflows')}>
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
                              Last run: {workflow.lastRun} • Success rate: {workflow.successRate}% • {workflow.executions} executions
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
                          <Dropdown>
                            <DropdownButton plain>
                              <MaterialSymbol name="more_vert" />
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
                                <MaterialSymbol name="pause" />
                                Pause
                              </DropdownItem>
                            </DropdownMenu>
                          </Dropdown>
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
        return <WorkflowPage onOpenWorkflow={handleOpenWorkflow} />

      case 'workflow-editor':
        if (currentWorkflow) {
          return (
            <WorkflowEditor
              workflowId={currentWorkflow.id}
              workflowName={currentWorkflow.name}
              onBack={handleBackToWorkflows}
              onSignOut={onSignOut}
              onSwitchOrganization={onSwitchOrganization}
            />
          )
        }
        return null

      case 'members':
        return <MembersPage />

      case 'organization-settings':
        return (
          <OrganizationSettings
            onBack={() => setCurrentPage('dashboard')}
            onSignOut={onSignOut}
            onSwitchOrganization={onSwitchOrganization}
          />
        )

      case 'settings':
        return (
          <div className="space-y-6">
            <div>
              <Subheading level={1} className="mb-2">Organization Settings</Subheading>
              <Text className="text-zinc-600 dark:text-zinc-400">
                Configure your organization preferences and settings
              </Text>
            </div>

            <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 p-6">
              <Text className="text-center text-zinc-500 dark:text-zinc-400">
                Organization settings would go here...
              </Text>
            </div>
          </div>
        )

      case 'billing':
        return (
          <div className="space-y-6">
            <div>
              <Subheading level={1} className="mb-2">Billing & Plans</Subheading>
              <Text className="text-zinc-600 dark:text-zinc-400">
                Manage your subscription and billing information
              </Text>
            </div>

            <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 p-6">
              <Text className="text-center text-zinc-500 dark:text-zinc-400">
                Billing interface would go here...
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

  // Special full-width layout for workflow editor and organization settings
  if (currentPage === 'workflow-editor' || currentPage === 'organization-settings') {
    return renderPageContent()
  }

  return (
    <div className="flex flex-col bg-zinc-50 dark:bg-zinc-900">
      {/* Top Navbar */}
      <Navbar>
        <Dropdown>
          <DropdownButton as={NavbarItem}>
            <Avatar 
              initials={currentOrganization.initials} 
              className="bg-blue-500 text-white" 
            />
            <NavbarLabel>{currentOrganization.name}</NavbarLabel>
            <MaterialSymbol name="expand_more" />
          </DropdownButton>
          <OrganizationDropdownMenu />
        </Dropdown>
        
        <NavbarSpacer />
        
        <NavbarSection>
          <NavbarItem aria-label="Search">
            <MaterialSymbol name="search" />
          </NavbarItem>
          <NavbarItem aria-label="Notifications">
            <MaterialSymbol name="notifications" />
          </NavbarItem>
          <Dropdown>
            <DropdownButton as={NavbarItem}>
              <Avatar 
                src={currentUser.avatar} 
                initials={currentUser.initials}
                square 
              />
            </DropdownButton>
            <UserDropdownMenu />
          </Dropdown>
        </NavbarSection>
      </Navbar>
      
      {/* Main content area with sidebar */}
      <div className="flex flex-1 items-start h-full overflow-hidden border-t border-zinc-200 dark:border-zinc-700">
        {/* Sidebar Navigation */}
        <div className="w-64 bg-white dark:bg-zinc-800 h-screen border-r border-zinc-200 dark:border-zinc-700">
          <Sidebar>
            
            <SidebarBody>
              <SidebarSection>
                {navItems.map(({ label, id, icon }) => (
                  <SidebarItem 
                    key={label}
                    current={currentPage === id}
                    onClick={() => handleNavigation(id)}
                  >
                    <MaterialSymbol name={icon} data-slot="icon" />
                    <SidebarLabel>{label}</SidebarLabel>
                  </SidebarItem>
                ))}
              </SidebarSection>
            </SidebarBody>
          </Sidebar>
        </div>

        {/* Page Content */}
        <div className="flex-1 overflow-auto">
          <div className="p-8">
            {renderPageContent()}
          </div>
        </div>
      </div>
    </div>
  )
}