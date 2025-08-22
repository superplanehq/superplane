import type { Meta, StoryObj } from '@storybook/react'
import { UserOrgDropdown } from './user-org-dropdown'

const meta: Meta<typeof UserOrgDropdown> = {
  title: 'Components/UserOrgDropdown',
  component: UserOrgDropdown,
  parameters: {
    layout: 'centered',
  },
  argTypes: {
    onUserMenuAction: { action: 'user menu clicked' },
    onOrganizationMenuAction: { action: 'organization menu clicked' },
  },
}

export default meta
type Story = StoryObj<typeof meta>

// Mock user and organization data
const currentUser = {
  id: '1',
  name: 'John Doe',
  email: 'john@superplane.com',
  initials: 'JD',
  avatar: 'https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?w=64&h=64&fit=crop&crop=face',
}

const currentOrganization = {
  id: '1',
  name: 'Confluent',
  plan: 'Pro',
  initials: 'C',
  avatar: 'https://confluent.io/favicon.ico',
}

export const Default: Story = {
  args: {
    user: currentUser,
    organization: currentOrganization,
  },
}

export const Plain: Story = {
  args: {
    user: currentUser,
    organization: currentOrganization,
    plain: true,
  },
}

export const WithoutPlan: Story = {
  args: {
    user: currentUser,
    organization: {
      ...currentOrganization,
      plan: undefined,
    },
  },
}