import type { Meta, StoryObj } from '@storybook/react'
import { useState } from 'react'
import { EmitEventModal } from './EmitEventModal'
import { Button } from '../Button/button'

// Wrapper component to handle modal state
function ModalWrapper({
  isOpen: initialIsOpen = false,
  sourceName = 'Test Event Source',
  lastEvent
}: Partial<React.ComponentProps<typeof EmitEventModal>> & {
  isOpen?: boolean
}) {
  const [isOpen, setIsOpen] = useState(initialIsOpen)

  return (
    <div>
      <Button onClick={() => setIsOpen(true)}>Open Emit Event Modal</Button>
      <EmitEventModal
        isOpen={isOpen}
        onClose={() => setIsOpen(false)}
        sourceName={sourceName}
        lastEvent={lastEvent}
        onCancel={() => {}}
        onSubmit={async () => {}}
      />
    </div>
  )
}

const meta: Meta<typeof ModalWrapper> = {
  title: 'Components/EmitEventModal',
  component: ModalWrapper,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
  argTypes: {
    sourceName: {
      control: 'text',
    },
    isOpen: {
      control: 'boolean',
    },
  },
}

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
  args: {
    sourceName: 'GitHub Webhook',
    isOpen: false,
  },
}

export const WithLastEvent: Story = {
  args: {
    sourceName: 'Slack Webhook',
    isOpen: false,
    lastEvent: {
      id: 'event-123',
      type: 'message.created',
      raw: {
        user: 'john.doe',
        channel: '#general',
        message: 'Hello, world!',
        timestamp: '2024-01-15T10:30:00Z',
        metadata: {
          source: 'slack',
          version: '2.0'
        }
      },
      created: '2024-01-15T10:30:00Z',
    },
  },
}
