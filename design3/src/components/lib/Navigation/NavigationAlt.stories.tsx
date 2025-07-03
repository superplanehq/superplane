import type { Meta, StoryObj } from '@storybook/react'
import { NavigationAlt } from './navigation-alt'

const meta: Meta<typeof NavigationAlt> = {
  title: 'Components/NavigationAlt',
  component: NavigationAlt,
  parameters: {
    layout: 'fullscreen',
  },
  tags: ['autodocs'],
}

export default meta
type Story = StoryObj<typeof meta>

// Sample data
const sampleUser = {
  id: '1',
  name: 'John Doe',
  email: 'john@superplane.com',
  initials: 'JD',
  avatar: 'https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?w=64&h=64&fit=crop&crop=face',
}

const sampleUserWithoutAvatar = {
  id: '2',
  name: 'Sarah Wilson',
  email: 'sarah@superplane.com',
  initials: 'SW',
}

const sampleOrganization = {
  id: '1',
  name: 'Development Team',
  plan: 'Pro Plan',
  initials: 'DT',
}

const sampleOrganizationWithAvatar = {
  id: '2',
  name: 'Acme Corporation',
  plan: 'Enterprise',
  initials: 'AC',
  avatar: 'https://images.unsplash.com/photo-1560179707-f14e90ef3623?w=64&h=64&fit=crop&crop=center',
}

export const Default: Story = {
  args: {
    user: sampleUser,
    organization: sampleOrganization,
    onHelpClick: () => alert('Help clicked'),
    onUserMenuAction: (action) => alert(`User menu action: ${action}`),
    onOrganizationMenuAction: (action) => alert(`Organization menu action: ${action}`),
  },
}

export const WithoutUserAvatar: Story = {
  args: {
    user: sampleUserWithoutAvatar,
    organization: sampleOrganization,
    onHelpClick: () => alert('Help clicked'),
    onUserMenuAction: (action) => alert(`User menu action: ${action}`),
    onOrganizationMenuAction: (action) => alert(`Organization menu action: ${action}`),
  },
}

export const WithOrganizationAvatar: Story = {
  args: {
    user: sampleUser,
    organization: sampleOrganizationWithAvatar,
    onHelpClick: () => alert('Help clicked'),
    onUserMenuAction: (action) => alert(`User menu action: ${action}`),
    onOrganizationMenuAction: (action) => alert(`Organization menu action: ${action}`),
  },
}

export const LongNames: Story = {
  args: {
    user: {
      id: '3',
      name: 'Alexander Montgomery-Richardson',
      email: 'alexander.montgomery-richardson@verylongdomainname.com',
      initials: 'AM',
    },
    organization: {
      id: '3',
      name: 'International Software Development Corporation Ltd.',
      plan: 'Enterprise Plus',
      initials: 'IS',
    },
    onHelpClick: () => alert('Help clicked'),
    onUserMenuAction: (action) => alert(`User menu action: ${action}`),
    onOrganizationMenuAction: (action) => alert(`Organization menu action: ${action}`),
  },
}

export const WithoutPlan: Story = {
  args: {
    user: sampleUser,
    organization: {
      id: '4',
      name: 'Startup Team',
      initials: 'ST',
    },
    onHelpClick: () => alert('Help clicked'),
    onUserMenuAction: (action) => alert(`User menu action: ${action}`),
    onOrganizationMenuAction: (action) => alert(`Organization menu action: ${action}`),
  },
}

export const CompactNames: Story = {
  args: {
    user: {
      id: '5',
      name: 'Al Kim',
      email: 'al@acme.co',
      initials: 'AK',
    },
    organization: {
      id: '5',
      name: 'Acme',
      plan: 'Free',
      initials: 'A',
    },
    onHelpClick: () => alert('Help clicked'),
    onUserMenuAction: (action) => alert(`User menu action: ${action}`),
    onOrganizationMenuAction: (action) => alert(`Organization menu action: ${action}`),
  },
}

export const Interactive: Story = {
  render: () => {
    const handleHelpClick = () => {
      console.log('Help clicked - opening help documentation')
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
          console.log('Signing out user...')
          break
      }
    }

    const handleOrganizationMenuAction = (action: 'settings' | 'billing' | 'members') => {
      switch (action) {
        case 'settings':
          console.log('Opening organization settings...')
          break
        case 'billing':
          console.log('Opening billing and plans...')
          break
        case 'members':
          console.log('Managing team members...')
          break
      }
    }

    return (
      <div className="min-h-screen bg-zinc-50 dark:bg-zinc-900">
        <NavigationAlt
          user={sampleUser}
          organization={sampleOrganization}
          onHelpClick={handleHelpClick}
          onUserMenuAction={handleUserMenuAction}
          onOrganizationMenuAction={handleOrganizationMenuAction}
        />
        <div className="p-8">
          <h1 className="text-2xl font-bold text-zinc-900 dark:text-white mb-4">
            Alternative Navigation Component Demo
          </h1>
          <p className="text-zinc-600 dark:text-zinc-400 mb-4">
            This version features the paper airplane logo on the left with the organization name prominently displayed.
            Check the browser console for action logs.
          </p>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            <div className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700">
              <h3 className="font-semibold mb-2">Paper Airplane Logo</h3>
              <p className="text-sm text-zinc-600 dark:text-zinc-400">
                Blue circular logo with white paper airplane icon representing SuperPlane brand.
              </p>
            </div>
            <div className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700">
              <h3 className="font-semibold mb-2">Organization Prominence</h3>
              <p className="text-sm text-zinc-600 dark:text-zinc-400">
                Organization name displayed prominently next to logo with dropdown menu access.
              </p>
            </div>
            <div className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700">
              <h3 className="font-semibold mb-2">Right-side Controls</h3>
              <p className="text-sm text-zinc-600 dark:text-zinc-400">
                Help and user controls remain easily accessible on the right side.
              </p>
            </div>
          </div>
        </div>
      </div>
    )
  },
}

export const MobileView: Story = {
  args: {
    user: sampleUser,
    organization: sampleOrganization,
    onHelpClick: () => alert('Help clicked'),
    onUserMenuAction: (action) => alert(`User menu action: ${action}`),
    onOrganizationMenuAction: (action) => alert(`Organization menu action: ${action}`),
  },
  parameters: {
    viewport: {
      defaultViewport: 'mobile1',
    },
  },
}

export const DarkMode: Story = {
  args: {
    user: sampleUser,
    organization: sampleOrganization,
    onHelpClick: () => alert('Help clicked'),
    onUserMenuAction: (action) => alert(`User menu action: ${action}`),
    onOrganizationMenuAction: (action) => alert(`Organization menu action: ${action}`),
  },
  parameters: {
    backgrounds: {
      default: 'dark',
    },
  },
  decorators: [
    (Story) => (
      <div className="dark">
        <Story />
      </div>
    ),
  ],
}

export const CompareBothVersions: Story = {
  render: () => {
    const { Navigation } = require('./navigation')
    
    return (
      <div className="min-h-screen bg-zinc-50 dark:bg-zinc-900">
        <div className="space-y-8">
          <div>
            <h2 className="text-lg font-semibold text-zinc-900 dark:text-white p-4">
              Original Navigation (Logo-centric)
            </h2>
            <Navigation
              user={sampleUser}
              organization={sampleOrganization}
              onHelpClick={() => console.log('Original nav: Help clicked')}
              onUserMenuAction={(action) => console.log(`Original nav: User ${action}`)}
              onOrganizationMenuAction={(action) => console.log(`Original nav: Org ${action}`)}
            />
          </div>
          
          <div>
            <h2 className="text-lg font-semibold text-zinc-900 dark:text-white p-4">
              Alternative Navigation (Organization-centric)
            </h2>
            <NavigationAlt
              user={sampleUser}
              organization={sampleOrganization}
              onHelpClick={() => console.log('Alt nav: Help clicked')}
              onUserMenuAction={(action) => console.log(`Alt nav: User ${action}`)}
              onOrganizationMenuAction={(action) => console.log(`Alt nav: Org ${action}`)}
            />
          </div>
        </div>
        
        <div className="p-8">
          <h1 className="text-2xl font-bold text-zinc-900 dark:text-white mb-4">
            Navigation Comparison
          </h1>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700">
              <h3 className="font-semibold mb-2">Original Navigation</h3>
              <ul className="text-sm text-zinc-600 dark:text-zinc-400 space-y-1">
                <li>• Full "SuperPlane" logo text prominent</li>
                <li>• Organization dropdown on right side</li>
                <li>• Traditional header layout</li>
                <li>• Equal weight to all elements</li>
              </ul>
            </div>
            <div className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700">
              <h3 className="font-semibold mb-2">Alternative Navigation</h3>
              <ul className="text-sm text-zinc-600 dark:text-zinc-400 space-y-1">
                <li>• Compact paper airplane icon logo</li>
                <li>• Organization name prominently left-aligned</li>
                <li>• Organization-centric layout</li>
                <li>• Better for multi-organization workflows</li>
              </ul>
            </div>
          </div>
        </div>
      </div>
    )
  },
}