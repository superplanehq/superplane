import { useState } from 'react'
import { Button } from './lib/Button/button'
import { Badge } from './lib/Badge/badge'
import { Avatar } from './lib/Avatar/avatar'
import { Text, TextLink } from './lib/Text/text'
import { Heading, Subheading } from './lib/Heading/heading'
import { Navigation, type User, type Organization } from './lib/Navigation/navigation'

interface WorkspacesPageProps {
  onSignOut?: () => void
  onWorkspaceSelect?: (workspaceId: string) => void
}

interface Workspace {
  id: string
  name: string
  description: string
  memberCount: number
  flowCount: number
  lastActivity: string
  role: 'owner' | 'admin' | 'member'
  status: 'active' | 'archived'
  avatar?: string
  initials: string
  color: 'blue' | 'green' | 'purple' | 'orange' | 'red' | 'yellow'
  plan: 'free' | 'pro' | 'enterprise'
}

export function WorkspacesPage({ onSignOut, onWorkspaceSelect }: WorkspacesPageProps) {
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false)

  // Mock user data
  const currentUser: User = {
    id: '1',
    name: 'John Doe',
    email: 'john@superplane.com',
    initials: 'JD',
    avatar: 'https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?w=64&h=64&fit=crop&crop=face',
  }

  // Mock organization data for navigation (placeholder)
  const currentOrganization: Organization = {
    id: 'personal',
    name: 'Personal',
    initials: 'P',
  }

  // Mock workspaces data
  const workspaces: Workspace[] = [
    {
      id: '1',
      name: 'Development Team',
      description: 'Main development workspace for core products and services',
      memberCount: 12,
      flowCount: 24,
      lastActivity: '5 minutes ago',
      role: 'owner',
      status: 'active',
      initials: 'DT',
      color: 'blue',
      plan: 'pro'
    },
    {
      id: '2',
      name: 'Design System',
      description: 'Collaborative space for design system development and maintenance',
      memberCount: 6,
      flowCount: 8,
      lastActivity: '2 hours ago',
      role: 'admin',
      status: 'active',
      initials: 'DS',
      color: 'purple',
      plan: 'pro'
    },
    {
      id: '3',
      name: 'Marketing Automation',
      description: 'Marketing workflows and campaign automation processes',
      memberCount: 4,
      flowCount: 15,
      lastActivity: '1 day ago',
      role: 'member',
      status: 'active',
      initials: 'MA',
      color: 'green',
      plan: 'enterprise'
    },
    {
      id: '4',
      name: 'Security & Compliance',
      description: 'Security audits, compliance checks, and vulnerability management',
      memberCount: 3,
      flowCount: 12,
      lastActivity: '3 days ago',
      role: 'admin',
      status: 'active',
      initials: 'SC',
      color: 'red',
      plan: 'enterprise'
    },
    {
      id: '5',
      name: 'Data Analytics',
      description: 'Data processing, analytics workflows, and reporting automation',
      memberCount: 8,
      flowCount: 18,
      lastActivity: '1 week ago',
      role: 'member',
      status: 'active',
      initials: 'DA',
      color: 'orange',
      plan: 'pro'
    },
    {
      id: '6',
      name: 'Legacy Projects',
      description: 'Archived workspace for legacy project workflows and documentation',
      memberCount: 2,
      flowCount: 5,
      lastActivity: '2 months ago',
      role: 'owner',
      status: 'archived',
      initials: 'LP',
      color: 'yellow',
      plan: 'free'
    }
  ]

  // Navigation handlers
  const handleHelpClick = () => {
    console.log('Opening help documentation...')
  }

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

  const handleOrganizationMenuAction = (action: 'settings' | 'billing' | 'members') => {
    console.log(`Organization action: ${action}`)
  }

  const getRoleBadge = (role: Workspace['role']) => {
    switch (role) {
      case 'owner':
        return <Badge color="purple">Owner</Badge>
      case 'admin':
        return <Badge color="blue">Admin</Badge>
      case 'member':
        return <Badge color="zinc">Member</Badge>
    }
  }

  const getPlanBadge = (plan: Workspace['plan']) => {
    switch (plan) {
      case 'free':
        return <Badge color="zinc">Free</Badge>
      case 'pro':
        return <Badge color="blue">Pro</Badge>
      case 'enterprise':
        return <Badge color="purple">Enterprise</Badge>
    }
  }

  const getStatusBadge = (status: Workspace['status']) => {
    switch (status) {
      case 'active':
        return <Badge color="green">Active</Badge>
      case 'archived':
        return <Badge color="yellow">Archived</Badge>
    }
  }

  const activeWorkspaces = workspaces.filter(w => w.status === 'active')
  const archivedWorkspaces = workspaces.filter(w => w.status === 'archived')

  return (
    <div className="min-h-screen bg-zinc-50 dark:bg-zinc-900">
      {/* Navigation */}
      <Navigation
        user={currentUser}
        organization={currentOrganization}
        onHelpClick={handleHelpClick}
        onUserMenuAction={handleUserMenuAction}
        onOrganizationMenuAction={handleOrganizationMenuAction}
      />

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Header */}
        <div className="flex items-center justify-between mb-8">
          <div>
            <Heading level={1} className="text-3xl mb-2">
              Your Workspaces
            </Heading>
            <Text className="text-lg text-zinc-600 dark:text-zinc-400">
              Collaborate with your team and manage workflows across different projects
            </Text>
          </div>
          <Button color="blue" onClick={() => setIsCreateModalOpen(true)}>
            <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
            Create Workspace
          </Button>
        </div>

        {/* Quick Stats */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
          <div className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700">
            <div className="flex items-center">
              <div className="flex-shrink-0">
                <svg className="w-8 h-8 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-4m-5 0H3m2 0h3M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4" />
                </svg>
              </div>
              <div className="ml-4">
                <Subheading level={3} className="text-2xl">{activeWorkspaces.length}</Subheading>
                <Text className="text-sm">Active Workspaces</Text>
              </div>
            </div>
          </div>

          <div className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700">
            <div className="flex items-center">
              <div className="flex-shrink-0">
                <svg className="w-8 h-8 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0" />
                </svg>
              </div>
              <div className="ml-4">
                <Subheading level={3} className="text-2xl">
                  {activeWorkspaces.reduce((acc, w) => acc + w.memberCount, 0)}
                </Subheading>
                <Text className="text-sm">Total Members</Text>
              </div>
            </div>
          </div>

          <div className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700">
            <div className="flex items-center">
              <div className="flex-shrink-0">
                <svg className="w-8 h-8 text-purple-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
                </svg>
              </div>
              <div className="ml-4">
                <Subheading level={3} className="text-2xl">
                  {activeWorkspaces.reduce((acc, w) => acc + w.flowCount, 0)}
                </Subheading>
                <Text className="text-sm">Total Workflows</Text>
              </div>
            </div>
          </div>

          <div className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700">
            <div className="flex items-center">
              <div className="flex-shrink-0">
                <svg className="w-8 h-8 text-orange-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              </div>
              <div className="ml-4">
                <Subheading level={3} className="text-2xl">
                  {workspaces.filter(w => w.role === 'owner').length}
                </Subheading>
                <Text className="text-sm">Owned by You</Text>
              </div>
            </div>
          </div>
        </div>

        {/* Active Workspaces */}
        <div className="mb-12">
          <div className="flex items-center justify-between mb-6">
            <Subheading level={2}>Active Workspaces</Subheading>
            <TextLink href="#" className="text-sm">View all →</TextLink>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {activeWorkspaces.map((workspace) => (
              <div 
                key={workspace.id}
                className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700 hover:shadow-lg transition-all cursor-pointer group"
                onClick={() => onWorkspaceSelect?.(workspace.id)}
              >
                <div className="flex items-start justify-between mb-4">
                  <Avatar
                    src={workspace.avatar}
                    initials={workspace.initials}
                    alt={workspace.name}
                    className={`w-12 h-12 bg-${workspace.color}-100 text-${workspace.color}-700 dark:bg-${workspace.color}-900 dark:text-${workspace.color}-300`}
                  />
                  <div className="flex items-center space-x-2">
                    {getRoleBadge(workspace.role)}
                    {getPlanBadge(workspace.plan)}
                  </div>
                </div>

                <div className="mb-4">
                  <Subheading level={3} className="text-lg mb-2 group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors">
                    {workspace.name}
                  </Subheading>
                  <Text className="text-sm text-zinc-600 dark:text-zinc-400 line-clamp-2">
                    {workspace.description}
                  </Text>
                </div>

                <div className="flex items-center justify-between text-xs text-zinc-500 mb-4">
                  <div className="flex items-center space-x-4">
                    <span>{workspace.memberCount} members</span>
                    <span>{workspace.flowCount} flows</span>
                  </div>
                  <span>{workspace.lastActivity}</span>
                </div>

                <div className="flex items-center justify-between">
                  {getStatusBadge(workspace.status)}
                  <Button 
                    plain 
                    className="opacity-0 group-hover:opacity-100 transition-opacity text-blue-600 hover:text-blue-700"
                    onClick={(e: React.MouseEvent) => {
                      e.stopPropagation()
                      console.log('Workspace settings:', workspace.id)
                    }}
                  >
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                    </svg>
                  </Button>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Archived Workspaces */}
        {archivedWorkspaces.length > 0 && (
          <div className="mb-8">
            <div className="flex items-center justify-between mb-6">
              <Subheading level={2}>Archived Workspaces</Subheading>
              <Text className="text-sm text-zinc-500">{archivedWorkspaces.length} archived</Text>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
              {archivedWorkspaces.map((workspace) => (
                <div 
                  key={workspace.id}
                  className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700 opacity-75 hover:opacity-100 transition-opacity"
                >
                  <div className="flex items-start justify-between mb-4">
                    <Avatar
                      src={workspace.avatar}
                      initials={workspace.initials}
                      alt={workspace.name}
                      className={`w-12 h-12 bg-${workspace.color}-100 text-${workspace.color}-700 dark:bg-${workspace.color}-900 dark:text-${workspace.color}-300`}
                    />
                    <div className="flex items-center space-x-2">
                      {getRoleBadge(workspace.role)}
                      {getStatusBadge(workspace.status)}
                    </div>
                  </div>

                  <div className="mb-4">
                    <Subheading level={3} className="text-lg mb-2">
                      {workspace.name}
                    </Subheading>
                    <Text className="text-sm text-zinc-600 dark:text-zinc-400 line-clamp-2">
                      {workspace.description}
                    </Text>
                  </div>

                  <div className="flex items-center justify-between text-xs text-zinc-500 mb-4">
                    <div className="flex items-center space-x-4">
                      <span>{workspace.memberCount} members</span>
                      <span>{workspace.flowCount} flows</span>
                    </div>
                    <span>{workspace.lastActivity}</span>
                  </div>

                  <div className="flex items-center space-x-2">
                    <Button outline className="text-xs">Restore</Button>
                    <Button plain className="text-xs text-red-600 hover:text-red-700">Delete</Button>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Create Workspace CTA */}
        <div className="bg-gradient-to-r from-blue-50 to-purple-50 dark:from-blue-900/20 dark:to-purple-900/20 p-8 rounded-lg border border-blue-200 dark:border-blue-800 text-center">
          <div className="max-w-2xl mx-auto">
            <Heading level={2} className="text-2xl mb-4">Ready to create a new workspace?</Heading>
            <Text className="text-zinc-600 dark:text-zinc-400 mb-6">
              Bring your team together and start building powerful workflows. Set up a dedicated space for your project with custom permissions and integrations.
            </Text>
            <div className="flex items-center justify-center space-x-4">
              <Button color="blue" onClick={() => setIsCreateModalOpen(true)}>
                <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                </svg>
                Create New Workspace
              </Button>
              <Button outline>
                <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                </svg>
                Browse Templates
              </Button>
            </div>
          </div>
        </div>
      </div>

      {/* Create Workspace Modal Placeholder */}
      {isCreateModalOpen && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-zinc-800 p-6 rounded-lg max-w-md w-full mx-4">
            <Heading level={2} className="text-xl mb-4">Create New Workspace</Heading>
            <Text className="text-sm text-zinc-600 dark:text-zinc-400 mb-4">
              This would open a form to create a new workspace with name, description, and initial settings.
            </Text>
            <div className="flex items-center space-x-3">
              <Button color="blue" onClick={() => setIsCreateModalOpen(false)}>
                Create Workspace
              </Button>
              <Button outline onClick={() => setIsCreateModalOpen(false)}>
                Cancel
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}