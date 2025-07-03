import { useState } from 'react'
import { Button } from './lib/Button/button'
import { Badge } from './lib/Badge/badge'
import { Avatar } from './lib/Avatar/avatar'
import { Text, TextLink } from './lib/Text/text'
import { Heading, Subheading } from './lib/Heading/heading'
import { Navigation, type User, type Organization } from './lib/Navigation/navigation'
import clsx from 'clsx'

interface AdministrationPageProps {
  onSignOut?: () => void
}

interface AdminUser {
  id: string
  name: string
  email: string
  role: 'owner' | 'admin' | 'member'
  status: 'active' | 'invited' | 'suspended'
  lastActive: string
  avatar?: string
  initials: string
}

interface Group {
  id: string
  name: string
  description: string
  memberCount: number
  permissions: string[]
  color: 'blue' | 'green' | 'purple' | 'orange' | 'red'
}

interface Role {
  id: string
  name: string
  description: string
  permissions: string[]
  userCount: number
  type: 'system' | 'custom'
}

type AdminSection = 'overview' | 'users' | 'groups' | 'roles' | 'settings'

export function AdministrationPage({ onSignOut }: AdministrationPageProps) {
  const [activeSection, setActiveSection] = useState<AdminSection>('overview')

  // Mock user and organization data
  const currentUser: User = {
    id: '1',
    name: 'John Doe',
    email: 'john@superplane.com',
    initials: 'JD',
    avatar: 'https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?w=64&h=64&fit=crop&crop=face',
  }

  const currentOrganization: Organization = {
    id: '1',
    name: 'Development Team',
    plan: 'Pro Plan',
    initials: 'DT',
  }

  // Mock data
  const users: AdminUser[] = [
    { id: '1', name: 'John Doe', email: 'john@superplane.com', role: 'owner', status: 'active', lastActive: '2 hours ago', initials: 'JD', avatar: 'https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?w=64&h=64&fit=crop&crop=face' },
    { id: '2', name: 'Sarah Wilson', email: 'sarah@superplane.com', role: 'admin', status: 'active', lastActive: '5 minutes ago', initials: 'SW', avatar: 'https://images.unsplash.com/photo-1494790108755-2616b612b786?w=64&h=64&fit=crop&crop=face' },
    { id: '3', name: 'Mike Chen', email: 'mike@superplane.com', role: 'member', status: 'active', lastActive: '1 day ago', initials: 'MC' },
    { id: '4', name: 'Emily Rodriguez', email: 'emily@superplane.com', role: 'member', status: 'invited', lastActive: 'Never', initials: 'ER' },
    { id: '5', name: 'David Kim', email: 'david@superplane.com', role: 'admin', status: 'suspended', lastActive: '1 week ago', initials: 'DK' },
  ]

  const groups: Group[] = [
    { id: '1', name: 'Engineering', description: 'Core development team with full workflow access', memberCount: 8, permissions: ['create_workflows', 'manage_deployments', 'view_analytics'], color: 'blue' },
    { id: '2', name: 'Design', description: 'Product design team with review permissions', memberCount: 4, permissions: ['view_workflows', 'create_designs', 'review_changes'], color: 'purple' },
    { id: '3', name: 'Operations', description: 'Infrastructure and monitoring team', memberCount: 3, permissions: ['manage_infrastructure', 'view_logs', 'create_alerts'], color: 'green' },
    { id: '4', name: 'Security', description: 'Security review and compliance team', memberCount: 2, permissions: ['security_review', 'audit_logs', 'manage_permissions'], color: 'red' },
  ]

  const roles: Role[] = [
    { id: '1', name: 'Organization Owner', description: 'Full access to all organization features', permissions: ['*'], userCount: 1, type: 'system' },
    { id: '2', name: 'Administrator', description: 'Manage users, groups, and organization settings', permissions: ['manage_users', 'manage_groups', 'view_analytics'], userCount: 2, type: 'system' },
    { id: '3', name: 'Member', description: 'Standard user with workflow access', permissions: ['create_workflows', 'view_own_workflows'], userCount: 12, type: 'system' },
    { id: '4', name: 'Workflow Manager', description: 'Custom role for workflow administration', permissions: ['manage_all_workflows', 'view_analytics', 'manage_integrations'], userCount: 3, type: 'custom' },
  ]

  // Sidebar navigation items
  const sidebarItems = [
    {
      id: 'overview',
      label: 'Overview',
      icon: (
        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
        </svg>
      ),
    },
    {
      id: 'users',
      label: 'Users',
      icon: (
        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0" />
        </svg>
      ),
      count: users.length,
    },
    {
      id: 'groups',
      label: 'Groups',
      icon: (
        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
        </svg>
      ),
      count: groups.length,
    },
    {
      id: 'roles',
      label: 'Roles',
      icon: (
        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
        </svg>
      ),
      count: roles.length,
    },
    {
      id: 'settings',
      label: 'Global Settings',
      icon: (
        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
        </svg>
      ),
    },
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
    switch (action) {
      case 'settings':
        setActiveSection('settings')
        break
      case 'billing':
        setActiveSection('settings')
        break
      case 'members':
        setActiveSection('users')
        break
    }
  }

  const getStatusBadge = (status: AdminUser['status']) => {
    switch (status) {
      case 'active':
        return <Badge color="green">Active</Badge>
      case 'invited':
        return <Badge color="yellow">Invited</Badge>
      case 'suspended':
        return <Badge color="red">Suspended</Badge>
    }
  }

  const getRoleBadge = (role: AdminUser['role']) => {
    switch (role) {
      case 'owner':
        return <Badge color="purple">Owner</Badge>
      case 'admin':
        return <Badge color="blue">Admin</Badge>
      case 'member':
        return <Badge color="zinc">Member</Badge>
    }
  }

  const renderOverviewContent = () => (
    <div className="space-y-6">
      <div>
        <Heading level={1} className="text-3xl mb-2">Organization Administration</Heading>
        <Text className="text-lg text-zinc-600 dark:text-zinc-400">
          Manage your organization settings, users, and permissions
        </Text>
      </div>

      {/* Organization Highlight */}
      <div className="bg-gradient-to-r from-blue-50 to-purple-50 dark:from-blue-900/20 dark:to-purple-900/20 p-6 rounded-lg border border-blue-200 dark:border-blue-800">
        <div className="flex items-center space-x-4">
          <Avatar
            src={currentOrganization.avatar}
            initials={currentOrganization.initials}
            alt={currentOrganization.name}
            className="w-16 h-16 text-lg"
          />
          <div>
            <Heading level={2} className="text-2xl mb-1">{currentOrganization.name}</Heading>
            <div className="flex items-center space-x-3">
              <Badge color="blue">{currentOrganization.plan}</Badge>
              <Text className="text-sm text-zinc-600 dark:text-zinc-400">
                {users.filter(u => u.status === 'active').length} active users
              </Text>
            </div>
          </div>
        </div>
      </div>

      {/* Quick Stats */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6">
        <div className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700">
          <div className="flex items-center">
            <div className="flex-shrink-0">
              <svg className="w-8 h-8 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0" />
              </svg>
            </div>
            <div className="ml-4">
              <Subheading level={3} className="text-2xl">{users.length}</Subheading>
              <Text className="text-sm">Total Users</Text>
            </div>
          </div>
        </div>

        <div className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700">
          <div className="flex items-center">
            <div className="flex-shrink-0">
              <svg className="w-8 h-8 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
            </div>
            <div className="ml-4">
              <Subheading level={3} className="text-2xl">{users.filter(u => u.status === 'active').length}</Subheading>
              <Text className="text-sm">Active Users</Text>
            </div>
          </div>
        </div>

        <div className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700">
          <div className="flex items-center">
            <div className="flex-shrink-0">
              <svg className="w-8 h-8 text-purple-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
              </svg>
            </div>
            <div className="ml-4">
              <Subheading level={3} className="text-2xl">{groups.length}</Subheading>
              <Text className="text-sm">Groups</Text>
            </div>
          </div>
        </div>

        <div className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700">
          <div className="flex items-center">
            <div className="flex-shrink-0">
              <svg className="w-8 h-8 text-orange-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
              </svg>
            </div>
            <div className="ml-4">
              <Subheading level={3} className="text-2xl">{roles.length}</Subheading>
              <Text className="text-sm">Roles</Text>
            </div>
          </div>
        </div>
      </div>

      {/* Quick Actions */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <Button 
          className="justify-start h-auto p-4" 
          color="blue"
          onClick={() => setActiveSection('users')}
        >
          <div className="text-left">
            <div className="font-medium">Manage Users</div>
            <div className="text-xs opacity-75">Invite, edit, and remove users</div>
          </div>
        </Button>
        <Button 
          className="justify-start h-auto p-4" 
          outline
          onClick={() => setActiveSection('groups')}
        >
          <div className="text-left">
            <div className="font-medium">Manage Groups</div>
            <div className="text-xs opacity-75">Organize users into groups</div>
          </div>
        </Button>
        <Button 
          className="justify-start h-auto p-4" 
          outline
          onClick={() => setActiveSection('roles')}
        >
          <div className="text-left">
            <div className="font-medium">Manage Roles</div>
            <div className="text-xs opacity-75">Configure permissions and access</div>
          </div>
        </Button>
        <Button 
          className="justify-start h-auto p-4" 
          outline
          onClick={() => setActiveSection('settings')}
        >
          <div className="text-left">
            <div className="font-medium">Global Settings</div>
            <div className="text-xs opacity-75">Organization-wide configuration</div>
          </div>
        </Button>
      </div>
    </div>
  )

  const renderUsersContent = () => (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <Heading level={1} className="text-3xl mb-2">Users</Heading>
          <Text className="text-lg text-zinc-600 dark:text-zinc-400">
            Manage organization members and their permissions
          </Text>
        </div>
        <Button color="blue">
          <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
          Invite User
        </Button>
      </div>

      <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700">
        <div className="p-6">
          <div className="space-y-4">
            {users.map((user) => (
              <div key={user.id} className="flex items-center justify-between p-4 bg-zinc-50 dark:bg-zinc-700 rounded-lg">
                <div className="flex items-center space-x-4">
                  <Avatar
                    src={user.avatar}
                    initials={user.initials}
                    alt={user.name}
                    className="w-10 h-10"
                  />
                  <div>
                    <Subheading level={3} className="text-base">{user.name}</Subheading>
                    <Text className="text-sm text-zinc-600 dark:text-zinc-400">{user.email}</Text>
                    <Text className="text-xs text-zinc-500">Last active: {user.lastActive}</Text>
                  </div>
                </div>
                <div className="flex items-center space-x-3">
                  {getRoleBadge(user.role)}
                  {getStatusBadge(user.status)}
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
      </div>
    </div>
  )

  const renderGroupsContent = () => (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <Heading level={1} className="text-3xl mb-2">Groups</Heading>
          <Text className="text-lg text-zinc-600 dark:text-zinc-400">
            Organize users and manage group permissions
          </Text>
        </div>
        <Button color="blue">
          <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
          Create Group
        </Button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {groups.map((group) => (
          <div key={group.id} className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700">
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center space-x-3">
                <div className={`w-4 h-4 rounded-full bg-${group.color}-500`}></div>
                <Subheading level={3} className="text-lg">{group.name}</Subheading>
              </div>
              <Badge color="zinc">{group.memberCount} members</Badge>
            </div>
            <Text className="text-sm text-zinc-600 dark:text-zinc-400 mb-4">{group.description}</Text>
            <div className="space-y-2">
              <Text className="text-xs font-medium text-zinc-500">Permissions:</Text>
              <div className="flex flex-wrap gap-1">
                {group.permissions.slice(0, 3).map((permission) => (
                  <Badge key={permission} color="blue" className="text-xs">
                    {permission.replace('_', ' ')}
                  </Badge>
                ))}
                {group.permissions.length > 3 && (
                  <Badge color="zinc" className="text-xs">
                    +{group.permissions.length - 3} more
                  </Badge>
                )}
              </div>
            </div>
            <div className="mt-4 pt-4 border-t border-zinc-200 dark:border-zinc-700">
              <Button outline className="w-full">Manage Group</Button>
            </div>
          </div>
        ))}
      </div>
    </div>
  )

  const renderRolesContent = () => (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <Heading level={1} className="text-3xl mb-2">Roles</Heading>
          <Text className="text-lg text-zinc-600 dark:text-zinc-400">
            Define permissions and access levels
          </Text>
        </div>
        <Button color="blue">
          <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
          Create Role
        </Button>
      </div>

      <div className="space-y-4">
        {roles.map((role) => (
          <div key={role.id} className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700">
            <div className="flex items-center justify-between">
              <div className="flex-1">
                <div className="flex items-center space-x-3 mb-2">
                  <Subheading level={3} className="text-lg">{role.name}</Subheading>
                  <Badge color={role.type === 'system' ? 'zinc' : 'blue'}>
                    {role.type === 'system' ? 'System' : 'Custom'}
                  </Badge>
                  <Badge color="green">{role.userCount} users</Badge>
                </div>
                <Text className="text-sm text-zinc-600 dark:text-zinc-400 mb-3">{role.description}</Text>
                <div className="flex flex-wrap gap-1">
                  {role.permissions.slice(0, 4).map((permission) => (
                    <Badge key={permission} color="purple" className="text-xs">
                      {permission === '*' ? 'All permissions' : permission.replace('_', ' ')}
                    </Badge>
                  ))}
                  {role.permissions.length > 4 && (
                    <Badge color="zinc" className="text-xs">
                      +{role.permissions.length - 4} more
                    </Badge>
                  )}
                </div>
              </div>
              <div className="flex items-center space-x-2">
                <Button outline disabled={role.type === 'system'}>
                  {role.type === 'system' ? 'View' : 'Edit'}
                </Button>
                <Button plain className="text-zinc-400 hover:text-zinc-600">
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 5v.01M12 12v.01M12 19v.01M12 6a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2z" />
                  </svg>
                </Button>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  )

  const renderSettingsContent = () => (
    <div className="space-y-6">
      <div>
        <Heading level={1} className="text-3xl mb-2">Global Settings</Heading>
        <Text className="text-lg text-zinc-600 dark:text-zinc-400">
          Configure organization-wide settings and preferences
        </Text>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700">
          <div className="px-6 py-4 border-b border-zinc-200 dark:border-zinc-700">
            <Subheading level={2}>Security Settings</Subheading>
          </div>
          <div className="p-6 space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <Text className="font-medium">Two-Factor Authentication</Text>
                <Text className="text-sm text-zinc-500">Require 2FA for all users</Text>
              </div>
              <Button outline>Configure</Button>
            </div>
            <div className="flex items-center justify-between">
              <div>
                <Text className="font-medium">Single Sign-On</Text>
                <Text className="text-sm text-zinc-500">SAML/OIDC integration</Text>
              </div>
              <Button outline>Setup</Button>
            </div>
            <div className="flex items-center justify-between">
              <div>
                <Text className="font-medium">Session Timeout</Text>
                <Text className="text-sm text-zinc-500">Automatic logout after inactivity</Text>
              </div>
              <Button outline>Edit</Button>
            </div>
          </div>
        </div>

        <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700">
          <div className="px-6 py-4 border-b border-zinc-200 dark:border-zinc-700">
            <Subheading level={2}>Workflow Settings</Subheading>
          </div>
          <div className="p-6 space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <Text className="font-medium">Default Permissions</Text>
                <Text className="text-sm text-zinc-500">New workflow permissions</Text>
              </div>
              <Button outline>Configure</Button>
            </div>
            <div className="flex items-center justify-between">
              <div>
                <Text className="font-medium">Retention Policy</Text>
                <Text className="text-sm text-zinc-500">Workflow data retention</Text>
              </div>
              <Button outline>Setup</Button>
            </div>
            <div className="flex items-center justify-between">
              <div>
                <Text className="font-medium">Execution Limits</Text>
                <Text className="text-sm text-zinc-500">Resource usage limits</Text>
              </div>
              <Button outline>Edit</Button>
            </div>
          </div>
        </div>
      </div>
    </div>
  )

  const renderContent = () => {
    switch (activeSection) {
      case 'overview':
        return renderOverviewContent()
      case 'users':
        return renderUsersContent()
      case 'groups':
        return renderGroupsContent()
      case 'roles':
        return renderRolesContent()
      case 'settings':
        return renderSettingsContent()
      default:
        return renderOverviewContent()
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

      <div className="flex">
        {/* Sidebar */}
        <aside className="w-64 bg-white dark:bg-zinc-800 border-r border-zinc-200 dark:border-zinc-700 min-h-[calc(100vh-4rem)]">
          <nav className="p-4">
            <div className="space-y-1">
              {sidebarItems.map((item) => (
                <button
                  key={item.id}
                  onClick={() => setActiveSection(item.id as AdminSection)}
                  className={clsx(
                    'w-full flex items-center justify-between px-3 py-2 text-sm font-medium rounded-md transition-colors',
                    activeSection === item.id
                      ? 'bg-blue-50 text-blue-700 dark:bg-blue-900/20 dark:text-blue-300'
                      : 'text-zinc-700 hover:text-zinc-900 hover:bg-zinc-100 dark:text-zinc-300 dark:hover:text-white dark:hover:bg-zinc-700'
                  )}
                >
                  <div className="flex items-center space-x-3">
                    <span className={clsx(
                      activeSection === item.id
                        ? 'text-blue-500 dark:text-blue-400'
                        : 'text-zinc-400'
                    )}>
                      {item.icon}
                    </span>
                    <span>{item.label}</span>
                  </div>
                  {item.count && (
                    <Badge 
                      color={activeSection === item.id ? 'blue' : 'zinc'} 
                      className="text-xs"
                    >
                      {item.count}
                    </Badge>
                  )}
                </button>
              ))}
            </div>
          </nav>
        </aside>

        {/* Main Content */}
        <main className="flex-1 p-8">
          {renderContent()}
        </main>
      </div>
    </div>
  )
}