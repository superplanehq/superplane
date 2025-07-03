import type { Meta, StoryObj } from '@storybook/react'
import { OrganizationPage } from './OrganizationPage'

const meta: Meta<typeof OrganizationPage> = {
  title: 'Pages/OrganizationPage',
  component: OrganizationPage,
  parameters: {
    layout: 'fullscreen',
    docs: {
      description: {
        component: 'A comprehensive organization dashboard built with our design system components, showcasing workflows, team members, groups, and settings.',
      },
    },
  },
  tags: ['autodocs'],
}

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
  args: {
    onSignOut: () => alert('Sign out clicked'),
  },
}

export const DarkMode: Story = {
  args: {
    onSignOut: () => alert('Sign out clicked'),
  },
  parameters: {
    backgrounds: {
      default: 'dark',
    },
    docs: {
      description: {
        story: 'The organization page with full dark mode support, showcasing how all components adapt to the dark theme.',
      },
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

export const Mobile: Story = {
  args: {
    onSignOut: () => alert('Sign out clicked'),
  },
  parameters: {
    viewport: {
      defaultViewport: 'mobile1',
    },
    docs: {
      description: {
        story: 'The responsive organization page on mobile devices with adaptive layout and stacked columns.',
      },
    },
  },
}

export const Tablet: Story = {
  args: {
    onSignOut: () => alert('Sign out clicked'),
  },
  parameters: {
    viewport: {
      defaultViewport: 'tablet',
    },
    docs: {
      description: {
        story: 'The organization page optimized for tablet viewing with balanced layout proportions.',
      },
    },
  },
}