import type { Meta, StoryObj } from '@storybook/react'
import { EventSourceSidebar } from './index'
import { SuperplaneEventSource, SuperplaneEvent } from '@/api-client'

const meta: Meta<typeof EventSourceSidebar> = {
  title: 'Components/EventSourceSidebar',
  component: EventSourceSidebar,
  parameters: {
    layout: 'fullscreen',
  },
  tags: ['autodocs'],
}

export default meta
type Story = StoryObj<typeof meta>

const mockEvents: SuperplaneEvent[] = [
  {
    id: '1',
    receivedAt: new Date().toISOString(),
    state: 'STATE_PROCESSED',
    type: 'push',
    sourceName: 'main-branch',
    headers: { 'content-type': 'application/json' },
    raw: { message: 'Push to main branch', author: 'john.doe' },
  },
  {
    id: '2',
    receivedAt: new Date(Date.now() - 300000).toISOString(), // 5 minutes ago
    state: 'STATE_PENDING',
    type: 'pull_request',
    sourceName: 'feature-branch',
    headers: { 'content-type': 'application/json' },
    raw: { message: 'Pull request opened', author: 'jane.smith' },
  },
  {
    id: '3',
    receivedAt: new Date(Date.now() - 900000).toISOString(), // 15 minutes ago
    state: 'STATE_REJECTED',
    stateReason: 'REASON_VALIDATION_FAILED',
    stateMessage: 'Invalid payload format',
    type: 'push',
    sourceName: 'develop-branch',
    headers: { 'content-type': 'application/json' },
    raw: { message: 'Push to develop branch' },
  },
]

const mockEventSourceWebhook: SuperplaneEventSource = {
  metadata: {
    id: 'webhook-source-1',
    name: 'Webhook Event Source',
  },
  spec: {
    integration: null,
    events: ['push', 'pull_request'],
  },
  events: mockEvents,
  eventSourceType: 'webhook',
}

const mockEventSourceGitHub: SuperplaneEventSource = {
  metadata: {
    id: 'github-source-1',
    name: 'My GitHub Repo',
  },
  spec: {
    integration: {
      name: 'github-integration',
    },
    events: ['push', 'pull_request'],
  },
  events: mockEvents,
  eventSourceType: 'github',
}

const mockEventSourceScheduledDaily: SuperplaneEventSource = {
  metadata: {
    id: 'scheduled-source-1',
    name: 'Daily Report Generator',
  },
  spec: {
    schedule: {
      type: 'TYPE_DAILY',
      daily: {
        time: '09:00'
      }
    },
    events: ['scheduled_event'],
  },
  events: mockEvents.slice(0, 2),
  eventSourceType: 'scheduled',
}

const mockEventSourceScheduledHourly: SuperplaneEventSource = {
  metadata: {
    id: 'scheduled-source-2',
    name: 'Hourly Data Sync',
  },
  spec: {
    schedule: {
      type: 'TYPE_HOURLY',
      hourly: {
        minute: 15
      }
    },
    events: ['sync_event'],
  },
  events: mockEvents,
  eventSourceType: 'scheduled',
}

export const WebhookEventSource: Story = {
  args: {
    selectedEventSource: mockEventSourceWebhook,
    onClose: () => console.log('Close sidebar'),
  },
}

export const GitHubEventSource: Story = {
  args: {
    selectedEventSource: mockEventSourceGitHub,
    onClose: () => console.log('Close sidebar'),
  },
}

export const ScheduledDailyEventSource: Story = {
  args: {
    selectedEventSource: mockEventSourceScheduledDaily,
    onClose: () => console.log('Close sidebar'),
  },
}

export const ScheduledHourlyEventSource: Story = {
  args: {
    selectedEventSource: mockEventSourceScheduledHourly,
    onClose: () => console.log('Close sidebar'),
  },
}

export const EmptyEventSource: Story = {
  args: {
    selectedEventSource: {
      ...mockEventSourceWebhook,
      events: [],
    },
    onClose: () => console.log('Close sidebar'),
  },
}

export const CustomWidth: Story = {
  args: {
    selectedEventSource: mockEventSourceGitHub,
    onClose: () => console.log('Close sidebar'),
    initialWidth: 400,
  },
}