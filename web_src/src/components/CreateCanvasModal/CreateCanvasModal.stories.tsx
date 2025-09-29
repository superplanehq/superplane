import type { Meta, StoryObj } from '@storybook/react'
import { useState } from 'react'
import { CreateCanvasModal } from './CreateCanvasModal'
import { Button } from '../Button/button'

function ModalWrapper({
  isOpen: initialIsOpen = false,
}: {
  isOpen?: boolean
}) {
  const [isOpen, setIsOpen] = useState(initialIsOpen)
  const [isLoading, setIsLoading] = useState(false)

  const handleSubmit = async () => {
    setIsLoading(true)
    await new Promise(resolve => setTimeout(resolve, 500))
    setIsLoading(false)
    setIsOpen(false)
  }

  return (
    <div style={{ minHeight: '100vh', padding: '2rem' }}>
      <Button onClick={() => setIsOpen(true)}>Create Canvas</Button>
      <CreateCanvasModal
        isOpen={isOpen}
        onClose={() => setIsOpen(false)}
        onSubmit={handleSubmit}
        isLoading={isLoading}
      />
    </div>
  )
}

const meta: Meta<typeof ModalWrapper> = {
  title: 'Components/CreateCanvasModal',
  component: ModalWrapper,
  parameters: {
    layout: 'fullscreen',
    viewport: {
      viewports: {
        small: {
          name: 'Small',
          styles: {
            width: '400px',
            height: '600px',
          },
        },
        medium: {
          name: 'Medium',
          styles: {
            width: '800px',
            height: '600px',
          },
        },
        large: {
          name: 'Large',
          styles: {
            width: '1200px',
            height: '800px',
          },
        },
      },
      defaultViewport: 'medium',
    },
  },
  tags: ['autodocs'],
  argTypes: {
    isOpen: {
      control: 'boolean',
    },
  },
}

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
  args: {
    isOpen: false,
  },
}

export const OpenedModal: Story = {
  args: {
    isOpen: true,
  },
}
