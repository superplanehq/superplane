import type { Meta, StoryObj } from '@storybook/react'
import { EventSourceZeroState } from './index'

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
      options: ['semaphore', 'github', 'webhook'],
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