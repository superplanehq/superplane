import type { Meta, StoryObj } from '@storybook/react'
import { WorkflowNodeAccordion, type WorkflowNodeData } from '../workflow-node-accordion'

const meta: Meta<typeof WorkflowNodeAccordion> = {
  title: 'Components/WorkflowNodeAccordion',
  component: WorkflowNodeAccordion,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component: 'A workflow node component with accordion-based editing interface. Provides an alternative to tab-based editing with collapsible sections for better content organization.'
      }
    }
  },
  argTypes: {
    variant: {
      control: 'radio',
      options: ['read', 'edit'],
      description: 'The display variant of the node'
    },
    sections: {
      control: 'object',
      description: 'Custom accordion sections for edit mode. If not provided, all default sections will be shown.'
    },
    multiple: {
      control: 'boolean',
      description: 'Allow multiple accordion sections to be open simultaneously'
    },
    onUpdate: { action: 'updated' },
    onDelete: { action: 'deleted' },
    onEdit: { action: 'edit clicked' },
    onSave: { action: 'saved' },
    onCancel: { action: 'cancelled' }
  }
}

export default meta
type Story = StoryObj<typeof WorkflowNodeAccordion>

const sampleData: WorkflowNodeData = {
  id: '1',
  title: 'Send Email Notification',
  description: 'Sends an email notification to the user when the process completes',
  type: 'stage',
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

export const EditMultipleSections: Story = {
  args: {
    data: sampleData,
    variant: 'edit',
    multiple: true
  }
}

export const EditSingleSection: Story = {
  args: {
    data: sampleData,
    variant: 'edit',
    multiple: false
  }
}

export const EditCustomSections: Story = {
  args: {
    data: sampleData,
    variant: 'edit',
    sections: [
      {
        id: 'basic',
        title: '⚙️ Basic Setup',
        defaultOpen: true,
        content: <div className="p-4 bg-blue-50 dark:bg-blue-900/20 rounded">Basic configuration content here...</div>
      },
      {
        id: 'security',
        title: '🔒 Security Settings',
        content: <div className="p-4 bg-red-50 dark:bg-red-900/20 rounded">Security configuration content here...</div>
      },
      {
        id: 'preview',
        title: '👁️ Preview & Export',
        content: <div className="p-4 bg-green-50 dark:bg-green-900/20 rounded">Preview and export options here...</div>
      }
    ]
  }
}

export const TriggerNode: Story = {
  args: {
    data: {
      id: '2',
      title: 'User Registration',
      description: 'Triggered when a new user registers for an account',
      type: 'event',
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

export const MinimalSections: Story = {
  args: {
    data: sampleData,
    variant: 'edit',
    sections: [
      {
        id: 'config',
        title: 'Configuration',
        defaultOpen: true,
        content: <div className="space-y-4">
          <input className="w-full p-2 border rounded" placeholder="Enter configuration..." />
          <textarea className="w-full p-2 border rounded" rows={3} placeholder="Additional settings..."></textarea>
        </div>
      },
      {
        id: 'preview',
        title: 'Preview',
        content: <div className="bg-zinc-100 dark:bg-zinc-800 p-4 rounded font-mono text-sm">
          Generated output preview...
        </div>
      }
    ]
  }
}