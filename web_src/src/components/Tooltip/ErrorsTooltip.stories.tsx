import type { Meta, StoryObj } from '@storybook/react'
import { ErrorsTooltip } from './alerts-tooltip'
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
    alerts: [
      createAlert('1', 'ALERT_TYPE_ERROR', 'Database connection failed'),
      createAlert('2', 'ALERT_TYPE_WARNING', 'High memory usage detected'),
      createAlert('3', 'ALERT_TYPE_INFO', 'Deployment completed successfully'),
    ],
    onAcknowledge: (alertId: string) => {
      console.log('Acknowledging alert:', alertId)
    },
  },
}

export const ErrorOnly: Story = {
  args: {
    alerts: [
      createAlert('1', 'ALERT_TYPE_ERROR', 'Critical system failure'),
      createAlert('2', 'ALERT_TYPE_ERROR', 'Database connection timeout'),
    ],
    onAcknowledge: (alertId: string) => {
      console.log('Acknowledging alert:', alertId)
    },
  },
}

export const WarningOnly: Story = {
  args: {
    alerts: [
      createAlert('1', 'ALERT_TYPE_WARNING', 'High CPU usage'),
      createAlert('2', 'ALERT_TYPE_WARNING', 'Disk space running low'),
    ],
    onAcknowledge: (alertId: string) => {
      console.log('Acknowledging alert:', alertId)
    },
  },
}

export const InfoOnly: Story = {
  args: {
    alerts: [
      createAlert('1', 'ALERT_TYPE_INFO', 'System update available'),
      createAlert('2', 'ALERT_TYPE_INFO', 'Backup completed'),
    ],
    onAcknowledge: (alertId: string) => {
      console.log('Acknowledging alert:', alertId)
    },
  },
}

export const MultipleGrouped: Story = {
  args: {
    alerts: [
      createAlert('1', 'ALERT_TYPE_ERROR', 'Database connection failed'),
      createAlert('2', 'ALERT_TYPE_ERROR', 'Database connection failed'),
      createAlert('3', 'ALERT_TYPE_WARNING', 'High memory usage'),
      createAlert('4', 'ALERT_TYPE_WARNING', 'High memory usage'),
      createAlert('5', 'ALERT_TYPE_WARNING', 'High memory usage'),
      createAlert('6', 'ALERT_TYPE_INFO', 'System healthy'),
    ],
    onAcknowledge: (alertId: string) => {
      console.log('Acknowledging alert:', alertId)
    },
  },
}

export const Loading: Story = {
  args: {
    alerts: [],
    isLoading: true,
    onAcknowledge: (alertId: string) => {
      console.log('Acknowledging alert:', alertId)
    },
  },
}

export const Empty: Story = {
  args: {
    alerts: [],
    isLoading: false,
    onAcknowledge: (alertId: string) => {
      console.log('Acknowledging alert:', alertId)
    },
  },
}