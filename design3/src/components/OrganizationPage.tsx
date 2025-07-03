import { useState } from 'react'
import { Button } from './lib/Button/button'
import { Badge } from './lib/Badge/badge'
import { Avatar } from './lib/Avatar/avatar'
import { Text, TextLink } from './lib/Text/text'
import { Heading, Subheading } from './lib/Heading/heading'
import { Tabs, type Tab } from './lib/Tabs/tabs'
import { NavigationAlt, type User, type Organization } from './lib/Navigation/navigation-alt'
import { Navigation } from './lib/Navigation/navigation'

interface OrganizationPageProps {
  onSignOut?: () => void
}

interface Workflow {
  id: string
  name: string
  status: 'active' | 'paused' | 'error'
  lastRun: string
  runs: number
}

interface Member {
  id: string
  name: string
  email: string
  role: 'owner' | 'admin' | 'member'
  avatar?: string
  initials: string
}

interface Group {
  id: string
  name: string
  description: string
  memberCount: number
  color: 'blue' | 'green' | 'purple' | 'orange'
}

type TabType = 'overview' | 'flows' | 'members' | 'groups' | 'settings'

export function OrganizationPage({ onSignOut }: OrganizationPageProps) {
  const [activeTab, setActiveTab] = useState<TabType>('overview')

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
    initials: 'DT',
  }

  // Mock data
  const workflows: Workflow[] = [
    { id: '1', name: 'Deploy to Production', status: 'active', lastRun: '2 hours ago', runs: 45 },
    { id: '2', name: 'Run Tests', status: 'active', lastRun: '5 minutes ago', runs: 128 },
    { id: '3', name: 'Build Docker Image', status: 'paused', lastRun: '1 day ago', runs: 23 },
    { id: '4', name: 'Security Scan', status: 'error', lastRun: '3 hours ago', runs: 67 },
    { id: '5', name: 'Deploy to Staging', status: 'active', lastRun: '1 hour ago', runs: 89 },
  ]

  const members: Member[] = [
    { id: '1', name: 'John Doe', email: 'john@superplane.com', role: 'owner', initials: 'JD' },
    { id: '2', name: 'Sarah Wilson', email: 'sarah@superplane.com', role: 'admin', initials: 'SW', avatar: 'https://images.unsplash.com/photo-1494790108755-2616b612b786?w=64&h=64&fit=crop&crop=face' },
    { id: '3', name: 'Mike Chen', email: 'mike@superplane.com', role: 'member', initials: 'MC' },
    { id: '4', name: 'Emily Rodriguez', email: 'emily@superplane.com', role: 'member', initials: 'ER', avatar: 'https://images.unsplash.com/photo-1438761681033-6461ffad8d80?w=64&h=64&fit=crop&crop=face' },
    { id: '5', name: 'David Kim', email: 'david@superplane.com', role: 'admin', initials: 'DK' },
  ]

  const groups: Group[] = [
    { id: '1', name: 'Engineering', description: 'Core development team', memberCount: 12, color: 'blue' },
    { id: '2', name: 'Design', description: 'Product design and UX', memberCount: 4, color: 'purple' },
    { id: '3', name: 'DevOps', description: 'Infrastructure and deployment', memberCount: 3, color: 'green' },
    { id: '4', name: 'QA', description: 'Quality assurance team', memberCount: 6, color: 'orange' },
  ]

  // Tab configuration
  const tabs: Tab[] = [
    {
      id: 'overview',
      label: 'Overview',
      icon: (
        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
        </svg>
      ),
    },
    {
      id: 'flows',
      label: 'Flows',
      icon: (
        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
        </svg>
      ),
      count: workflows.length,
    },
    {
      id: 'members',
      label: 'Members',
      icon: (
        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0" />
        </svg>
      ),
      count: members.length,
    },
    {
      id: 'groups',
      label: 'Groups',
      icon: (
        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
        </svg>
      ),
      count: groups.length,
    },
    {
      id: 'settings',
      label: 'Settings',
      icon: (
        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
        </svg>
      ),
    },
  ]

  // Navigation handlers
  const handleHelpClick = () => {
    // Open help documentation
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
        setActiveTab('settings')
        break
      case 'billing':
        setActiveTab('settings') // Navigate to settings tab for billing
        break
      case 'members':
        setActiveTab('members')
        break
    }
  }

  const getStatusBadge = (status: Workflow['status']) => {
    switch (status) {
      case 'active':
        return <Badge color="green">Active</Badge>
      case 'paused':
        return <Badge color="yellow">Paused</Badge>
      case 'error':
        return <Badge color="red">Error</Badge>
      default:
        return <Badge color="zinc">Unknown</Badge>
    }
  }

  const getRoleBadge = (role: Member['role']) => {
    switch (role) {
      case 'owner':
        return <Badge color="purple">Owner</Badge>
      case 'admin':
        return <Badge color="blue">Admin</Badge>
      case 'member':
        return <Badge color="zinc">Member</Badge>
      default:
        return <Badge color="zinc">Unknown</Badge>
    }
  }

  const renderTabContent = () => {
    switch (activeTab) {
      case 'overview':
        return renderOverviewTab()
      case 'flows':
        return renderFlowsTab()
      case 'members':
        return renderMembersTab()
      case 'groups':
        return renderGroupsTab()
      case 'settings':
        return renderSettingsTab()
      default:
        return renderOverviewTab()
    }
  }

  const renderOverviewTab = () => (
    <>
      {/* Quick Stats */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
        <div className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700">
          <div className="flex items-center">
            <div className="flex-shrink-0">
              <svg className="w-8 h-8 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v10a2 2 0 002 2h8a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
              </svg>
            </div>
            <div className="ml-4">
              <Subheading level={3} className="text-2xl">{workflows.length}</Subheading>
              <Text className="text-sm">Active Workflows</Text>
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
              <Subheading level={3} className="text-2xl">{members.length}</Subheading>
              <Text className="text-sm">Team Members</Text>
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
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
              </svg>
            </div>
            <div className="ml-4">
              <Subheading level={3} className="text-2xl">352</Subheading>
              <Text className="text-sm">Total Runs</Text>
            </div>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        {/* Recent Workflows */}
        <div className="lg:col-span-2">
          <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700">
            <div className="px-6 py-4 border-b border-zinc-200 dark:border-zinc-700">
              <div className="flex items-center justify-between">
                <Subheading level={2}>Recent Workflows</Subheading>
                <TextLink href="#" className="text-sm">View all workflows →</TextLink>
              </div>
            </div>
            <div className="p-6">
              <div className="space-y-4">
                {workflows.slice(0, 3).map((workflow) => (
                  <div key={workflow.id} className="flex items-center justify-between p-4 bg-zinc-50 dark:bg-zinc-700 rounded-lg">
                    <div className="flex items-center space-x-4">
                      <div className="flex-shrink-0">
                        <svg className="w-6 h-6 text-zinc-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v10a2 2 0 002 2h8a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
                        </svg>
                      </div>
                      <div>
                        <Subheading level={3} className="text-base">{workflow.name}</Subheading>
                        <Text className="text-sm">Last run {workflow.lastRun} • {workflow.runs} total runs</Text>
                      </div>
                    </div>
                    <div className="flex items-center space-x-3">
                      {getStatusBadge(workflow.status)}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>

        {/* Quick Summary */}
        <div className="space-y-6">
          <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 p-6">
            <Subheading level={2} className="mb-4">Team Summary</Subheading>
            <div className="space-y-3">
              <div className="flex justify-between">
                <Text className="text-sm">Total Members</Text>
                <Text className="text-sm font-medium">{members.length}</Text>
              </div>
              <div className="flex justify-between">
                <Text className="text-sm">Active Groups</Text>
                <Text className="text-sm font-medium">{groups.length}</Text>
              </div>
              <div className="flex justify-between">
                <Text className="text-sm">Running Workflows</Text>
                <Text className="text-sm font-medium">{workflows.filter(w => w.status === 'active').length}</Text>
              </div>
            </div>
          </div>
        </div>
      </div>
    </>
  )

  const renderFlowsTab = () => (
    <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700">
      <div className="px-6 py-4 border-b border-zinc-200 dark:border-zinc-700">
        <div className="flex items-center justify-between">
          <Subheading level={2}>All Workflows</Subheading>
          <Button color="blue">
            <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
            Create Workflow
          </Button>
        </div>
      </div>
      <div className="p-6">
        <div className="space-y-4">
          {workflows.map((workflow) => (
            <div key={workflow.id} className="flex items-center justify-between p-4 bg-zinc-50 dark:bg-zinc-700 rounded-lg">
              <div className="flex items-center space-x-4">
                <div className="flex-shrink-0">
                  <svg className="w-6 h-6 text-zinc-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v10a2 2 0 002 2h8a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
                  </svg>
                </div>
                <div>
                  <Subheading level={3} className="text-base">{workflow.name}</Subheading>
                  <Text className="text-sm">Last run {workflow.lastRun} • {workflow.runs} total runs</Text>
                </div>
              </div>
              <div className="flex items-center space-x-3">
                {getStatusBadge(workflow.status)}
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
  )

  const renderMembersTab = () => (
    <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700">
      <div className="px-6 py-4 border-b border-zinc-200 dark:border-zinc-700">
        <div className="flex items-center justify-between">
          <Subheading level={2}>Team Members</Subheading>
          <Button color="blue">
            <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
            Invite Member
          </Button>
        </div>
      </div>
      <div className="p-6">
        <div className="space-y-4">
          {members.map((member) => (
            <div key={member.id} className="flex items-center justify-between p-4 bg-zinc-50 dark:bg-zinc-700 rounded-lg">
              <div className="flex items-center space-x-4">
                <Avatar
                  src={member.avatar}
                  initials={member.initials}
                  alt={member.name}
                  className="w-10 h-10"
                />
                <div>
                  <Subheading level={3} className="text-base">{member.name}</Subheading>
                  <Text className="text-sm">{member.email}</Text>
                </div>
              </div>
              <div className="flex items-center space-x-3">
                {getRoleBadge(member.role)}
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
  )

  const renderGroupsTab = () => (
    <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700">
      <div className="px-6 py-4 border-b border-zinc-200 dark:border-zinc-700">
        <div className="flex items-center justify-between">
          <Subheading level={2}>Groups</Subheading>
          <Button color="blue">
            <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
            Create Group
          </Button>
        </div>
      </div>
      <div className="p-6">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {groups.map((group) => (
            <div key={group.id} className="p-6 bg-zinc-50 dark:bg-zinc-700 rounded-lg">
              <div className="flex items-center justify-between mb-3">
                <div className="flex items-center space-x-3">
                  <div className={`w-4 h-4 rounded-full bg-${group.color}-500`}></div>
                  <Subheading level={3} className="text-lg">{group.name}</Subheading>
                </div>
                <Badge color="zinc">{group.memberCount} members</Badge>
              </div>
              <Text className="text-sm mb-4">{group.description}</Text>
              <Button outline className="w-full">Manage Group</Button>
            </div>
          ))}
        </div>
      </div>
    </div>
  )

  const renderSettingsTab = () => (
    <div className="space-y-6">
      <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700">
        <div className="px-6 py-4 border-b border-zinc-200 dark:border-zinc-700">
          <Subheading level={2}>Organization Settings</Subheading>
        </div>
        <div className="p-6">
          <div className="space-y-6">
            <div>
              <Subheading level={3} className="text-base mb-2">Organization Name</Subheading>
              <Text className="text-sm mb-3">This is your organization's display name.</Text>
              <div className="flex gap-3">
                <input 
                  type="text" 
                  defaultValue="Development Team"
                  className="flex-1 px-3 py-2 border border-zinc-300 rounded-md dark:border-zinc-600 dark:bg-zinc-700"
                />
                <Button outline>Save</Button>
              </div>
            </div>
            
            <div>
              <Subheading level={3} className="text-base mb-2">Plan & Billing</Subheading>
              <div className="flex items-center justify-between p-4 bg-zinc-50 dark:bg-zinc-700 rounded-lg">
                <div>
                  <Text className="font-medium">Pro Plan</Text>
                  <Text className="text-sm">$29/month • Up to 100 workflows</Text>
                </div>
                <Button outline>Manage Billing</Button>
              </div>
            </div>

            <div>
              <Subheading level={3} className="text-base mb-2">Security</Subheading>
              <div className="space-y-3">
                <TextLink href="#" className="flex items-center text-sm">
                  <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
                  </svg>
                  Two-Factor Authentication
                </TextLink>
                <TextLink href="#" className="flex items-center text-sm">
                  <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
                  </svg>
                  API Keys
                </TextLink>
                <TextLink href="#" className="flex items-center text-sm">
                  <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
                  </svg>
                  Access Logs
                </TextLink>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )

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
        {/* Page Title */}
        <div className="mb-6">
          <Heading level={1} className="text-3xl mb-2">Development Team</Heading>
          <Text className="text-lg">Manage your workflows, team members, and organization settings</Text>
        </div>

        {/* Navigation Tabs */}
        <div className="mb-8">
          <Tabs
            tabs={tabs}
            defaultTab="overview"
            onTabChange={(tabId) => setActiveTab(tabId as TabType)}
            variant="default"
          />
        </div>

        {/* Tab Content */}
        {renderTabContent()}
      </div>
    </div>
  )
}