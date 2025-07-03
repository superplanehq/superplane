import type { Meta, StoryObj } from '@storybook/react'
import { NavigationVertical } from './navigation-vertical'

const meta: Meta<typeof NavigationVertical> = {
  title: 'Components/NavigationVertical',
  component: NavigationVertical,
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
    showOrganization: true,
    onHelpClick: () => alert('Help clicked'),
    onUserMenuAction: (action) => alert(`User menu action: ${action}`),
    onOrganizationMenuAction: (action) => alert(`Organization menu action: ${action}`),
  },
  render: (args) => (
    <div className="flex min-h-screen">
      <NavigationVertical {...args} />
      <div className="flex-1 p-8 bg-zinc-50 dark:bg-zinc-900">
        <div className="max-w-2xl">
          <h1 className="text-2xl font-bold text-zinc-900 dark:text-white mb-4">
            Vertical Navigation Demo
          </h1>
          <p className="text-zinc-600 dark:text-zinc-400 mb-6">
            This is the main content area. The vertical navigation is fixed on the left side.
          </p>
          <div className="space-y-4">
            <div className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700">
              <h3 className="font-semibold mb-2">Navigation Features</h3>
              <ul className="text-sm text-zinc-600 dark:text-zinc-400 space-y-1">
                <li>• Compact 64px width (16 Tailwind units)</li>
                <li>• Logo at the top</li>
                <li>• Help icon above avatars</li>
                <li>• Organization avatar (configurable)</li>
                <li>• User avatar with online indicator</li>
                <li>• Dropdown menus appear to the right</li>
              </ul>
            </div>
          </div>
        </div>
      </div>
    </div>
  ),
}

export const WithoutOrganization: Story = {
  args: {
    user: sampleUser,
    showOrganization: false,
    onHelpClick: () => alert('Help clicked'),
    onUserMenuAction: (action) => alert(`User menu action: ${action}`),
  },
  render: (args) => (
    <div className="flex min-h-screen">
      <NavigationVertical {...args} />
      <div className="flex-1 p-8 bg-zinc-50 dark:bg-zinc-900">
        <div className="max-w-2xl">
          <h1 className="text-2xl font-bold text-zinc-900 dark:text-white mb-4">
            Without Organization
          </h1>
          <p className="text-zinc-600 dark:text-zinc-400">
            When showOrganization is false, only the user avatar is shown at the bottom.
          </p>
        </div>
      </div>
    </div>
  ),
}

export const WithoutUserAvatar: Story = {
  args: {
    user: sampleUserWithoutAvatar,
    organization: sampleOrganization,
    showOrganization: true,
    onHelpClick: () => alert('Help clicked'),
    onUserMenuAction: (action) => alert(`User menu action: ${action}`),
    onOrganizationMenuAction: (action) => alert(`Organization menu action: ${action}`),
  },
  render: (args) => (
    <div className="flex min-h-screen">
      <NavigationVertical {...args} />
      <div className="flex-1 p-8 bg-zinc-50 dark:bg-zinc-900">
        <div className="max-w-2xl">
          <h1 className="text-2xl font-bold text-zinc-900 dark:text-white mb-4">
            Without User Avatar
          </h1>
          <p className="text-zinc-600 dark:text-zinc-400">
            When no user avatar is provided, initials are shown instead.
          </p>
        </div>
      </div>
    </div>
  ),
}

export const WithOrganizationAvatar: Story = {
  args: {
    user: sampleUser,
    organization: sampleOrganizationWithAvatar,
    showOrganization: true,
    onHelpClick: () => alert('Help clicked'),
    onUserMenuAction: (action) => alert(`User menu action: ${action}`),
    onOrganizationMenuAction: (action) => alert(`Organization menu action: ${action}`),
  },
  render: (args) => (
    <div className="flex min-h-screen">
      <NavigationVertical {...args} />
      <div className="flex-1 p-8 bg-zinc-50 dark:bg-zinc-900">
        <div className="max-w-2xl">
          <h1 className="text-2xl font-bold text-zinc-900 dark:text-white mb-4">
            With Organization Avatar
          </h1>
          <p className="text-zinc-600 dark:text-zinc-400">
            Organization avatar image is displayed instead of initials.
          </p>
        </div>
      </div>
    </div>
  ),
}

export const DarkMode: Story = {
  args: {
    user: sampleUser,
    organization: sampleOrganization,
    showOrganization: true,
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
  render: (args) => (
    <div className="flex min-h-screen">
      <NavigationVertical {...args} />
      <div className="flex-1 p-8 bg-zinc-50 dark:bg-zinc-900">
        <div className="max-w-2xl">
          <h1 className="text-2xl font-bold text-zinc-900 dark:text-white mb-4">
            Dark Mode
          </h1>
          <p className="text-zinc-600 dark:text-zinc-400">
            The vertical navigation adapts to dark mode with proper contrast and styling.
          </p>
        </div>
      </div>
    </div>
  ),
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
      <div className="flex min-h-screen">
        <NavigationVertical
          user={sampleUser}
          organization={sampleOrganization}
          showOrganization={true}
          onHelpClick={handleHelpClick}
          onUserMenuAction={handleUserMenuAction}
          onOrganizationMenuAction={handleOrganizationMenuAction}
        />
        <div className="flex-1 p-8 bg-zinc-50 dark:bg-zinc-900">
          <div className="max-w-2xl">
            <h1 className="text-2xl font-bold text-zinc-900 dark:text-white mb-4">
              Interactive Demo
            </h1>
            <p className="text-zinc-600 dark:text-zinc-400 mb-6">
              Click on the avatars and help icon to see the interactive menus. 
              Check the browser console for action logs.
            </p>
            <div className="grid grid-cols-1 gap-4">
              <div className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700">
                <h3 className="font-semibold mb-2">Visual Indicators</h3>
                <ul className="text-sm text-zinc-600 dark:text-zinc-400 space-y-1">
                  <li>• Blue dot on organization avatar indicates active plan</li>
                  <li>• Green dot on user avatar shows online status</li>
                  <li>• Hover effects on all interactive elements</li>
                  <li>• Dropdown menus positioned to the right of the navigation</li>
                </ul>
              </div>
              <div className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700">
                <h3 className="font-semibold mb-2">Space Efficiency</h3>
                <ul className="text-sm text-zinc-600 dark:text-zinc-400 space-y-1">
                  <li>• Only 64px wide (4rem) for maximum content space</li>
                  <li>• Vertical layout works well for applications</li>
                  <li>• Consistent with modern dashboard designs</li>
                  <li>• Scalable for different screen sizes</li>
                </ul>
              </div>
            </div>
          </div>
        </div>
      </div>
    )
  },
}

export const CompactLayout: Story = {
  args: {
    user: sampleUser,
    organization: sampleOrganization,
    showOrganization: true,
    onHelpClick: () => alert('Help clicked'),
    onUserMenuAction: (action) => alert(`User menu action: ${action}`),
    onOrganizationMenuAction: (action) => alert(`Organization menu action: ${action}`),
  },
  render: (args) => (
    <div className="flex h-96 border border-zinc-200 dark:border-zinc-700 rounded-lg overflow-hidden">
      <NavigationVertical {...args} />
      <div className="flex-1 p-6 bg-zinc-50 dark:bg-zinc-900">
        <h3 className="font-semibold mb-2">Compact Demo</h3>
        <p className="text-sm text-zinc-600 dark:text-zinc-400">
          The navigation maintains its proportions even in smaller containers.
        </p>
      </div>
    </div>
  ),
}