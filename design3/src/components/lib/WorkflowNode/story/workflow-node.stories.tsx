import type { Meta, StoryObj } from '@storybook/react'
import { WorkflowNode, type WorkflowNodeData } from '../workflow-node'

const meta: Meta<typeof WorkflowNode> = {
  title: 'Components/WorkflowNode',
  component: WorkflowNode,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component: 'A workflow node component with read and edit variants. Used for building workflow diagrams and process flows. Supports configurable tabs in edit mode for customizing the editing interface.'
      }
    }
  },
  argTypes: {
    variant: {
      control: 'radio',
      options: ['read', 'edit'],
      description: 'The display variant of the node'
    },
    tabs: {
      control: 'object',
      description: 'Custom tab configuration for edit mode. If not provided, all default tabs will be shown.'
    },
    onUpdate: { action: 'updated' },
    onDelete: { action: 'deleted' },
    onEdit: { action: 'edit clicked' },
    onSave: { action: 'saved' },
    onCancel: { action: 'cancelled' }
  }
}

export default meta
type Story = StoryObj<typeof WorkflowNode>

const sampleData: WorkflowNodeData = {
  id: '1',
  title: 'Send Email Notification',
  description: 'Sends an email notification to the user when the process completes',
  type: 'action',
  status: 'success',
  yamlConfig: {
    apiVersion: 'v1',
    kind: 'Stage',
    metadata: {
      name: 'send-email-notification',
      canvasId: 'c2181c55-64ac-41ba-8925-0eaf0357b3f6'
    },
    spec: {
      secrets: [
        { name: 'email-service', key: 'api-key', value: 'sk-1234567890' }
      ],
      connections: [
        { 
          name: 'smtp-server', 
          type: 'email', 
          config: { 
            host: 'smtp.gmail.com', 
            port: 587,
            secure: false
          }
        }
      ],
      inputs: [
        { name: 'recipient', type: 'string', required: true },
        { name: 'subject', type: 'string', required: true },
        { name: 'body', type: 'string', required: false, defaultValue: 'Default email body' }
      ],
      inputMappings: {
        'recipient': '{{user.email}}',
        'subject': '{{process.title}} completed'
      },
      outputs: [
        { name: 'messageId', type: 'string', value: '{{response.messageId}}' },
        { name: 'status', type: 'string', value: '{{response.status}}' }
      ],
      executor: {
        type: 'email',
        config: {
          provider: 'smtp',
          timeout: 30000,
          retries: 3
        }
      }
    }
  }
}

export const ReadVariant: Story = {
  args: {
    data: sampleData,
    variant: 'read'
  }
}

export const EditVariant: Story = {
  args: {
    data: sampleData,
    variant: 'edit'
  }
}

export const EditWithCustomTabs: Story = {
  args: {
    data: sampleData,
    variant: 'edit',
    tabs: [
      { id: 'basic', label: 'Configuration' },
      { id: 'secrets', label: 'Security' },
      { id: 'preview', label: 'Output' }
    ]
  }
}

export const EditMinimalTabs: Story = {
  args: {
    data: sampleData,
    variant: 'edit',
    tabs: [
      { id: 'basic', label: 'Setup' },
      { id: 'preview', label: 'Preview' }
    ]
  }
}

export const TriggerNode: Story = {
  args: {
    data: {
      id: '2',
      title: 'User Registration',
      description: 'Triggered when a new user registers for an account',
      type: 'trigger',
      status: 'running',
      yamlConfig: {
        apiVersion: 'v1',
        kind: 'Stage',
        metadata: {
          name: 'user-registration-trigger',
          canvasId: 'c2181c55-64ac-41ba-8925-0eaf0357b3f6'
        },
        spec: {
          outputs: [
            { name: 'userId', type: 'string', value: '{{event.user.id}}' },
            { name: 'userEmail', type: 'string', value: '{{event.user.email}}' }
          ],
          executor: {
            type: 'webhook',
            config: {
              endpoint: '/webhooks/user-registration',
              method: 'POST'
            }
          }
        }
      }
    },
    variant: 'read'
  }
}

export const ConditionNode: Story = {
  args: {
    data: {
      id: '3',
      title: 'Check User Type',
      description: 'Determines if the user is a premium or free tier user',
      type: 'condition',
      status: 'pending'
    },
    variant: 'read'
  }
}

export const OutputNode: Story = {
  args: {
    data: {
      id: '4',
      title: 'Generate Report',
      description: 'Creates a detailed analytics report with user data',
      type: 'output',
      status: 'error'
    },
    variant: 'read'
  }
}

export const DisabledNode: Story = {
  args: {
    data: {
      id: '5',
      title: 'Legacy Process',
      description: 'Old workflow step that is no longer active',
      type: 'action',
      status: 'disabled'
    },
    variant: 'read'
  }
}

export const MinimalNode: Story = {
  args: {
    data: {
      id: '6',
      title: 'Simple Action',
      type: 'action'
    },
    variant: 'read'
  }
}