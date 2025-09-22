import type { Meta, StoryObj } from '@storybook/react'
import { EventSourceZeroState } from './index'
import { SuperplaneEventSourceSchedule } from '@/api-client/types.gen'

const meta: Meta<typeof EventSourceZeroState> = {
  title: 'Components/EventSourceZeroState',
  component: EventSourceZeroState,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
  argTypes: {
    eventSourceType: {
      control: 'select',
      options: ['semaphore', 'github', 'webhook', 'scheduled'],
    },
  },
}

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
  args: {
    eventSourceType: 'webhook',
  },
}

export const Semaphore: Story = {
  args: {
    eventSourceType: 'semaphore',
  },
}

export const GitHub: Story = {
  args: {
    eventSourceType: 'github',
  },
}

export const Webhook: Story = {
  args: {
    eventSourceType: 'webhook',
  },
}

export const ScheduledDaily: Story = {
  args: {
    eventSourceType: 'scheduled',
    schedule: {
      type: 'TYPE_DAILY',
      daily: {
        time: '09:00'
      }
    } as SuperplaneEventSourceSchedule,
  },
}

export const ScheduledWeekly: Story = {
  args: {
    eventSourceType: 'scheduled',
    schedule: {
      type: 'TYPE_WEEKLY',
      weekly: {
        weekDay: 'WEEK_DAY_FRIDAY',
        time: '14:30'
      }
    } as SuperplaneEventSourceSchedule,
  },
}

export const ScheduledHourly: Story = {
  args: {
    eventSourceType: 'scheduled',
    schedule: {
      type: 'TYPE_HOURLY',
      hourly: {
        minute: 15
      }
    } as SuperplaneEventSourceSchedule,
  },
}