import type { Meta, StoryObj } from '@storybook/react'
import { Navigation } from './navigation'

const meta: Meta<typeof Navigation> = {
  title: 'Components/Navigation',
  component: Navigation,
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
        <Navigation
          user={sampleUser}
          organization={sampleOrganization}
          onHelpClick={handleHelpClick}
          onUserMenuAction={handleUserMenuAction}
          onOrganizationMenuAction={handleOrganizationMenuAction}
        />
        <div className="p-8">
          <h1 className="text-2xl font-bold text-zinc-900 dark:text-white mb-4">
            Navigation Component Demo
          </h1>
          <p className="text-zinc-600 dark:text-zinc-400 mb-4">
            Click on the help icon, organization dropdown, or user avatar to see the interactive menus.
            Check the browser console for action logs.
          </p>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            <div className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700">
              <h3 className="font-semibold mb-2">Help</h3>
              <p className="text-sm text-zinc-600 dark:text-zinc-400">
                Click the help icon (?) to open help documentation.
              </p>
            </div>
            <div className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700">
              <h3 className="font-semibold mb-2">Organization Menu</h3>
              <p className="text-sm text-zinc-600 dark:text-zinc-400">
                Click the organization name to access settings, members, and billing.
              </p>
            </div>
            <div className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700">
              <h3 className="font-semibold mb-2">User Menu</h3>
              <p className="text-sm text-zinc-600 dark:text-zinc-400">
                Click your avatar to access profile, settings, and sign out.
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