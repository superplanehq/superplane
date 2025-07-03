import { useState } from 'react'
import { Button } from './lib/Button/button'
import { Avatar } from './lib/Avatar/avatar'
import { Text } from './lib/Text/text'
import { Heading } from './lib/Heading/heading'
import { NavigationVertical, type User, type Organization, type NavigationLink } from './lib/Navigation/navigation-vertical'
import { MaterialSymbol } from './lib/MaterialSymbol/material-symbol'

interface CanvasesPageProps {
  onSignOut?: () => void
  navigationLinks?: NavigationLink[]
  onLinkClick?: (linkId: string) => void
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
  type: 'canvas'
}

export function CanvasesPage({ onSignOut, navigationLinks = [], onLinkClick }: CanvasesPageProps) {
  const [searchQuery, setSearchQuery] = useState('')
  const [filterStatus, setFilterStatus] = useState<'all' | 'draft' | 'published' | 'archived'>('all')
  const [viewMode, setViewMode] = useState<'grid' | 'list'>('grid')
  const showIcons = false

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
      type: 'canvas'
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
      type: 'canvas'
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

  // Navigation link click handler
  const handleLinkClick = (linkId: string) => {
    if (onLinkClick) {
      onLinkClick(linkId)
    } else {
      console.log(`Navigation link clicked: ${linkId}`)
    }
  }

  const getCanvasIcon = () => {
    return <MaterialSymbol name="automation" size='md' weight={400} className="text-blue-600" />
  }

  const getStatusBadge = (status: Canvas['status']) => {
    switch (status) {
      case 'published':
        return <span className="inline-flex items-center px-2.5 py-0.5 rounded-md text-xs font-medium bg-purple-100 text-purple-800 dark:bg-purple-900/20 dark:text-purple-400">Published</span>
      case 'draft':
        return <span className="inline-flex items-center px-2.5 py-0.5 rounded-md text-xs font-medium bg-orange-100 text-orange-800 dark:bg-orange-900/20 dark:text-orange-400">Draft</span>
      case 'archived':
        return <span className="inline-flex items-center px-2.5 py-0.5 rounded-md text-xs font-medium bg-zinc-100 text-zinc-800 dark:bg-zinc-800 dark:text-zinc-400">Archived</span>
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

            {/* View Mode Toggle */}
            <div className="flex items-center border border-zinc-200 dark:border-zinc-700 rounded-lg overflow-hidden">
              <button
                onClick={() => setViewMode('grid')}
                className={`p-2 transition-colors ${
                  viewMode === 'grid'
                    ? 'bg-blue-50 text-blue-600 dark:bg-blue-900/20 dark:text-blue-400'
                    : 'text-zinc-500 hover:text-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-300'
                }`}
                title="Grid view"
              >
                <MaterialSymbol name="grid_view" />
              </button>
              <button
                onClick={() => setViewMode('list')}
                className={`p-2 transition-colors ${
                  viewMode === 'list'
                    ? 'bg-blue-50 text-blue-600 dark:bg-blue-900/20 dark:text-blue-400'
                    : 'text-zinc-500 hover:text-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-300'
                }`}
                title="List view"
              >
                <MaterialSymbol name="view_list" />
              </button>
            </div>

            

            {/* Create Canvas Button */}
            <Button className='flex items-center bg-blue-700 text-white hover:bg-blue-600'>
              <MaterialSymbol name="add" className="mr-2" />
              New Canvas
            </Button>
          </div>

          {/* Canvases Display */}
          {viewMode === 'grid' ? (
            /* Grid View */
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
              {filteredCanvases.map((canvas) => (
                <div key={canvas.id} className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 hover:shadow-md transition-shadow group">
                  <div className="p-6 flex flex-col justify-between h-full">
                    <div>
                      {/* Header */}
                      <div className="flex items-start mb-4">
                        <div className="flex items-start justify-between space-x-3 flex-1">
                          {showIcons && (
                            <div className="p-1 bg-zinc-100 dark:bg-zinc-700 rounded-md h-8 w-8 text-center">
                              {getCanvasIcon()}
                            </div>
                          )}
                          <div className='flex flex-col'>
                            <button
                              onClick={() => {
                                window.history.pushState(null, '', `/canvas/${canvas.id}`)
                                window.dispatchEvent(new PopStateEvent('popstate'))
                              }}
                              className="block text-left w-full"
                            >
                              <Heading level={3} className="!text-md font-semibold text-zinc-900 dark:text-white hover:text-blue-600 dark:hover:text-blue-400 transition-colors mb-0 !leading-6">
                                {canvas.name}
                              </Heading>
                            </button>
                            <div>
                              {getStatusBadge(canvas.status)}
                            </div>
                          </div>
                          <Button plain className="text-zinc-400 hover:text-zinc-600 opacity-100 group-hover:opacity-100 transition-opacity">
                            <MaterialSymbol name="more_vert" />
                          </Button>
                        </div>
                        
                      </div>

                      {/* Content */}
                      <div className="mb-4">
                        
                        {canvas.description && (
                          <Text className="text-sm text-zinc-600 dark:text-zinc-400 line-clamp-2 mt-2">
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
          ) : (
            /* List View */
            <div className="space-y-3">
              {filteredCanvases.map((canvas) => (
                <div key={canvas.id} className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 hover:shadow-sm transition-shadow group">
                  <div className="p-4">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center space-x-4 flex-1">
                        {/* Icon */}
                        {showIcons && (
                          <div className="p-2 bg-zinc-100 dark:bg-zinc-700 rounded-lg flex-shrink-0">
                            {getCanvasIcon()}
                          </div>
                        )}
                        
                        {/* Content */}
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center space-x-3 mb-1">
                            <button
                              onClick={() => {
                                window.history.pushState(null, '', `/canvas/${canvas.id}`)
                                window.dispatchEvent(new PopStateEvent('popstate'))
                              }}
                              className="block text-left"
                            >
                              <Heading level={3} className="text-base font-semibold text-zinc-900 dark:text-white hover:text-blue-600 dark:hover:text-blue-400 transition-colors truncate">
                                {canvas.name}
                              </Heading>
                            </button>
                            {getStatusBadge(canvas.status)}
                          </div>
                          
                          {canvas.description && (
                            <Text className="text-sm text-zinc-600 dark:text-zinc-400 mb-2 line-clamp-1">
                              {canvas.description}
                            </Text>
                          )}
                          
                          <div className="flex items-center space-x-4 text-xs text-zinc-500">
                            <span>Created {canvas.createdAt}</span>
                            <span>•</span>
                            <span>Updated {canvas.updatedAt}</span>
                            <span>•</span>
                            <div className="flex items-center space-x-2">
                              <Avatar
                                src={canvas.createdBy.avatar}
                                initials={canvas.createdBy.initials}
                                alt={canvas.createdBy.name}
                                className="w-4 h-4"
                              />
                              <span>by {canvas.createdBy.name}</span>
                            </div>
                          </div>
                        </div>
                      </div>
                      
                      {/* Actions */}
                      <div className="flex items-center space-x-2 flex-shrink-0">
                        <Button plain className="text-zinc-400 hover:text-zinc-600 opacity-0 group-hover:opacity-100 transition-opacity">
                          <MaterialSymbol name="more_vert" />
                        </Button>
                      </div>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}

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