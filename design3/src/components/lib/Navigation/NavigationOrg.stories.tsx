import type { Meta, StoryObj } from '@storybook/react'
import { NavigationOrg } from './navigation-org'

const meta: Meta<typeof NavigationOrg> = {
  title: 'Components/Navigation/NavigationOrg',
  component: NavigationOrg,
  parameters: {
    layout: 'fullscreen',
  },
  tags: ['autodocs'],
  argTypes: {
    className: {
      control: 'text',
      description: 'Additional CSS classes to apply to the navigation'
    },
    onHelpClick: {
      action: 'help-clicked',
      description: 'Callback when help button is clicked'
    },
    onUserMenuAction: {
      action: 'user-action',
      description: 'Callback when user menu action is triggered'
    },
    onOrganizationMenuAction: {
      action: 'org-action', 
      description: 'Callback when organization menu action is triggered'
    }
  }
}

export default meta
type Story = StoryObj<typeof meta>

// Mock data
const defaultUser = {
  id: '1',
  name: 'John Doe',
  email: 'john@superplane.com',
  initials: 'JD',
  avatar: 'https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?w=64&h=64&fit=crop&crop=face'
}

const defaultOrganization = {
  id: '1', 
  name: 'Acme Corporation',
  plan: 'Pro Plan',
  initials: 'AC'
}

export const Default: Story = {
  args: {
    user: defaultUser,
    organization: defaultOrganization
  }
}

export const WithoutUserAvatar: Story = {
  args: {
    user: {
      ...defaultUser,
      avatar: undefined
    },
    organization: defaultOrganization
  },
  parameters: {
    docs: {
      description: {
        story: 'Navigation with user initials displayed when no avatar is provided.'
      }
    }
  }
}

export const WithOrganizationAvatar: Story = {
  args: {
    user: defaultUser,
    organization: {
      ...defaultOrganization,
      avatar: 'https://images.unsplash.com/photo-1560472354-b33ff0c44a43?w=64&h=64&fit=crop&crop=center'
    }
  },
  parameters: {
    docs: {
      description: {
        story: 'Navigation with organization avatar instead of initials.'
      }
    }
  }
}

export const WithoutPlan: Story = {
  args: {
    user: defaultUser,
    organization: {
      id: '1',
      name: 'Startup Inc',
      initials: 'SI'
    }
  },
  parameters: {
    docs: {
      description: {
        story: 'Navigation for organizations without a plan badge.'
      }
    }
  }
}

export const LongNames: Story = {
  args: {
    user: {
      id: '1',
      name: 'Alexander Christopher Wellington',
      email: 'alexander.christopher.wellington@verylongdomainname.com',
      initials: 'AW'
    },
    organization: {
      id: '1',
      name: 'Very Long Organization Name That Might Overflow',
      plan: 'Enterprise Premium Plan',
      initials: 'VL'
    }
  },
  parameters: {
    docs: {
      description: {
        story: 'Navigation with long user and organization names to test text truncation.'
      }
    }
  }
}

export const DifferentPlanTypes: Story = {
  args: {
    user: defaultUser,
    organization: {
      ...defaultOrganization,
      plan: 'Free'
    }
  },
  parameters: {
    docs: {
      description: {
        story: 'Navigation with different plan types.'
      }
    }
  }
}

export const EnterprisePlan: Story = {
  args: {
    user: defaultUser,
    organization: {
      ...defaultOrganization,
      name: 'Enterprise Corp',
      plan: 'Enterprise',
      initials: 'EC'
    }
  }
}

export const PersonalAccount: Story = {
  args: {
    user: {
      id: '1',
      name: 'Jane Smith',
      email: 'jane@gmail.com',
      initials: 'JS',
      avatar: 'https://images.unsplash.com/photo-1494790108755-2616b612b789?w=64&h=64&fit=crop&crop=face'
    },
    organization: {
      id: '1',
      name: 'Personal Workspace',
      initials: 'PW'
    }
  },
  parameters: {
    docs: {
      description: {
        story: 'Navigation for personal accounts without organization plans.'
      }
    }
  }
}

export const CustomStyling: Story = {
  args: {
    user: defaultUser,
    organization: defaultOrganization,
    className: 'border-b-2 border-blue-500'
  },
  parameters: {
    docs: {
      description: {
        story: 'Navigation with custom styling applied via className prop.'
      }
    }
  }
}

export const DarkMode: Story = {
  args: {
    user: defaultUser,
    organization: defaultOrganization
  },
  parameters: {
    backgrounds: { default: 'dark' },
    docs: {
      description: {
        story: 'Navigation component in dark mode theme.'
      }
    }
  },
  decorators: [
    (Story) => (
      <div className="dark">
        <Story />
      </div>
    )
  ]
}

export const Interactive: Story = {
  args: {
    user: defaultUser,
    organization: defaultOrganization
  },
  parameters: {
    docs: {
      description: {
        story: 'Interactive navigation to demonstrate functionality.'
      }
    }
  }
}

export const WithContent: Story = {
  args: {
    user: defaultUser,
    organization: defaultOrganization
  },
  decorators: [
    (Story) => (
      <div>
        <Story />
        <div className="p-8 bg-zinc-50 min-h-screen">
          <div className="max-w-4xl mx-auto">
            <h1 className="text-3xl font-bold text-zinc-900 mb-4">Dashboard</h1>
            <p className="text-zinc-600 mb-8">
              This shows how the navigation looks when placed above page content.
            </p>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
              {[1, 2, 3, 4, 5, 6].map((i) => (
                <div key={i} className="bg-white p-6 rounded-lg border border-zinc-200">
                  <h3 className="font-semibold text-zinc-900 mb-2">Card {i}</h3>
                  <p className="text-zinc-600 text-sm">
                    Sample content to demonstrate the navigation in context.
                  </p>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    )
  ],
  parameters: {
    docs: {
      description: {
        story: 'Navigation shown in context with page content below.'
      }
    }
  }
}

export const MobileView: Story = {
  args: {
    user: defaultUser,
    organization: defaultOrganization
  },
  parameters: {
    viewport: {
      defaultViewport: 'mobile1'
    },
    docs: {
      description: {
        story: 'Navigation optimized for mobile viewports where organization info is hidden.'
      }
    }
  }
}

export const TabletView: Story = {
  args: {
    user: defaultUser,
    organization: defaultOrganization
  },
  parameters: {
    viewport: {
      defaultViewport: 'tablet'
    },
    docs: {
      description: {
        story: 'Navigation on tablet-sized screens.'
      }
    }
  }
}