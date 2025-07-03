import type { Meta, StoryObj } from '@storybook/react'
import { useState } from 'react'
import { Tabs, ControlledTabs, useTabs, type Tab } from './tabs'

const meta: Meta<typeof Tabs> = {
  title: 'Components/Tabs',
  component: Tabs,
  parameters: {
    layout: 'padded',
  },
  tags: ['autodocs'],
  argTypes: {
    variant: {
      control: 'select',
      options: ['default', 'pills', 'underline'],
    },
  },
}

export default meta
type Story = StoryObj<typeof meta>

// Sample tabs data
const sampleTabs: Tab[] = [
  {
    id: 'overview',
    label: 'Overview',
    icon: (
      <svg fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
      </svg>
    ),
  },
  {
    id: 'analytics',
    label: 'Analytics',
    icon: (
      <svg fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 8v8m-4-5v5m-4-2v2m-2 4h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
      </svg>
    ),
    count: 3,
  },
  {
    id: 'settings',
    label: 'Settings',
    icon: (
      <svg fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
      </svg>
    ),
  },
  {
    id: 'support',
    label: 'Support',
    icon: (
      <svg fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M18.364 5.636l-3.536 3.536m0 5.656l3.536 3.536M9.172 9.172L5.636 5.636m3.536 9.192L5.636 18.364M12 2.5a9.5 9.5 0 100 19 9.5 9.5 0 000-19z" />
      </svg>
    ),
    count: 12,
  },
]

const tabsWithDisabled: Tab[] = [
  ...sampleTabs,
  {
    id: 'premium',
    label: 'Premium Features',
    icon: (
      <svg fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
      </svg>
    ),
    disabled: true,
  },
]

export const Default: Story = {
  args: {
    tabs: sampleTabs,
    defaultTab: 'overview',
  },
}

export const Pills: Story = {
  args: {
    tabs: sampleTabs,
    defaultTab: 'analytics',
    variant: 'pills',
  },
}

export const Underline: Story = {
  args: {
    tabs: sampleTabs,
    defaultTab: 'settings',
    variant: 'underline',
  },
}

export const WithCounts: Story = {
  render: () => {
    const tabsWithCounts: Tab[] = [
      { id: 'inbox', label: 'Inbox', count: 23 },
      { id: 'drafts', label: 'Drafts', count: 5 },
      { id: 'sent', label: 'Sent', count: 0 },
      { id: 'spam', label: 'Spam', count: 127 },
    ]

    return (
      <div className="space-y-8">
        <div>
          <h3 className="text-lg font-semibold mb-4">Default with Counts</h3>
          <Tabs tabs={tabsWithCounts} defaultTab="inbox" />
        </div>
        <div>
          <h3 className="text-lg font-semibold mb-4">Pills with Counts</h3>
          <Tabs tabs={tabsWithCounts} defaultTab="spam" variant="pills" />
        </div>
      </div>
    )
  },
}

export const WithIcons: Story = {
  render: () => (
    <div className="space-y-8">
      <div>
        <h3 className="text-lg font-semibold mb-4">Icons Only</h3>
        <Tabs 
          tabs={sampleTabs.map(tab => ({ ...tab, label: '' }))} 
          defaultTab="overview" 
        />
      </div>
      <div>
        <h3 className="text-lg font-semibold mb-4">Icons with Labels</h3>
        <Tabs tabs={sampleTabs} defaultTab="analytics" />
      </div>
    </div>
  ),
}

export const DisabledTabs: Story = {
  args: {
    tabs: tabsWithDisabled,
    defaultTab: 'overview',
  },
}

export const Interactive: Story = {
  render: () => {
    const [activeTab, setActiveTab] = useState('overview')

    return (
      <div className="space-y-6">
        <div>
          <h3 className="text-lg font-semibold mb-4">Interactive Tabs</h3>
          <Tabs 
            tabs={sampleTabs} 
            defaultTab="overview"
            onTabChange={(tabId) => {
              setActiveTab(tabId)
              alert(`Tab changed to: ${tabId}`)
            }}
          />
        </div>
        
        <div className="p-4 bg-zinc-100 dark:bg-zinc-800 rounded-lg">
          <p className="text-sm">
            <strong>Active Tab:</strong> {activeTab}
          </p>
        </div>
      </div>
    )
  },
}

export const ControlledExample: Story = {
  render: () => {
    const [activeTab, setActiveTab] = useState('analytics')

    return (
      <div className="space-y-6">
        <div>
          <h3 className="text-lg font-semibold mb-4">Controlled Tabs</h3>
          <ControlledTabs 
            tabs={sampleTabs}
            activeTab={activeTab}
            onTabChange={setActiveTab}
            variant="underline"
          />
        </div>
        
        <div className="space-y-3">
          <p className="text-sm font-medium">External Controls:</p>
          <div className="flex gap-2">
            {sampleTabs.map((tab) => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`px-3 py-1 text-xs rounded ${
                  activeTab === tab.id
                    ? 'bg-blue-500 text-white'
                    : 'bg-zinc-200 text-zinc-700 hover:bg-zinc-300'
                }`}
              >
                {tab.label}
              </button>
            ))}
          </div>
        </div>
        
        <div className="p-4 bg-zinc-100 dark:bg-zinc-800 rounded-lg">
          <p className="text-sm">
            <strong>Current Tab:</strong> {activeTab}
          </p>
        </div>
      </div>
    )
  },
}

export const WithHook: Story = {
  render: () => {
    const tabs = useTabs('overview', sampleTabs)

    return (
      <div className="space-y-6">
        <div>
          <h3 className="text-lg font-semibold mb-4">Using useTabs Hook</h3>
          <ControlledTabs 
            tabs={tabs.tabs}
            activeTab={tabs.active}
            onTabChange={tabs.setActive}
            variant="pills"
          />
        </div>
        
        <div className="p-4 bg-zinc-100 dark:bg-zinc-800 rounded-lg">
          <p className="text-sm">
            <strong>Hook State:</strong> {tabs.active}
          </p>
        </div>
      </div>
    )
  },
}

export const ResponsiveExample: Story = {
  render: () => {
    const manyTabs: Tab[] = [
      { id: 'dashboard', label: 'Dashboard', count: 5 },
      { id: 'projects', label: 'Projects', count: 12 },
      { id: 'tasks', label: 'Tasks', count: 23 },
      { id: 'calendar', label: 'Calendar' },
      { id: 'documents', label: 'Documents', count: 45 },
      { id: 'reports', label: 'Reports', count: 8 },
      { id: 'team', label: 'Team Members', count: 15 },
      { id: 'settings', label: 'Settings' },
    ]

    return (
      <div className="space-y-8">
        <div>
          <h3 className="text-lg font-semibold mb-4">Many Tabs (Scrollable)</h3>
          <div className="overflow-x-auto">
            <Tabs tabs={manyTabs} defaultTab="tasks" />
          </div>
        </div>
      </div>
    )
  },
}

export const CustomStyling: Story = {
  render: () => (
    <div className="space-y-8">
      <div>
        <h3 className="text-lg font-semibold mb-4">Custom Styled Tabs</h3>
        <Tabs 
          tabs={sampleTabs} 
          defaultTab="overview"
          className="bg-gradient-to-r from-blue-50 to-purple-50 dark:from-blue-900/20 dark:to-purple-900/20 p-4 rounded-lg"
        />
      </div>
      
      <div>
        <h3 className="text-lg font-semibold mb-4">Compact Pills</h3>
        <Tabs 
          tabs={sampleTabs} 
          defaultTab="analytics"
          variant="pills"
          className="w-fit"
        />
      </div>
    </div>
  ),
}