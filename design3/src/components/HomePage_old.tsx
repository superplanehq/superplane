import { useState } from 'react'
import { NavigationOrg, type User, type Organization } from './lib/Navigation/navigation-org'
import { Heading, Subheading } from './lib/Heading/heading'
import { Text } from './lib/Text/text'
import { Button } from './lib/Button/button'
import { MaterialSymbol } from './lib/MaterialSymbol/material-symbol'
import { Avatar } from './lib/Avatar/avatar'
import { Dropdown, DropdownButton, DropdownItem, DropdownMenu } from './lib/Dropdown/dropdown'
import { Checkbox, CheckboxField } from './lib/Checkbox/checkbox'
import { Input, InputGroup } from './lib/Input/input'
import { Label } from './lib/Fieldset/fieldset'

interface HomePageProps {
  onSignOut?: () => void
  onLinkClick?: (linkId: string) => void
}

export function HomePage({ 
  onSignOut, 
  onLinkClick
}: HomePageProps) {
  const [searchQuery, setSearchQuery] = useState('')
  const [filterStatus, setFilterStatus] = useState<'all' | 'draft' | 'published' | 'archived'>('all')
  const [viewMode, setViewMode] = useState<'grid' | 'list'>('grid')
  const showIcons = false
  const [currentPage, setCurrentPage] = useState('home')


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
    name: 'Confluent',
    initials: 'C',
  }

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
    if (action === 'settings') {
      // Navigate to settings page
      window.history.pushState(null, '', '/settings')
      window.dispatchEvent(new PopStateEvent('popstate'))
    } else {
      console.log(`Organization action: ${action}`)
    }
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
    <div className="min-h-screen flex flex-col bg-zinc-50 dark:bg-zinc-900">
      {/* Navigation */}
      <NavigationOrg
        user={currentUser}
        organization={currentOrganization}
        onHelpClick={handleHelpClick}
        onUserMenuAction={handleUserMenuAction}
        onOrganizationMenuAction={handleOrganizationMenuAction}
      />

      {/* Main Content */}
      <main className="w-full h-full flex flex-column flex-grow-1">
        <div className='flex flex-row flex-grow-1 justify-items-between'>
          <nav className='w-18 h-full bg-zinc-50 dark:bg-zinc-900 border-r border-zinc-200 dark:border-zinc-800 flex flex-col flex-grow-1'>
                {/* Top Section - Logo */}
              
          
              
                  <div className="flex-shrink-0 flex flex-col items-center text-center">
                  <div className='flex flex-col items-center py-2 mb-2 mt-2'>
                      <div className='text-zinc-700 hover:text-zinc-800 hover:bg-zinc-100 dark:text-zinc-400 dark:hover:text-zinc-300 dark:hover:bg-zinc-800 w-7 h-7 block rounded-md'>
                        <MaterialSymbol name="home" size='lg'/>
                      </div>
                      <span className="text-xs block">Home</span>
                    
              
                  </div>
                    <div className='flex flex-col items-center py-2 rounded-md'>
                      <div className='text-zinc-700 hover:text-zinc-800 bg-blue-100 dark:bg-zinc-800 hover:bg-zinc-100 dark:text-zinc-400 dark:hover:text-zinc-300 dark:hover:bg-zinc-800 w-8 h-8 rounded-md flex items-center justify-center'>
                        <MaterialSymbol name="automation" size='lg'/>
                      </div>
                      <span className="text-xs block">Canvases</span>
                    
              
                  </div>
                  </div>
                
          
                {/* Middle Section - Spacer */}
                <div className="flex-1" />
                
                
              
          
              
          
                      
          
                  
          </nav>
          <div className='bg-zinc-50 dark:bg-zinc-900 w-full flex-grow-1 p-6'>
            <div className="p-4">
                      {/* Page Header */}
                      <div className='flex items-center justify-between mb-8'>
                          <Heading level={1} className="!text-3xl mb-2">Canvases</Heading>
                          <Text className="text-lg text-zinc-600 dark:text-zinc-400 hidden">
                            Create and manage your interactive canvases
                          </Text>
                        <Button className='flex items-center bg-blue-700 text-white hover:bg-blue-600'>
                          <MaterialSymbol name="add" className="mr-2" />
                          New Canvas
                        </Button>
                      </div>
            
                      {/* Actions and Filters */}
                      <div className="flex flex-col sm:flex-row gap-4 mb-6 justify-between">
                        {/* Search */}
                        <div className='flex items-center gap-2'>
                          <div className="flex-1 w-100">
                            <div className="relative">
                              <MaterialSymbol name="search" className="absolute left-3 top-1/2 transform -translate-y-1/2 text-zinc-400" />
                              <input
                                type="text"
                                placeholder="Search canvases..."
                                value={searchQuery}
                                onChange={(e) => setSearchQuery(e.target.value)}
                                className="h-9 w-full pl-10 pr-4 py-2 border border-zinc-200 dark:border-zinc-700 rounded-lg bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 placeholder-zinc-500 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                              />
                             
                            </div>
                          </div>
                          <div className='hidden'>
                          <InputGroup className='flex items-center'>
                            <MaterialSymbol name="search" size='sm' className="text-zinc-400" />
                            <Input name="search" placeholder="Search&hellip;" aria-label="Search" />
                          </InputGroup>
                          </div>
                          {/* Status Filter */}
                          <div className="flex items-center gap-2">
                            <Dropdown>
                              <DropdownButton outline className='flex items-center'>
                                Status
                                <MaterialSymbol name="arrow_drop_down" />
                              </DropdownButton>
                              <DropdownMenu>
                                <DropdownItem href="/users/1">
                                <CheckboxField>
                                  <Checkbox name="status_archived" />
                                  <Label>Published</Label>
                                </CheckboxField>
                                </DropdownItem>
                                <DropdownItem href="/users/1">
                                <CheckboxField>
                                  <Checkbox name="status_archived" />
                                  <Label>Draft</Label>
                                </CheckboxField>
                                </DropdownItem>
                                <DropdownItem href="/users/1">
                                <CheckboxField>
                                  <Checkbox name="status_archived" />
                                  <Label>Archived</Label>
                                </CheckboxField>
                                </DropdownItem>
                              </DropdownMenu>
                            </Dropdown>
                          </div>
              
                          {/* Vi  ew Mode Toggle */}
                          
                        </div>
                        
            
                        {/* Create Canvas Button */}
                        <div className="flex items-center">
                            <Button
                              color='light'
                              onClick={() => setViewMode('grid')}
                              plain={
                                (viewMode === 'grid'
                                  ? false
                                  : true)
                              }
                              title="Grid view"
                            >
                              <MaterialSymbol name="grid_view" />
                            </Button>
                            <Button
                              color='light'
                              onClick={() => setViewMode('list')}
                              plain={viewMode === 'list' ? false : true}
                              title="List view"
                            >
                              <MaterialSymbol name="view_list" />
                            </Button>
                          </div>
                      </div>
            
                      {/* Canvases Display */}
                      {viewMode === 'grid' ? (
                        /* Grid View */
                        <div className="grid grid-cols-1 md:grid-cols-3 lg:grid-cols-5 gap-6">
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
                                          
                                        </div>
                                      </div>
                                      {getStatusBadge(canvas.status)}
                                      
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
                                <div className="flex justify-between items-center">
                                  <div className='flex items-center space-x-2'>
                                    <Avatar
                                      src={canvas.createdBy.avatar}
                                      initials={canvas.createdBy.initials}
                                      alt={canvas.createdBy.name}
                                      className="w-6 h-6 bg-blue-700 dark:bg-blue-900 text-blue-100 dark:text-blue-100"
                                    />
                                    <div className="text-zinc-500">
                                      <p className="text-xs text-zinc-600 dark:text-zinc-400 leading-none mb-1">
                                        Created by <strong>{canvas.createdBy.name}</strong>
                                      </p>
                                      <p className="text-xs text-zinc-600 dark:text-zinc-400 leading-none">
                                        Updated {canvas.updatedAt}
                                      </p>                                    
                                    </div>
                                  </div>
                                  <Button plain className="text-zinc-400 hover:text-zinc-600 opacity-100 group-hover:opacity-100 transition-opacity">
                                    <MaterialSymbol name="more_vert" />
                                  </Button>
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
          </div>
        </div>
      </main>
    </div>
  )
}