import { useState } from 'react'
import { Button } from './lib/Button/button'
import { Avatar } from './lib/Avatar/avatar'
import { Text } from './lib/Text/text'
import { Heading } from './lib/Heading/heading'
import { NavigationVertical, type User, type Organization, type NavigationLink } from './lib/Navigation/navigation-vertical'
import { Link } from './lib/Link/link'
import { MaterialSymbol } from './lib/MaterialSymbol/material-symbol'

interface CanvasesPageProps {
  onSignOut?: () => void
}

interface Canvas {
  id: string
  name: string
  description?: string
  createdAt: string
  updatedAt: string
  createdBy: {
    name: string
    avatar?: string
    initials: string
  }
  status: 'draft' | 'published' | 'archived'
  type: 'canvas' | 'dashboard' | 'form' | 'report'
}

export function CanvasesPage({ onSignOut }: CanvasesPageProps) {
  const [searchQuery, setSearchQuery] = useState('')
  const [filterStatus, setFilterStatus] = useState<'all' | 'draft' | 'published' | 'archived'>('all')

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

  // Mock canvases data
  const canvases: Canvas[] = [
    {
      id: '1',
      name: 'User Onboarding Flow',
      description: 'Interactive workflow for new user registration and setup',
      createdAt: '2 hours ago',
      updatedAt: '30 minutes ago',
      createdBy: {
        name: 'John Doe',
        initials: 'JD',
        avatar: 'https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?w=64&h=64&fit=crop&crop=face',
      },
      status: 'published',
      type: 'canvas'
    },
    {
      id: '2',
      name: 'Sales Analytics Dashboard',
      description: 'Real-time sales metrics and performance indicators',
      createdAt: '1 day ago',
      updatedAt: '4 hours ago',
      createdBy: {
        name: 'Sarah Wilson',
        initials: 'SW',
      },
      status: 'published',
      type: 'canvas'
    },
    {
      id: '3',
      name: 'Customer Support Form',
      description: 'Support ticket creation and tracking form',
      createdAt: '3 days ago',
      updatedAt: '1 day ago',
      createdBy: {
        name: 'Mike Chen',
        initials: 'MC',
      },
      status: 'draft',
      type: 'form'
    },
    {
      id: '4',
      name: 'Monthly Revenue Report',
      description: 'Automated monthly financial reporting canvas',
      createdAt: '1 week ago',
      updatedAt: '2 days ago',
      createdBy: {
        name: 'Emily Rodriguez',
        initials: 'ER',
      },
      status: 'published',
      type: 'report'
    },
    {
      id: '5',
      name: 'Inventory Management',
      description: 'Product inventory tracking and management system',
      createdAt: '2 weeks ago',
      updatedAt: '1 week ago',
      createdBy: {
        name: 'David Kim',
        initials: 'DK',
      },
      status: 'archived',
      type: 'canvas'
    },
    {
      id: '6',
      name: 'Team Performance Dashboard',
      description: 'Track team productivity and project progress',
      createdAt: '3 weeks ago',
      updatedAt: '5 days ago',
      createdBy: {
        name: 'John Doe',
        initials: 'JD',
        avatar: 'https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?w=64&h=64&fit=crop&crop=face',
      },
      status: 'draft',
      type: 'canvas'
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

  // Navigation links configuration
  const navigationLinks: NavigationLink[] = [
    {
      id: 'canvases',
      label: 'Canvases',
      icon: <MaterialSymbol size='lg' opticalSize={20} weight={400} name="automation" />,
      isActive: true,
      tooltip: 'Canvases'
    }
  ]

  const handleLinkClick = (linkId: string) => {
    console.log(`Navigation link clicked: ${linkId}`)
  }

  const getCanvasIcon = (type: Canvas['type']) => {
    switch (type) {
      case 'canvas':
        return <MaterialSymbol name="workflow" className="text-blue-600" />
      case 'dashboard':
        return <MaterialSymbol name="dashboard" className="text-green-600" />
      case 'form':
        return <MaterialSymbol name="description" className="text-purple-600" />
      case 'report':
        return <MaterialSymbol name="analytics" className="text-orange-600" />
      default:
        return <MaterialSymbol name="canvas" className="text-blue-600" />
    }
  }

  const getStatusBadge = (status: Canvas['status']) => {
    switch (status) {
      case 'published':
        return <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-purple-100 text-purple-800 dark:bg-purple-900/20 dark:text-purple-400">Published</span>
      case 'draft':
        return <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-orange-100 text-orange-800 dark:bg-orange-900/20 dark:text-orange-400">Draft</span>
      case 'archived':
        return <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-zinc-100 text-zinc-800 dark:bg-zinc-800 dark:text-zinc-400">Archived</span>
    }
  }

  // Filter canvases based on search and status
  const filteredCanvases = canvases.filter(canvas => {
    const matchesSearch = canvas.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
                         canvas.description?.toLowerCase().includes(searchQuery.toLowerCase())
    const matchesStatus = filterStatus === 'all' || canvas.status === filterStatus
    return matchesSearch && matchesStatus
  })

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
          {/* Page Header */}
          <div className="mb-8">
            <Heading level={1} className="!text-3xl mb-2">Canvases</Heading>
            <Text className="text-lg text-zinc-600 dark:text-zinc-400">
              Create and manage your interactive canvases
            </Text>
          </div>

          {/* Actions and Filters */}
          <div className="flex flex-col sm:flex-row gap-4 mb-6">
            {/* Search */}
            <div className="flex-1">
              <div className="relative">
                <MaterialSymbol name="search" className="absolute left-3 top-1/2 transform -translate-y-1/2 text-zinc-400" />
                <input
                  type="text"
                  placeholder="Search canvases..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="w-full pl-10 pr-4 py-2 border border-zinc-200 dark:border-zinc-700 rounded-lg bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 placeholder-zinc-500 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
              </div>
            </div>

            {/* Status Filter */}
            <div className="flex items-center gap-2">
              <Text className="text-sm font-medium text-zinc-700 dark:text-zinc-300">Status:</Text>
              <select
                value={filterStatus}
                onChange={(e) => setFilterStatus(e.target.value as typeof filterStatus)}
                className="px-3 py-2 border border-zinc-200 dark:border-zinc-700 rounded-lg bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              >
                <option value="all">All</option>
                <option value="published">Published</option>
                <option value="draft">Draft</option>
                <option value="archived">Archived</option>
              </select>
            </div>

            {/* Create Canvas Button */}
            <Button color="blue">
              <MaterialSymbol name="add" className="mr-2" />
              New Canvas
            </Button>
          </div>

          {/* Canvases Grid */}
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {filteredCanvases.map((canvas) => (
              <div key={canvas.id} className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 hover:shadow-md transition-shadow group">
                <div className="p-6 flex flex-col justify-between h-full">
                  <div>
                    {/* Header */}
                    <div className="flex items-start justify-between mb-4">
                      <div>
                        <Link href={`/canvas/${canvas.id}`} className="block">
                          <Heading level={3} className="text-lg font-semibold text-zinc-900 dark:text-white hover:text-blue-600 dark:hover:text-blue-400 transition-colors !leading-7">
                            {canvas.name} 
                          </Heading>
                        </Link>
                      </div>
                      <div className='flex items-center gap-2'>
                        {getStatusBadge(canvas.status)}
                        <Button plain className="text-zinc-400 hover:text-zinc-600 opacity-100 group-hover:opacity-100 transition-opacity">
                          <MaterialSymbol name="more_vert" />
                        </Button>
                      </div>
                    </div>

                  {/* Content */}
                  <div className="mb-4">
                    
                    {canvas.description && (
                      <Text className="text-sm text-zinc-600 dark:text-zinc-400 mt-1 line-clamp-2">
                        {canvas.description}
                      </Text>
                    )}
                  </div>
                  </div>
                  {/* Footer */}
                  <div className="space-y-2">
                    <div className="flex items-center justify-between text-xs text-zinc-500">
                      <span>Created {canvas.createdAt}</span>
                      <span>Updated {canvas.updatedAt}</span>
                    </div>
                    <div className="flex items-center space-x-2">
                      <Avatar
                        src={canvas.createdBy.avatar}
                        initials={canvas.createdBy.initials}
                        alt={canvas.createdBy.name}
                        className="w-5 h-5"
                      />
                      <Text className="text-xs text-zinc-600 dark:text-zinc-400">
                        by {canvas.createdBy.name}
                      </Text>
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>

          {/* Empty State */}
          {filteredCanvases.length === 0 && (
            <div className="text-center py-12">
              <MaterialSymbol name="canvas" className="mx-auto text-zinc-400 mb-4" size="xl" />
              <Heading level={3} className="text-lg text-zinc-900 dark:text-white mb-2">
                {searchQuery || filterStatus !== 'all' ? 'No canvases found' : 'No canvases yet'}
              </Heading>
              <Text className="text-zinc-600 dark:text-zinc-400 mb-6">
                {searchQuery || filterStatus !== 'all' 
                  ? 'Try adjusting your search or filter criteria.' 
                  : 'Get started by creating your first canvas.'}
              </Text>
              {(!searchQuery && filterStatus === 'all') && (
                <Button color="blue" className='flex items-center gap-2'>
                  <MaterialSymbol name="add" />
                  Create Canvas
                </Button>
              )}
            </div>
          )}
        </div>
      </main>
    </div>
  )
}