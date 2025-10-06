import type { Meta, StoryObj } from '@storybook/react'
import { ErrorsTooltip } from './errors-tooltip'
import { SuperplaneAlert, AlertAlertType } from '@/api-client'

const meta: Meta<typeof ErrorsTooltip> = {
  title: 'Components/Tooltip/ErrorsTooltip',
  component: ErrorsTooltip,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
  argTypes: {
    isLoading: {
      control: 'boolean',
    },
  },
}

export default meta
type Story = StoryObj<typeof meta>

const createAlert = (id: string, type: AlertAlertType, message: string): SuperplaneAlert => ({
  id,
  type,
  message,
  canvasId: 'canvas-1',
  sourceId: 'source-1',
  sourceType: 'STAGE',
  acknowledged: false,
  createdAt: new Date().toISOString(),
})

export const Default: Story = {
  args: {
    errors: [
      createAlert('1', 'ERROR', 'Database connection failed'),
      createAlert('2', 'WARNING', 'High memory usage detected'),
      createAlert('3', 'INFO', 'Deployment completed successfully'),
    ],
    onAcknowledge: (alertId: string) => {
      console.log('Acknowledging alert:', alertId)
    },
  },
}

export const ErrorOnly: Story = {
  args: {
    errors: [
      createAlert('1', 'ERROR', 'Critical system failure'),
      createAlert('2', 'ERROR', 'Database connection timeout'),
    ],
    onAcknowledge: (alertId: string) => {
      console.log('Acknowledging alert:', alertId)
    },
  },
}

export const WarningOnly: Story = {
  args: {
    errors: [
      createAlert('1', 'WARNING', 'High CPU usage'),
      createAlert('2', 'WARNING', 'Disk space running low'),
    ],
    onAcknowledge: (alertId: string) => {
      console.log('Acknowledging alert:', alertId)
    },
  },
}

export const InfoOnly: Story = {
  args: {
    errors: [
      createAlert('1', 'INFO', 'System update available'),
      createAlert('2', 'INFO', 'Backup completed'),
    ],
    onAcknowledge: (alertId: string) => {
      console.log('Acknowledging alert:', alertId)
    },
  },
}

export const MultipleGrouped: Story = {
  args: {
    errors: [
      createAlert('1', 'ERROR', 'Database connection failed'),
      createAlert('2', 'ERROR', 'Database connection failed'),
      createAlert('3', 'WARNING', 'High memory usage'),
      createAlert('4', 'WARNING', 'High memory usage'),
      createAlert('5', 'WARNING', 'High memory usage'),
      createAlert('6', 'INFO', 'System healthy'),
    ],
    onAcknowledge: (alertId: string) => {
      console.log('Acknowledging alert:', alertId)
    },
  },
}

export const Loading: Story = {
  args: {
    errors: [],
    isLoading: true,
    onAcknowledge: (alertId: string) => {
      console.log('Acknowledging alert:', alertId)
    },
  },
}

export const Empty: Story = {
  args: {
    errors: [],
    isLoading: false,
    onAcknowledge: (alertId: string) => {
      console.log('Acknowledging alert:', alertId)
    },
  },
}