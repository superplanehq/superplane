import { NavigationVertical, type User, type Organization, type NavigationLink } from './lib/Navigation/navigation-vertical'
import { Subheading } from './lib/Heading/heading'
import { Text } from './lib/Text/text'

interface HomePageProps {
  onSignOut?: () => void
  navigationLinks?: NavigationLink[]
  onLinkClick?: (linkId: string) => void
  onConfigurationClick?: () => void
}

export function HomePage({ 
  onSignOut, 
  navigationLinks = [], 
  onLinkClick,
  onConfigurationClick 
}: HomePageProps) {
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

  const handleLinkClick = (linkId: string) => {
    if (onLinkClick) {
      onLinkClick(linkId)
    } else {
      console.log(`Navigation link clicked: ${linkId}`)
    }
  }

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
        onConfigurationClick={onConfigurationClick}
      />

      {/* Main Content */}
      <main className="flex-1 p-8">
        <div className="max-w-4xl mx-auto">
          <div className="text-center py-12">
            <Subheading level={2} className="mb-4">Welcome to SuperPlane</Subheading>
            <Text className="text-lg text-zinc-600 dark:text-zinc-400 mb-8">
              Your workflow automation platform
            </Text>
            
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 mt-12">
              <div className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700">
                <Subheading level={3} className="mb-2">Canvases</Subheading>
                <Text className="text-zinc-600 dark:text-zinc-400">
                  Create and manage your workflow canvases
                </Text>
              </div>
              
              <div className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700">
                <Subheading level={3} className="mb-2">Automations</Subheading>
                <Text className="text-zinc-600 dark:text-zinc-400">
                  Set up automated processes and workflows
                </Text>
              </div>
              
              <div className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700">
                <Subheading level={3} className="mb-2">Analytics</Subheading>
                <Text className="text-zinc-600 dark:text-zinc-400">
                  Monitor and analyze your workflow performance
                </Text>
              </div>
            </div>
          </div>
        </div>
      </main>
    </div>
  )
}