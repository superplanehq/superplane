import { useState } from 'react'
import { Button } from './lib/Button/button'
import { Badge } from './lib/Badge/badge'
import { Avatar } from './lib/Avatar/avatar'
import { Text } from './lib/Text/text'
import { Heading } from './lib/Heading/heading'
import { Tabs } from './lib/Tabs/tabs'
import { NavigationVertical, type User, type Organization, type NavigationLink } from './lib/Navigation/navigation-vertical'
import { Link } from './lib/Link/link'
import { MaterialSymbol } from './lib/MaterialSymbol/material-symbol'

interface MainLandingPageProps {
  onSignOut?: () => void
  navigationLinks?: NavigationLink[]
  onLinkClick?: (linkId: string) => void
}

interface WorkItem {
  id: string
  name: string
  createdAt: string
  createdBy: {
    name: string
    avatar?: string
    initials: string
  }
}

interface Credential {
  id: string
  name: string
  type: 'api_key' | 'oauth' | 'database' | 'ssh_key'
  createdAt: string
  createdBy: {
    name: string
    avatar?: string
    initials: string
  }
  status: 'active' | 'expired' | 'revoked'
}

export function MainLandingPage({ onSignOut, navigationLinks = [], onLinkClick }: MainLandingPageProps) {
  const [activeTab, setActiveTab] = useState<'my-work' | 'credentials'>('my-work')

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

  // Mock work items data
  const workItems: WorkItem[] = [
    {
      id: '1',
      name: 'User Onboarding Flow',
      createdAt: '2 hours ago',
      createdBy: {
        name: 'John Doe',
        initials: 'JD',
        avatar: 'https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?w=64&h=64&fit=crop&crop=face',
      }
    },
    {
      id: '2',
      name: 'Payment Processing Workflow',
      createdAt: '1 day ago',
      createdBy: {
        name: 'Sarah Wilson',
        initials: 'SW',
      }
    },
    {
      id: '3',
      name: 'Customer Support Dashboard',
      createdAt: '3 days ago',
      createdBy: {
        name: 'Mike Chen',
        initials: 'MC',
      }
    },
    {
      id: '4',
      name: 'Inventory Management System',
      createdAt: '1 week ago',
      createdBy: {
        name: 'Emily Rodriguez',
        initials: 'ER',
      }
    },
    {
      id: '5',
      name: 'Analytics Dashboard',
      createdAt: '2 weeks ago',
      createdBy: {
        name: 'John Doe',
        initials: 'JD',
        avatar: 'https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?w=64&h=64&fit=crop&crop=face',
      }
    }
  ]

  // Mock credentials data
  const credentials: Credential[] = [
    {
      id: '1',
      name: 'AWS Production Keys',
      type: 'api_key',
      createdAt: '1 week ago',
      createdBy: {
        name: 'John Doe',
        initials: 'JD',
        avatar: 'https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?w=64&h=64&fit=crop&crop=face',
      },
      status: 'active'
    },
    {
      id: '2',
      name: 'GitHub OAuth App',
      type: 'oauth',
      createdAt: '2 weeks ago',
      createdBy: {
        name: 'Sarah Wilson',
        initials: 'SW',
        avatar: 'https://images.unsplash.com/photo-1494790108755-2616b612b786?w=64&h=64&fit=crop&crop=face',
      },
      status: 'active'
    },
    {
      id: '3',
      name: 'Production Database',
      type: 'database',
      createdAt: '1 month ago',
      createdBy: {
        name: 'Mike Chen',
        initials: 'MC',
      },
      status: 'active'
    },
    {
      id: '4',
      name: 'Legacy API Keys',
      type: 'api_key',
      createdAt: '3 months ago',
      createdBy: {
        name: 'Emily Rodriguez',
        initials: 'ER',
      },
      status: 'expired'
    },
    {
      id: '5',
      name: 'SSH Deploy Key',
      type: 'ssh_key',
      createdAt: '2 months ago',
      createdBy: {
        name: 'David Kim',
        initials: 'DK',
      },
      status: 'revoked'
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

  // Navigation link click handler
  const handleLinkClick = (linkId: string) => {
    if (onLinkClick) {
      onLinkClick(linkId)
    } else {
      console.log(`Navigation link clicked: ${linkId}`)
    }
  }

  const getWorkItemIcon = () => {
    return <MaterialSymbol name="canvas" className="text-blue-600" />
  }

  const getCredentialIcon = (type: Credential['type']) => {
    switch (type) {
      case 'api_key':
        return (
          <svg className="w-5 h-5 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
          </svg>
        )
      case 'oauth':
        return (
          <svg className="w-5 h-5 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
          </svg>
        )
      case 'database':
        return (
          <svg className="w-5 h-5 text-orange-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4" />
          </svg>
        )
      case 'ssh_key':
        return (
          <svg className="w-5 h-5 text-purple-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 11V7a4 4 0 118 0m-4 8v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2z" />
          </svg>
        )
    }
  }

  const getCredentialStatusBadge = (status: Credential['status']) => {
    switch (status) {
      case 'active':
        return <Badge color="green">Active</Badge>
      case 'expired':
        return <Badge color="yellow">Expired</Badge>
      case 'revoked':
        return <Badge color="red">Revoked</Badge>
    }
  }

  const renderMyWorkContent = () => (
    <div className="space-y-4">
      <div className="flex justify-end">
        <Button color="blue">
          <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
          New Canvas
        </Button>
      </div>
      
      <div className="space-y-3">
        {workItems.map((item) => (
          <div key={item.id} className="bg-white dark:bg-zinc-800 p-4 rounded-lg border border-zinc-200 dark:border-zinc-700 hover:shadow-sm transition-shadow">
            <div className="flex items-center justify-between">
              <div className="flex items-center space-x-3">
                <div className='p-2 bg-zinc-100 dark:bg-zinc-800 rounded-lg'>
                {getWorkItemIcon()}
                </div>
                <div>
                  <Link href={`/work-item/${item.id}`} className="font-medium text-sm text-blue-600 hover:text-blue-500">{item.name}</Link>
                  <div className="flex items-center space-x-4 text-xs text-zinc-500">
                    <span>Created {item.createdAt}</span>
                    <span>•</span>
                    <div className="flex items-center space-x-2">
                      <Avatar
                        src={item.createdBy.avatar}
                        initials={item.createdBy.initials}
                        alt={item.createdBy.name}
                        className="w-4 h-4"
                      />
                      <span>by {item.createdBy.name}</span>
                    </div>
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
  )

  const renderCredentialsContent = () => (
    <div className="space-y-4">
      <div className="flex justify-end">
        <Button color="blue">
          <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
          New Credentials
        </Button>
      </div>
      
      <div className="space-y-3">
        {credentials.map((credential) => (
          <div key={credential.id} className="bg-white dark:bg-zinc-800 p-4 rounded-lg border border-zinc-200 dark:border-zinc-700 hover:shadow-sm transition-shadow">
            <div className="flex items-center justify-between">
              <div className="flex items-center space-x-3">
                {getCredentialIcon(credential.type)}
                <div>
                  <div className="flex items-center space-x-2">
                    <Text className="font-medium">{credential.name}</Text>
                    {getCredentialStatusBadge(credential.status)}
                  </div>
                  <div className="flex items-center space-x-4 text-sm text-zinc-500">
                    <span>Created {credential.createdAt}</span>
                    <span>•</span>
                    <div className="flex items-center space-x-2">
                      <Avatar
                        src={credential.createdBy.avatar}
                        initials={credential.createdBy.initials}
                        alt={credential.createdBy.name}
                        className="w-4 h-4"
                      />
                      <span>by {credential.createdBy.name}</span>
                    </div>
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
  )

  return (
    <div className="min-h-screen bg-zinc-50 dark:bg-zinc-900 flex">
      {/* Vertical Navigation */}
      <NavigationVertical
        user={currentUser}
        organization={currentOrganization}
        showOrganization={false}
        links={navigationLinks}
        onHelpClick={handleHelpClick}
        onUserMenuAction={handleUserMenuAction}
        onOrganizationMenuAction={handleOrganizationMenuAction}
        onLinkClick={handleLinkClick}
      />

      {/* Main Content */}
      <main className="flex-1 p-8">
        <div className="max-w-6xl mx-auto">
          {/* Welcome Heading */}
          <div className="mb-8">
            <Heading level={1} className="!text-3xl mb-1">Hello {currentUser.name}</Heading>
          </div>
          {/* Tabs */}
          <div className="space-y-6">
            <Tabs
              tabs={[
                { id: 'my-work', label: 'Canvases' },
                { id: 'credentials', label: 'Credentials' }
              ]}
              variant="underline"
              defaultTab="my-work"
              onTabChange={(tabId) => setActiveTab(tabId as 'my-work' | 'credentials')}
            />
            
            {/* Tab Content */}
            <div>
              {activeTab === 'my-work' && renderMyWorkContent()}
              {activeTab === 'credentials' && renderCredentialsContent()}
            </div>
          </div>
        </div>
      </main>
    </div>
  )
}