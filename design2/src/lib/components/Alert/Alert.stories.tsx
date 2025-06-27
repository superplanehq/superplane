import type { Meta, StoryObj } from '@storybook/react'
import { useState } from 'react'
import { Alert, AlertTitle, AlertDescription, AlertBody, AlertActions } from './alert'
import { Button } from '../Button/button'

const meta: Meta<typeof Alert> = {
  title: 'Components/Alert',
  component: Alert,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
  argTypes: {
    size: {
      control: 'select',
      options: ['xs', 'sm', 'md', 'lg', 'xl', '2xl', '3xl', '4xl', '5xl'],
    },
    open: {
      control: 'boolean',
    },
  },
}

export default meta
type Story = StoryObj<typeof meta>

function AlertExample({ size = 'md', title = 'Confirm action', description = 'Are you sure you want to continue?' }) {
  const [isOpen, setIsOpen] = useState(false)

  return (
    <div>
      <Button onClick={() => setIsOpen(true)}>Open Alert</Button>
      <Alert open={isOpen} onClose={() => setIsOpen(false)} size={size}>
        <AlertTitle>{title}</AlertTitle>
        <AlertDescription>{description}</AlertDescription>
        <AlertActions>
          <Button color="red" onClick={() => setIsOpen(false)}>
            Delete
          </Button>
          <Button plain onClick={() => setIsOpen(false)}>
            Cancel
          </Button>
        </AlertActions>
      </Alert>
    </div>
  )
}

export const Default: Story = {
  render: () => <AlertExample />,
}

export const WithBody: Story = {
  render: () => {
    const [isOpen, setIsOpen] = useState(false)

    return (
      <div>
        <Button onClick={() => setIsOpen(true)}>Open Alert with Body</Button>
        <Alert open={isOpen} onClose={() => setIsOpen(false)}>
          <AlertTitle>Delete account</AlertTitle>
          <AlertDescription>
            This action cannot be undone. This will permanently delete your account and remove your data from our servers.
          </AlertDescription>
          <AlertBody>
            <div className="space-y-3">
              <p className="text-sm text-zinc-600 dark:text-zinc-400">
                To confirm deletion, type "DELETE" in the box below:
              </p>
              <input
                type="text"
                className="w-full px-3 py-2 border border-zinc-300 rounded-md dark:border-zinc-600 dark:bg-zinc-800"
                placeholder="Type DELETE here"
              />
            </div>
          </AlertBody>
          <AlertActions>
            <Button color="red" onClick={() => setIsOpen(false)}>
              Delete Account
            </Button>
            <Button plain onClick={() => setIsOpen(false)}>
              Cancel
            </Button>
          </AlertActions>
        </Alert>
      </div>
    )
  },
}

export const Sizes: Story = {
  render: () => (
    <div className="space-y-4">
      <AlertExample size="xs" title="Small Alert" description="This is a small alert." />
      <AlertExample size="md" title="Medium Alert" description="This is a medium alert." />
      <AlertExample size="lg" title="Large Alert" description="This is a large alert." />
    </div>
  ),
}

export const Success: Story = {
  render: () => {
    const [isOpen, setIsOpen] = useState(false)

    return (
      <div>
        <Button color="green" onClick={() => setIsOpen(true)}>
          Show Success
        </Button>
        <Alert open={isOpen} onClose={() => setIsOpen(false)}>
          <AlertTitle>Success!</AlertTitle>
          <AlertDescription>Your changes have been saved successfully.</AlertDescription>
          <AlertActions>
            <Button color="green" onClick={() => setIsOpen(false)}>
              Continue
            </Button>
          </AlertActions>
        </Alert>
      </div>
    )
  },
}

export const Warning: Story = {
  render: () => {
    const [isOpen, setIsOpen] = useState(false)

    return (
      <div>
        <Button color="amber" onClick={() => setIsOpen(true)}>
          Show Warning
        </Button>
        <Alert open={isOpen} onClose={() => setIsOpen(false)}>
          <AlertTitle>Warning</AlertTitle>
          <AlertDescription>
            This action will affect multiple users. Please proceed with caution.
          </AlertDescription>
          <AlertActions>
            <Button color="amber" onClick={() => setIsOpen(false)}>
              Proceed
            </Button>
            <Button plain onClick={() => setIsOpen(false)}>
              Cancel
            </Button>
          </AlertActions>
        </Alert>
      </div>
    )
  },
}