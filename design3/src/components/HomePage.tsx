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
import { Link } from './lib/Link/link'
import { Divider } from './lib/Divider/divider'
import Tippy from '@tippyjs/react'

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
          <nav className='w-18 h-full bg-zinc-50 dark:bg-zinc-950 border-r border-zinc-200 dark:border-zinc-800 flex flex-col flex-grow-1'>
                {/* Top Section - Logo */}
              
          
              
                  <div className="flex-shrink-0 flex flex-col items-center text-center">
                  <div className='flex flex-col items-center py-2 mb-2 mt-2'>
                     <Link href='/'>  
                      <div className='text-zinc-700 hover:text-zinc-800 bg-blue-100 dark:bg-zinc-800 hover:bg-zinc-100 dark:text-zinc-400 dark:hover:text-zinc-300 dark:hover:bg-zinc-800 w-8 h-8 rounded-md flex items-center justify-center'>
                        <MaterialSymbol fill={1} name="home" size='lg'/>
                      </div>
                      <span className="text-xs block text-zinc-700 dark:text-zinc-300">Home</span>
                    </Link>
                    
              
                  </div>
                    <div className='flex flex-col items-center py-2 rounded-md'>
                    <Link href='/canvases' className='text-zinc-700 hover:text-zinc-800 hover:bg-blue-200 dark:text-zinc-400 dark:hover:text-zinc-300 dark:hover:bg-zinc-800 w-8 h-8 block rounded-md flex items-center justify-center'>
                        <MaterialSymbol  name="automation" size='lg'/>
                      </Link>
                      <span className="text-xs block text-zinc-700 dark:text-zinc-300">Canvases</span>
                    
              
                  </div>
                  <Divider className='my-4'/>
                  <Tippy placement="right" className="text-xs bg-zinc-900 dark:bg-zinc-800 text-zinc-100 dark:text-zinc-200 p-2 rounded-md" content="Create new canvas">
                    <Button plain>
                      <MaterialSymbol name="add" size='lg'/>
                    </Button>
                  </Tippy>
                  </div>
                 
          
                {/* Middle Section - Spacer */}
                <div className="flex-1" />
                
                
              
          
              
          
                      
          
                  
          </nav>
          <div className='bg-zinc-50 dark:bg-zinc-900 w-full flex-grow-1 p-6'>
                    TODO
          </div>
        </div>
      </main>
    </div>
  )
}