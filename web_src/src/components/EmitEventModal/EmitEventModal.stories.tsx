import type { Meta, StoryObj } from '@storybook/react'
import { useState } from 'react'
import { EmitEventModal } from './EmitEventModal'
import { Button } from '../Button/button'
import { SuperplaneEvent } from '@/api-client'

// Wrapper component to handle modal state
function ModalWrapper({
  isOpen: initialIsOpen = false,
  sourceName = 'Test Event Source',
  nodeType = 'event_source',
  loadLastEvent,
  simulateLoading = false
}: Partial<React.ComponentProps<typeof EmitEventModal>> & {
  isOpen?: boolean
  simulateLoading?: boolean
}) {
  const [isOpen, setIsOpen] = useState(initialIsOpen)

  const defaultLoadLastEvent = async (): Promise<SuperplaneEvent | null> => {
    if (simulateLoading) {
      await new Promise(resolve => setTimeout(resolve, 2000))
    }
    return null
  }

  return (
    <div>
      <Button onClick={() => setIsOpen(true)}>Open Emit Event Modal</Button>
      <EmitEventModal
        isOpen={isOpen}
        onClose={() => setIsOpen(false)}
        sourceName={sourceName}
        nodeType={nodeType}
        loadLastEvent={loadLastEvent || defaultLoadLastEvent}
        onCancel={() => {}}
        onSubmit={async (eventType, eventData) => {
          await new Promise(resolve => setTimeout(resolve, 1000))
        }}
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
    nodeType: {
      control: 'select',
      options: ['event_source', 'stage'],
    },
    isOpen: {
      control: 'boolean',
    },
    simulateLoading: {
      control: 'boolean',
    },
  },
}

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
  args: {
    sourceName: 'GitHub Webhook',
    nodeType: 'event_source',
    isOpen: false,
  },
}

export const WithLastEvent: Story = {
  args: {
    sourceName: 'Slack Webhook',
    nodeType: 'event_source',
    isOpen: false,
    loadLastEvent: async () => ({
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
    }),
  },
}

export const LoadingState: Story = {
  args: {
    sourceName: 'API Webhook',
    nodeType: 'event_source',
    isOpen: false,
    simulateLoading: true,
  },
}

export const OpenedModal: Story = {
  args: {
    sourceName: 'GitHub Push',
    nodeType: 'event_source',
    isOpen: true,
  },
}

export const StageTemplates: Story = {
  args: {
    sourceName: 'SuperPlane Stage',
    nodeType: 'stage',
    isOpen: true,
  },
}
