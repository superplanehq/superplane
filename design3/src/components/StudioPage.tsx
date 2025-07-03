import { useState } from 'react'
import { Button } from './lib/Button/button'
import { Badge } from './lib/Badge/badge'
import { Avatar } from './lib/Avatar/avatar'
import { Text, TextLink } from './lib/Text/text'
import { Heading, Subheading } from './lib/Heading/heading'
import { Navigation, type User, type Organization } from './lib/Navigation/navigation'
import clsx from 'clsx'

interface StudioPageProps {
  onSignOut?: () => void
}

interface Workflow {
  id: string
  name: string
  description: string
  status: 'active' | 'paused' | 'draft' | 'error'
  lastRun: string
  runs: number
  category: 'automation' | 'integration' | 'deployment' | 'monitoring'
  tags: string[]
}

interface Template {
  id: string
  name: string
  description: string
  category: 'automation' | 'integration' | 'deployment' | 'monitoring'
  difficulty: 'beginner' | 'intermediate' | 'advanced'
  icon: string
  estimatedTime: string
}

type StudioSection = 'workflows' | 'builder' | 'templates' | 'integrations'

export function StudioPage({ onSignOut }: StudioPageProps) {
  const [activeSection, setActiveSection] = useState<StudioSection>('workflows')

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

  // Mock workflows data
  const workflows: Workflow[] = [
    {
      id: '1',
      name: 'Deploy to Production',
      description: 'Automated deployment pipeline for the main application',
      status: 'active',
      lastRun: '2 hours ago',
      runs: 45,
      category: 'deployment',
      tags: ['CI/CD', 'Production', 'Docker']
    },
    {
      id: '2',
      name: 'Data Sync Process',
      description: 'Synchronize customer data between CRM and analytics platform',
      status: 'active',
      lastRun: '15 minutes ago',
      runs: 128,
      category: 'integration',
      tags: ['CRM', 'Analytics', 'ETL']
    },
    {
      id: '3',
      name: 'Security Vulnerability Scan',
      description: 'Weekly automated security assessment of all repositories',
      status: 'paused',
      lastRun: '3 days ago',
      runs: 23,
      category: 'monitoring',
      tags: ['Security', 'SAST', 'Compliance']
    },
    {
      id: '4',
      name: 'Backup Automation',
      description: 'Daily backup of production databases with rotation',
      status: 'error',
      lastRun: '1 day ago',
      runs: 67,
      category: 'automation',
      tags: ['Backup', 'Database', 'AWS']
    },
    {
      id: '5',
      name: 'Notification System',
      description: 'Send alerts to team channels based on system events',
      status: 'draft',
      lastRun: 'Never',
      runs: 0,
      category: 'automation',
      tags: ['Notifications', 'Slack', 'Alerts']
    }
  ]

  // Mock templates data
  const templates: Template[] = [
    {
      id: '1',
      name: 'CI/CD Pipeline',
      description: 'Complete continuous integration and deployment setup',
      category: 'deployment',
      difficulty: 'intermediate',
      icon: '🚀',
      estimatedTime: '30 min'
    },
    {
      id: '2',
      name: 'Data Pipeline',
      description: 'Extract, transform, and load data between systems',
      category: 'integration',
      difficulty: 'advanced',
      icon: '📊',
      estimatedTime: '45 min'
    },
    {
      id: '3',
      name: 'Monitoring Setup',
      description: 'Set up comprehensive system monitoring and alerting',
      category: 'monitoring',
      difficulty: 'beginner',
      icon: '📈',
      estimatedTime: '20 min'
    },
    {
      id: '4',
      name: 'Slack Bot',
      description: 'Create an automated Slack bot for team notifications',
      category: 'automation',
      difficulty: 'beginner',
      icon: '🤖',
      estimatedTime: '15 min'
    },
    {
      id: '5',
      name: 'API Integration',
      description: 'Connect and sync data between REST APIs',
      category: 'integration',
      difficulty: 'intermediate',
      icon: '🔗',
      estimatedTime: '25 min'
    },
    {
      id: '6',
      name: 'Infrastructure as Code',
      description: 'Manage cloud infrastructure with Terraform',
      category: 'deployment',
      difficulty: 'advanced',
      icon: '☁️',
      estimatedTime: '60 min'
    }
  ]

  // Sidebar navigation items
  const sidebarItems = [
    {
      id: 'workflows',
      label: 'My Workflows',
      icon: (
        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
        </svg>
      ),
      count: workflows.length,
    },
    {
      id: 'builder',
      label: 'Workflow Builder',
      icon: (
        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19.428 15.428a2 2 0 00-1.022-.547l-2.387-.477a6 6 0 00-3.86.517l-.318.158a6 6 0 01-3.86.517L6.05 15.21a2 2 0 00-1.806.547M8 4h8l-1 1v5.172a2 2 0 00.586 1.414l5 5c1.26 1.26.367 3.414-1.415 3.414H4.828c-1.782 0-2.674-2.154-1.414-3.414l5-5A2 2 0 009 10.172V5L8 4z" />
        </svg>
      ),
    },
    {
      id: 'templates',
      label: 'Templates',
      icon: (
        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
        </svg>
      ),
      count: templates.length,
    },
    {
      id: 'integrations',
      label: 'Integrations',
      icon: (
        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v14a2 2 0 002 2z" />
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
    console.log(`Organization action: ${action}`)
  }

  const getStatusBadge = (status: Workflow['status']) => {
    switch (status) {
      case 'active':
        return <Badge color="green">Active</Badge>
      case 'paused':
        return <Badge color="yellow">Paused</Badge>
      case 'draft':
        return <Badge color="zinc">Draft</Badge>
      case 'error':
        return <Badge color="red">Error</Badge>
    }
  }

  const getCategoryBadge = (category: Workflow['category']) => {
    const colors = {
      automation: 'blue',
      integration: 'purple',
      deployment: 'green',
      monitoring: 'orange'
    } as const

    return <Badge color={colors[category]}>{category}</Badge>
  }

  const getDifficultyBadge = (difficulty: Template['difficulty']) => {
    const colors = {
      beginner: 'green',
      intermediate: 'yellow',
      advanced: 'red'
    } as const

    return <Badge color={colors[difficulty]}>{difficulty}</Badge>
  }

  const renderWorkflowsContent = () => (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <Heading level={1} className="text-3xl mb-2">My Workflows</Heading>
          <Text className="text-lg text-zinc-600 dark:text-zinc-400">
            Build, manage, and monitor your automated workflows
          </Text>
        </div>
        <div className="flex items-center space-x-3">
          <Button outline>
            <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
            </svg>
            Import
          </Button>
          <Button color="blue" onClick={() => setActiveSection('builder')}>
            <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
            Create Workflow
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-4">
        {workflows.map((workflow) => (
          <div key={workflow.id} className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700 hover:shadow-md transition-shadow">
            <div className="flex items-start justify-between">
              <div className="flex-1">
                <div className="flex items-center space-x-3 mb-2">
                  <Subheading level={3} className="text-lg">{workflow.name}</Subheading>
                  {getStatusBadge(workflow.status)}
                  {getCategoryBadge(workflow.category)}
                </div>
                <Text className="text-sm text-zinc-600 dark:text-zinc-400 mb-3">
                  {workflow.description}
                </Text>
                <div className="flex items-center space-x-4 text-xs text-zinc-500 mb-3">
                  <span>Last run: {workflow.lastRun}</span>
                  <span>•</span>
                  <span>{workflow.runs} total runs</span>
                </div>
                <div className="flex flex-wrap gap-1">
                  {workflow.tags.map((tag) => (
                    <Badge key={tag} color="zinc" className="text-xs">
                      {tag}
                    </Badge>
                  ))}
                </div>
              </div>
              <div className="flex items-center space-x-2">
                <Button outline>Edit</Button>
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

  const renderBuilderContent = () => (
    <div className="space-y-6">
      <div>
        <Heading level={1} className="text-3xl mb-2">Workflow Builder</Heading>
        <Text className="text-lg text-zinc-600 dark:text-zinc-400">
          Visual workflow builder for creating automated processes
        </Text>
      </div>

      {/* Builder Canvas Placeholder */}
      <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 min-h-[600px] flex items-center justify-center">
        <div className="text-center space-y-4">
          <div className="w-24 h-24 mx-auto bg-zinc-100 dark:bg-zinc-700 rounded-full flex items-center justify-center">
            <svg className="w-12 h-12 text-zinc-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19.428 15.428a2 2 0 00-1.022-.547l-2.387-.477a6 6 0 00-3.86.517l-.318.158a6 6 0 01-3.86.517L6.05 15.21a2 2 0 00-1.806.547M8 4h8l-1 1v5.172a2 2 0 00.586 1.414l5 5c1.26 1.26.367 3.414-1.415 3.414H4.828c-1.782 0-2.674-2.154-1.414-3.414l5-5A2 2 0 009 10.172V5L8 4z" />
            </svg>
          </div>
          <div>
            <Heading level={2} className="text-xl mb-2">Visual Workflow Builder</Heading>
            <Text className="text-zinc-500 max-w-md">
              Drag and drop components to build your workflow. Connect triggers, actions, and conditions to create powerful automations.
            </Text>
          </div>
          <div className="flex items-center justify-center space-x-3">
            <Button color="blue">Start Building</Button>
            <Button outline onClick={() => setActiveSection('templates')}>Browse Templates</Button>
          </div>
        </div>
      </div>

      {/* Quick Tools */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700">
          <Subheading level={3} className="mb-3">Triggers</Subheading>
          <div className="space-y-2">
            <div className="flex items-center space-x-2 p-2 bg-zinc-50 dark:bg-zinc-700 rounded">
              <div className="w-2 h-2 bg-green-500 rounded-full"></div>
              <Text className="text-sm">Webhook</Text>
            </div>
            <div className="flex items-center space-x-2 p-2 bg-zinc-50 dark:bg-zinc-700 rounded">
              <div className="w-2 h-2 bg-blue-500 rounded-full"></div>
              <Text className="text-sm">Schedule</Text>
            </div>
            <div className="flex items-center space-x-2 p-2 bg-zinc-50 dark:bg-zinc-700 rounded">
              <div className="w-2 h-2 bg-purple-500 rounded-full"></div>
              <Text className="text-sm">File Change</Text>
            </div>
          </div>
        </div>

        <div className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700">
          <Subheading level={3} className="mb-3">Actions</Subheading>
          <div className="space-y-2">
            <div className="flex items-center space-x-2 p-2 bg-zinc-50 dark:bg-zinc-700 rounded">
              <div className="w-2 h-2 bg-orange-500 rounded-full"></div>
              <Text className="text-sm">HTTP Request</Text>
            </div>
            <div className="flex items-center space-x-2 p-2 bg-zinc-50 dark:bg-zinc-700 rounded">
              <div className="w-2 h-2 bg-red-500 rounded-full"></div>
              <Text className="text-sm">Send Email</Text>
            </div>
            <div className="flex items-center space-x-2 p-2 bg-zinc-50 dark:bg-zinc-700 rounded">
              <div className="w-2 h-2 bg-yellow-500 rounded-full"></div>
              <Text className="text-sm">Database Query</Text>
            </div>
          </div>
        </div>

        <div className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700">
          <Subheading level={3} className="mb-3">Conditions</Subheading>
          <div className="space-y-2">
            <div className="flex items-center space-x-2 p-2 bg-zinc-50 dark:bg-zinc-700 rounded">
              <div className="w-2 h-2 bg-indigo-500 rounded-full"></div>
              <Text className="text-sm">If/Else</Text>
            </div>
            <div className="flex items-center space-x-2 p-2 bg-zinc-50 dark:bg-zinc-700 rounded">
              <div className="w-2 h-2 bg-pink-500 rounded-full"></div>
              <Text className="text-sm">Filter</Text>
            </div>
            <div className="flex items-center space-x-2 p-2 bg-zinc-50 dark:bg-zinc-700 rounded">
              <div className="w-2 h-2 bg-teal-500 rounded-full"></div>
              <Text className="text-sm">Loop</Text>
            </div>
          </div>
        </div>
      </div>
    </div>
  )

  const renderTemplatesContent = () => (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <Heading level={1} className="text-3xl mb-2">Templates</Heading>
          <Text className="text-lg text-zinc-600 dark:text-zinc-400">
            Pre-built workflows to get you started quickly
          </Text>
        </div>
        <Button outline>
          <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
          </svg>
          Browse Community
        </Button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {templates.map((template) => (
          <div key={template.id} className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700 hover:shadow-md transition-shadow">
            <div className="flex items-start justify-between mb-4">
              <div className="text-3xl">{template.icon}</div>
              <div className="flex items-center space-x-2">
                {getDifficultyBadge(template.difficulty)}
                {getCategoryBadge(template.category)}
              </div>
            </div>
            <Subheading level={3} className="text-lg mb-2">{template.name}</Subheading>
            <Text className="text-sm text-zinc-600 dark:text-zinc-400 mb-4">
              {template.description}
            </Text>
            <div className="flex items-center justify-between">
              <Text className="text-xs text-zinc-500">
                ~{template.estimatedTime}
              </Text>
              <Button color="blue">Use Template</Button>
            </div>
          </div>
        ))}
      </div>
    </div>
  )

  const renderIntegrationsContent = () => (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <Heading level={1} className="text-3xl mb-2">Integrations</Heading>
          <Text className="text-lg text-zinc-600 dark:text-zinc-400">
            Connect with your favorite tools and services
          </Text>
        </div>
        <Button color="blue">
          <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
          Add Integration
        </Button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        {[
          { name: 'GitHub', description: 'Git repository management', icon: '🐙', connected: true },
          { name: 'Slack', description: 'Team communication', icon: '💬', connected: true },
          { name: 'AWS', description: 'Cloud infrastructure', icon: '☁️', connected: false },
          { name: 'Docker', description: 'Container platform', icon: '🐳', connected: true },
          { name: 'Kubernetes', description: 'Container orchestration', icon: '⚙️', connected: false },
          { name: 'MongoDB', description: 'NoSQL database', icon: '🍃', connected: false },
          { name: 'PostgreSQL', description: 'Relational database', icon: '🐘', connected: true },
          { name: 'Redis', description: 'In-memory data store', icon: '🔴', connected: false },
        ].map((integration) => (
          <div key={integration.name} className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700">
            <div className="text-center">
              <div className="text-4xl mb-3">{integration.icon}</div>
              <Subheading level={3} className="text-lg mb-2">{integration.name}</Subheading>
              <Text className="text-sm text-zinc-600 dark:text-zinc-400 mb-4">
                {integration.description}
              </Text>
              {integration.connected ? (
                <Badge color="green" className="mb-3">Connected</Badge>
              ) : (
                <Button outline className="w-full">Connect</Button>
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  )

  const renderContent = () => {
    switch (activeSection) {
      case 'workflows':
        return renderWorkflowsContent()
      case 'builder':
        return renderBuilderContent()
      case 'templates':
        return renderTemplatesContent()
      case 'integrations':
        return renderIntegrationsContent()
      default:
        return renderWorkflowsContent()
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
                  onClick={() => setActiveSection(item.id as StudioSection)}
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