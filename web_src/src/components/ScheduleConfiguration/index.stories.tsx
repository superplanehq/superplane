import type { Meta, StoryObj } from '@storybook/react'
import { useState } from 'react'
import { ScheduleConfiguration } from './index'
import { SuperplaneEventSourceSchedule } from '@/api-client/types.gen'

const meta: Meta<typeof ScheduleConfiguration> = {
  title: 'Components/ScheduleConfiguration',
  component: ScheduleConfiguration,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
  decorators: [
    (Story) => (
      <div style={{ width: '400px', padding: '20px' }}>
        <Story />
      </div>
    ),
  ],
}

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
  render: () => {
    const [schedule, setSchedule] = useState<SuperplaneEventSourceSchedule | null>(null)

    return (
      <ScheduleConfiguration
        schedule={schedule}
        onScheduleChange={setSchedule}
      />
    )
  },
}

export const WithDailySchedule: Story = {
  render: () => {
    const [schedule, setSchedule] = useState<SuperplaneEventSourceSchedule | null>({
      type: 'TYPE_DAILY',
      daily: {
        time: '14:30'
      }
    })

    return (
      <ScheduleConfiguration
        schedule={schedule}
        onScheduleChange={setSchedule}
      />
    )
  },
}

export const WithWeeklySchedule: Story = {
  render: () => {
    const [schedule, setSchedule] = useState<SuperplaneEventSourceSchedule | null>({
      type: 'TYPE_WEEKLY',
      weekly: {
        weekDay: 'WEEK_DAY_FRIDAY',
        time: '10:00'
      }
    })

    return (
      <ScheduleConfiguration
        schedule={schedule}
        onScheduleChange={setSchedule}
      />
    )
  },
}

export const WithHourlySchedule: Story = {
  render: () => {
    const [schedule, setSchedule] = useState<SuperplaneEventSourceSchedule | null>({
      type: 'TYPE_HOURLY',
      hourly: {
        minute: 30
      }
    })

    return (
      <ScheduleConfiguration
        schedule={schedule}
        onScheduleChange={setSchedule}
      />
    )
  },
}

export const WithErrors: Story = {
  render: () => {
    const [schedule, setSchedule] = useState<SuperplaneEventSourceSchedule | null>({
      type: 'TYPE_DAILY',
      daily: {
        time: '09:00'
      }
    })

    const errors = {
      scheduleType: 'Schedule type is required',
      time: 'Time must be specified',
    }

    return (
      <ScheduleConfiguration
        schedule={schedule}
        onScheduleChange={setSchedule}
        errors={errors}
      />
    )
  },
}