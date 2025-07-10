import type { Meta, StoryObj } from '@storybook/react'
import { Breadcrumbs } from './breadcrumbs'

const meta = {
  title: 'Components/Breadcrumbs',
  component: Breadcrumbs,
  parameters: {
    layout: 'padded',
  },
  tags: ['autodocs'],
  argTypes: {
    separator: {
      control: { type: 'select' },
      options: ['/', '>', '•'],
    },
    showDivider: {
      control: { type: 'boolean' },
    },
  },
} satisfies Meta<typeof Breadcrumbs>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
  args: {
    items: [
      { label: 'Canvases', href: '/canvases', icon: 'folder' },
      { label: 'Canvas name', current: true }
    ],
  },
}

export const MultipleItems: Story = {
  args: {
    items: [
      { label: 'Organization', href: '/org', icon: 'business' },
      { label: 'Teams', href: '/teams', icon: 'group' },
      { label: 'Engineering Team', current: true }
    ],
  },
}

export const WithoutIcons: Story = {
  args: {
    items: [
      { label: 'Home', href: '/' },
      { label: 'Projects', href: '/projects' },
      { label: 'Project Alpha', current: true }
    ],
  },
}

export const DifferentSeparators: Story = {
  args: {
    items: [
      { label: 'Dashboard', href: '/dashboard', icon: 'dashboard' },
      { label: 'Settings', current: true, icon: 'settings' }
    ],
    separator: '>',
  },
}

export const BulletSeparator: Story = {
  args: {
    items: [
      { label: 'Workflows', href: '/workflows', icon: 'account_tree' },
      { label: 'Customer Onboarding', current: true }
    ],
    separator: '•',
  },
}

export const WithoutDivider: Story = {
  args: {
    items: [
      { label: 'Teams', href: '/teams', icon: 'group' },
      { label: 'Development Team', current: true }
    ],
    showDivider: false,
  },
}

export const SingleItem: Story = {
  args: {
    items: [
      { label: 'Current Page', current: true, icon: 'home' }
    ],
  },
}

export const LongBreadcrumbs: Story = {
  args: {
    items: [
      { label: 'Organization', href: '/org', icon: 'business' },
      { label: 'Teams', href: '/teams', icon: 'group' },
      { label: 'Engineering', href: '/teams/engineering', icon: 'code' },
      { label: 'Frontend Team', href: '/teams/engineering/frontend' },
      { label: 'React Components', current: true, icon: 'components' }
    ],
  },
}

export const NonClickableItems: Story = {
  args: {
    items: [
      { label: 'Archive', icon: 'archive' },
      { label: '2024', icon: 'calendar_today' },
      { label: 'January Reports', current: true, icon: 'description' }
    ],
  },
}

export const WithClickHandlers: Story = {
  args: {
    items: [
      { 
        label: 'Teams', 
        icon: 'group', 
        onClick: () => alert('Navigate to Teams') 
      },
      { 
        label: 'Engineering Team', 
        current: true 
      }
    ],
  },
  parameters: {
    docs: {
      description: {
        story: 'Breadcrumbs with onClick handlers instead of href links.'
      }
    }
  }
}

export const MixedInteractions: Story = {
  args: {
    items: [
      { label: 'Dashboard', href: '/dashboard', icon: 'dashboard' },
      { 
        label: 'Teams', 
        icon: 'group', 
        onClick: () => alert('Navigate to Teams') 
      },
      { label: 'Frontend Team', current: true }
    ],
  },
  parameters: {
    docs: {
      description: {
        story: 'Breadcrumbs with mixed href and onClick interactions.'
      }
    }
  }
}