import { useState } from 'react'
import { Button } from './lib/Button/button'
import { Badge } from './lib/Badge/badge'
import { Avatar } from './lib/Avatar/avatar'
import { Text, TextLink } from './lib/Text/text'
import { Heading, Subheading } from './lib/Heading/heading'
import { Navigation, type User, type Organization } from './lib/Navigation/navigation'

interface DashboardPageProps {
  onSignOut?: () => void
  onWorkspaceSelect?: (workspaceId: string) => void
}

interface RecentWork {
  id: string
  type: 'workflow' | 'document' | 'project'
  title: string
  description: string
  lastAccessed: string
  workspace: string
  status?: 'active' | 'paused' | 'completed' | 'draft'
}

interface Workspace {
  id: string
  name: string
  description: string
  memberCount: number
  flowCount: number
  lastActivity: string
  color: 'blue' | 'green' | 'purple' | 'orange' | 'red'
  role: 'owner' | 'admin' | 'member'
  avatar?: string
  initials: string
}

export function DashboardPage({ onSignOut, onWorkspaceSelect }: DashboardPageProps) {
  // Mock user data
  const currentUser: User = {
    id: '1',
    name: 'John Doe',
    email: 'john@superplane.com',
    initials: 'JD',
    avatar: 'https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?w=64&h=64&fit=crop&crop=face',
  }

  // Mock organization data for navigation (will be empty/placeholder for dashboard)
  const currentOrganization: Organization = {
    id: 'personal',
    name: 'Personal',
    initials: 'P',
  }

  // Mock recent work data
  const recentWork: RecentWork[] = [
    {
      id: '1',
      type: 'workflow',
      title: 'Deploy to Production',
      description: 'Automated deployment pipeline for the main application',
      lastAccessed: '2 hours ago',
      workspace: 'Development Team',
      status: 'active'
    },
    {
      id: '2',
      type: 'document',
      title: 'API Documentation',
      description: 'Complete REST API documentation and examples',
      lastAccessed: '4 hours ago',
      workspace: 'Development Team',
      status: 'draft'
    },
    {
      id: '3',
      type: 'workflow',
      title: 'Security Scan',
      description: 'Weekly security vulnerability assessment',
      lastAccessed: '1 day ago',
      workspace: 'Security Team',
      status: 'completed'
    },
    {
      id: '4',
      type: 'project',
      title: 'Mobile App Redesign',
      description: 'Complete redesign of the mobile application UI',
      lastAccessed: '2 days ago',
      workspace: 'Design Team',
      status: 'active'
    },
    {
      id: '5',
      type: 'workflow',
      title: 'Data Backup',
      description: 'Automated daily backup of production databases',
      lastAccessed: '3 days ago',
      workspace: 'Operations',
      status: 'active'
    },
    {
      id: '6',
      type: 'document',
      title: 'User Research Report',
      description: 'Q4 user research findings and recommendations',
      lastAccessed: '1 week ago',
      workspace: 'Product Team',
      status: 'completed'
    }
  ]

  // Mock workspaces data
  const workspaces: Workspace[] = [
    {
      id: '1',
      name: 'Development Team',
      description: 'Core development and engineering workflows',
      memberCount: 12,
      flowCount: 24,
      lastActivity: '5 minutes ago',
      color: 'blue',
      role: 'owner',
      initials: 'DT'
    },
    {
      id: '2',
      name: 'Design Team',
      description: 'Product design and user experience',
      memberCount: 6,
      flowCount: 8,
      lastActivity: '2 hours ago',
      color: 'purple',
      role: 'admin',
      initials: 'DS'
    },
    {
      id: '3',
      name: 'Security Team',
      description: 'Security audits and compliance workflows',
      memberCount: 4,
      flowCount: 12,
      lastActivity: '1 day ago',
      color: 'red',
      role: 'member',
      initials: 'ST'
    },
    {
      id: '4',
      name: 'Product Team',
      description: 'Product management and strategy',
      memberCount: 8,
      flowCount: 15,
      lastActivity: '3 days ago',
      color: 'green',
      role: 'admin',
      initials: 'PT'
    },
    {
      id: '5',
      name: 'Operations',
      description: 'Infrastructure and deployment operations',
      memberCount: 5,
      flowCount: 18,
      lastActivity: '1 week ago',
      color: 'orange',
      role: 'member',
      initials: 'OP'
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

  const getWorkTypeIcon = (type: RecentWork['type']) => {
    switch (type) {
      case 'workflow':
        return (
          <svg className="w-5 h-5 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
          </svg>
        )
      case 'document':
        return (
          <svg className="w-5 h-5 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
          </svg>
        )
      case 'project':
        return (
          <svg className="w-5 h-5 text-purple-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
          </svg>
        )
    }
  }

  const getStatusBadge = (status?: string) => {
    if (!status) return null
    
    switch (status) {
      case 'active':
        return <Badge color="green">Active</Badge>
      case 'paused':
        return <Badge color="yellow">Paused</Badge>
      case 'completed':
        return <Badge color="blue">Completed</Badge>
      case 'draft':
        return <Badge color="zinc">Draft</Badge>
      default:
        return null
    }
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
        {/* Welcome Header */}
        <div className="mb-8">
          <Heading level={1} className="text-3xl mb-2">
            Welcome back, {currentUser.name.split(' ')[0]}
          </Heading>
          <Text className="text-lg text-zinc-600 dark:text-zinc-400">
            Here's what you've been working on recently
          </Text>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
          {/* Recent Work */}
          <div className="lg:col-span-2 space-y-6">
            <div className="flex items-center justify-between">
              <Subheading level={2}>Recent Work</Subheading>
              <TextLink href="#" className="text-sm">View all →</TextLink>
            </div>

            <div className="space-y-4">
              {recentWork.map((work) => (
                <div key={work.id} className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700 hover:shadow-md transition-shadow cursor-pointer">
                  <div className="flex items-start justify-between">
                    <div className="flex items-start space-x-4">
                      <div className="flex-shrink-0 mt-1">
                        {getWorkTypeIcon(work.type)}
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center space-x-2 mb-1">
                          <Subheading level={3} className="text-base">{work.title}</Subheading>
                          {getStatusBadge(work.status)}
                        </div>
                        <Text className="text-sm text-zinc-600 dark:text-zinc-400 mb-2">
                          {work.description}
                        </Text>
                        <div className="flex items-center space-x-4 text-xs text-zinc-500">
                          <span>{work.workspace}</span>
                          <span>•</span>
                          <span>Accessed {work.lastAccessed}</span>
                        </div>
                      </div>
                    </div>
                    <Button plain className="text-zinc-400 hover:text-zinc-600">
                      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 5v.01M12 12v.01M12 19v.01M12 6a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2z" />
                      </svg>
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Workspaces Sidebar */}
          <div className="space-y-6">
            <div className="flex items-center justify-between">
              <Subheading level={2}>Your Workspaces</Subheading>
              <Button outline className="text-sm">
                <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                </svg>
                Create
              </Button>
            </div>

            <div className="space-y-4">
              {workspaces.map((workspace) => (
                <div 
                  key={workspace.id} 
                  className="bg-white dark:bg-zinc-800 p-4 rounded-lg border border-zinc-200 dark:border-zinc-700 hover:shadow-md transition-shadow cursor-pointer"
                  onClick={() => onWorkspaceSelect?.(workspace.id)}
                >
                  <div className="flex items-start justify-between mb-3">
                    <div className="flex items-center space-x-3">
                      <Avatar
                        src={workspace.avatar}
                        initials={workspace.initials}
                        alt={workspace.name}
                        className={`w-10 h-10 bg-${workspace.color}-100 text-${workspace.color}-700`}
                      />
                      <div>
                        <Subheading level={3} className="text-base">{workspace.name}</Subheading>
                        {getRoleBadge(workspace.role)}
                      </div>
                    </div>
                  </div>
                  
                  <Text className="text-sm text-zinc-600 dark:text-zinc-400 mb-3">
                    {workspace.description}
                  </Text>
                  
                  <div className="flex items-center justify-between text-xs text-zinc-500">
                    <div className="flex items-center space-x-4">
                      <span>{workspace.memberCount} members</span>
                      <span>{workspace.flowCount} flows</span>
                    </div>
                    <span>{workspace.lastActivity}</span>
                  </div>
                </div>
              ))}
            </div>

            {/* Quick Actions */}
            <div className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700">
              <Subheading level={2} className="mb-4">Quick Actions</Subheading>
              <div className="space-y-3">
                <Button className="w-full justify-start" color="blue">
                  <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
                  </svg>
                  Create Workflow
                </Button>
                <Button className="w-full justify-start" outline>
                  <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0" />
                  </svg>
                  Invite Teammates
                </Button>
                <Button className="w-full justify-start" outline>
                  <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
                  </svg>
                  New Workspace
                </Button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}