import type { Meta, StoryObj } from '@storybook/react';
import { EmptyState } from '../empty-state';
import { MaterialSymbol } from '../../MaterialSymbol/material-symbol';

const meta: Meta<typeof EmptyState> = {
  title: 'Components/EmptyState',
  component: EmptyState,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component: 'A reusable empty state component with optional image, title, body text, and call-to-action buttons.'
      }
    }
  },
  argTypes: {
    size: {
      control: 'select',
      options: ['sm', 'md', 'lg']
    },
    icon: {
      control: 'text'
    },
    animated: {
      control: 'boolean'
    },
    animationType: {
      control: 'select',
      options: ['pulse', 'bounce', 'spin', 'ping']
    },
    primaryAction: {
      control: 'object'
    },
    secondaryAction: {
      control: 'object'
    }
  },
  decorators: [
    (Story) => (
      <div className="w-full max-w-2xl mx-auto p-8 bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800">
        <Story />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof EmptyState>;

// Basic empty state with default icon
export const Default: Story = {
  args: {
    title: 'Start by adding data assets',
    body: 'Data assets help you organize and manage your information. Click the button below to create your first data asset and get started.',
    primaryAction: {
      label: 'Add data asset',
      onClick: () => alert('Primary action clicked!')
    },
    secondaryAction: {
      label: 'Learn more about data assets',
      onClick: () => alert('Secondary action clicked!')
    }
  }
};

// With custom icon
export const WithCustomIcon: Story = {
  args: {
    icon: 'integration_instructions',
    title: 'Connect your first integration',
    body: 'Integrations allow you to connect external services and automate your workflows. Browse our catalog to find the perfect integration for your needs.',
    primaryAction: {
      label: 'Browse integrations',
      color: 'blue'
    },
    secondaryAction: {
      label: 'View documentation'
    }
  }
};

// With custom image
export const WithCustomImage: Story = {
  args: {
    image: (
      <div className="w-16 h-16 bg-blue-100 dark:bg-blue-900 rounded-full flex items-center justify-center">
        <MaterialSymbol name="cloud_upload" className="text-blue-600 dark:text-blue-400 text-3xl" />
      </div>
    ),
    title: 'Upload your first file',
    body: 'Drag and drop files here or click the upload button to get started. Supported formats include CSV, JSON, and XML.',
    primaryAction: {
      label: 'Upload files',
      color: 'blue'
    }
  }
};

// Small size variant
export const SmallSize: Story = {
  args: {
    size: 'sm',
    icon: 'person_add',
    title: 'Invite team members',
    body: 'Collaborate with your team by inviting members to this workspace.',
    primaryAction: {
      label: 'Send invite',
      color: 'green'
    }
  }
};

// Large size variant
export const LargeSize: Story = {
  args: {
    size: 'lg',
    icon: 'analytics',
    title: 'No data to display',
    body: 'Once you start collecting data, your analytics and insights will appear here. Connect a data source or import existing data to begin.',
    primaryAction: {
      label: 'Connect data source',
      color: 'blue'
    },
    secondaryAction: {
      label: 'Import existing data'
    }
  }
};

// Without primary action
export const WithoutPrimaryAction: Story = {
  args: {
    icon: 'schedule',
    title: 'No scheduled tasks yet',
    body: 'You haven\'t scheduled any tasks yet. Tasks will appear here once you create them.',
    secondaryAction: {
      label: 'Learn about task scheduling'
    }
  }
};

// Without secondary action
export const WithoutSecondaryAction: Story = {
  args: {
    icon: 'folder_open',
    title: 'Create your first project',
    body: 'Projects help you organize your work and collaborate with team members. Click below to create your first project.',
    primaryAction: {
      label: 'Create project',
      color: 'blue'
    }
  }
};

// Error state variant
export const ErrorState: Story = {
  args: {
    icon: 'error_outline',
    title: 'Unable to load data',
    body: 'We encountered an issue while trying to load your data. Please check your connection and try again.',
    primaryAction: {
      label: 'Retry',
      color: 'blue'
    },
    secondaryAction: {
      label: 'Contact support'
    }
  }
};

// Success state variant  
export const SuccessState: Story = {
  args: {
    icon: 'check_circle_outline',
    title: 'All caught up!',
    body: 'You\'ve completed all your tasks. Great job! New tasks will appear here when they\'re assigned to you.',
    secondaryAction: {
      label: 'View completed tasks'
    }
  }
};

// Animated variants
export const AnimatedPulse: Story = {
  args: {
    icon: 'hearing',
    animated: true,
    animationType: 'pulse',
    title: 'Listening for events',
    body: 'Your event source is ready to receive webhooks and trigger workflows. Events will appear here once received.',
    primaryAction: {
      label: 'Test webhook',
      color: 'blue'
    }
  }
};

export const AnimatedBounce: Story = {
  args: {
    icon: 'refresh',
    animated: true,
    animationType: 'bounce',
    title: 'Syncing data',
    body: 'We\'re fetching the latest information from your connected services. This usually takes a few moments.',
    secondaryAction: {
      label: 'Learn about sync process'
    }
  }
};

export const AnimatedSpin: Story = {
  args: {
    icon: 'settings',
    animated: true,
    animationType: 'spin',
    title: 'Processing configuration',
    body: 'Your settings are being applied and synchronized across all services. Please wait a moment.',
    primaryAction: {
      label: 'View progress',
      color: 'blue'
    }
  }
};

export const AnimatedPing: Story = {
  args: {
    icon: 'notifications',
    animated: true,
    animationType: 'ping',
    title: 'Waiting for notifications',
    body: 'You\'ll receive alerts and updates here when important events occur in your workflows.',
    primaryAction: {
      label: 'Configure alerts',
      color: 'green'
    },
    secondaryAction: {
      label: 'Notification settings'
    }
  }
};

// Animated with custom image
export const AnimatedCustomImage: Story = {
  args: {
    animated: true,
    animationType: 'pulse',
    image: (
      <div className="w-16 h-16 bg-gradient-to-br from-blue-100 to-purple-100 dark:from-blue-900 dark:to-purple-900 rounded-full flex items-center justify-center">
        <MaterialSymbol name="cloud_sync" size={48} className="text-blue-600 dark:text-blue-400" />
      </div>
    ),
    title: 'Syncing with cloud',
    body: 'Your data is being synchronized with cloud services. This ensures all your information is up to date.',
    primaryAction: {
      label: 'View sync status',
      color: 'blue'
    }
  }
};

// Large size showcase
export const LargeSize: Story = {
  args: {
    size: 'lg',
    icon: 'analytics',
    title: 'No analytics data',
    body: 'Your analytics dashboard will populate with data as users interact with your application.',
    primaryAction: {
      label: 'Learn about analytics',
      color: 'blue'
    }
  }
};

// Custom large icon sizes
export const CustomIconSizes: Story = {
  render: () => (
    <div className="space-y-8">
      <div className="text-center">
        <h3 className="text-lg font-semibold mb-4">Custom Large Icon Sizes</h3>
        <div className="space-y-6">
          <EmptyState
            size="sm"
            icon="favorite"
            title="Small (48px icon)"
            body="Perfect for inline empty states and small containers."
          />
          <EmptyState
            size="md" 
            icon="star"
            title="Medium (64px icon)"
            body="Great for standard empty states and main content areas."
          />
          <EmptyState
            size="lg"
            icon="celebration"
            title="Large (80px icon)"
            body="Ideal for prominent empty states and hero sections."
          />
        </div>
      </div>
    </div>
  )
};