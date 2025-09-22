import React, { useState } from 'react'
import type { Meta, StoryObj } from '@storybook/react'
import { ApiTokenForm } from './ApiTokenForm'
import type { IntegrationData, FormErrors } from './types'

const meta: Meta<typeof ApiTokenForm> = {
  title: 'Components/IntegrationForm/ApiTokenForm',
  component: ApiTokenForm,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
  decorators: [
    (Story) => (
      <div className="w-[600px] p-6 bg-white dark:bg-zinc-900 rounded-lg">
        <Story />
      </div>
    ),
  ],
}

export default meta
type Story = StoryObj<typeof meta>

const mockSecrets = [
  {
    metadata: { id: '1', name: 'github-pat' },
    spec: {
      local: {
        data: {
          'api-token': 'ghp_xxxxxxxxxxxxxxxxxxxx',
          'backup-token': 'ghp_yyyyyyyyyyyyyyyyyyyy',
          'webhook-secret': 'whs_zzzzzzzzzzzzzzzzzzzz'
        }
      }
    }
  },
  {
    metadata: { id: '2', name: 'semaphore-key' },
    spec: {
      local: {
        data: {
          'token': 'smp_aaaaaaaaaaaaaaaaaaa',
          'api-key': 'smp_bbbbbbbbbbbbbbbbbb'
        }
      }
    }
  },
  {
    metadata: { id: '3', name: 'service-keys' },
    spec: {
      local: {
        data: {
          'primary': 'key_111111111111111',
          'secondary': 'key_222222222222222',
          'fallback': 'key_333333333333333'
        }
      }
    }
  }
]

export const NewSecretTab: Story = {
  render: (args) => {
    const [integrationData, setIntegrationData] = useState<IntegrationData>({
      orgUrl: 'https://github.com/myorg',
      name: 'myorg-account',
      apiToken: {
        secretName: '',
        secretKey: ''
      }
    })

    const [errors, setErrors] = useState<FormErrors>({})
    const [apiTokenTab, setApiTokenTab] = useState<'existing' | 'new'>('new')
    const [newSecretToken, setNewSecretToken] = useState('')

    return (
      <ApiTokenForm
        integrationData={integrationData}
        setIntegrationData={setIntegrationData}
        errors={errors}
        setErrors={setErrors}
        apiTokenTab={apiTokenTab}
        setApiTokenTab={setApiTokenTab}
        newSecretToken={newSecretToken}
        setNewSecretToken={setNewSecretToken}
        secrets={mockSecrets}
        organizationId="org-123"
        canvasId="canvas-456"
        {...args}
      />
    )
  }
}

export const ExistingSecretTab: Story = {
  render: (args) => {
    const [integrationData, setIntegrationData] = useState<IntegrationData>({
      orgUrl: 'https://github.com/myorg',
      name: 'myorg-account',
      apiToken: {
        secretName: '',
        secretKey: ''
      }
    })

    const [errors, setErrors] = useState<FormErrors>({})
    const [apiTokenTab, setApiTokenTab] = useState<'existing' | 'new'>('existing')
    const [newSecretToken, setNewSecretToken] = useState('')

    return (
      <ApiTokenForm
        integrationData={integrationData}
        setIntegrationData={setIntegrationData}
        errors={errors}
        setErrors={setErrors}
        apiTokenTab={apiTokenTab}
        setApiTokenTab={setApiTokenTab}
        newSecretToken={newSecretToken}
        setNewSecretToken={setNewSecretToken}
        secrets={mockSecrets}
        organizationId="org-123"
        canvasId="canvas-456"
        {...args}
      />
    )
  }
}

export const ExistingSecretSelected: Story = {
  render: (args) => {
    const [integrationData, setIntegrationData] = useState<IntegrationData>({
      orgUrl: 'https://github.com/myorg',
      name: 'myorg-account',
      apiToken: {
        secretName: 'github-pat',
        secretKey: 'api-token'
      }
    })

    const [errors, setErrors] = useState<FormErrors>({})
    const [apiTokenTab, setApiTokenTab] = useState<'existing' | 'new'>('existing')
    const [newSecretToken, setNewSecretToken] = useState('')

    return (
      <ApiTokenForm
        integrationData={integrationData}
        setIntegrationData={setIntegrationData}
        errors={errors}
        setErrors={setErrors}
        apiTokenTab={apiTokenTab}
        setApiTokenTab={setApiTokenTab}
        newSecretToken={newSecretToken}
        setNewSecretToken={setNewSecretToken}
        secrets={mockSecrets}
        organizationId="org-123"
        canvasId="canvas-456"
        {...args}
      />
    )
  }
}

export const WithNewSecretValue: Story = {
  render: (args) => {
    const [integrationData, setIntegrationData] = useState<IntegrationData>({
      orgUrl: 'https://api.semaphoreci.com',
      name: 'semaphore-integration',
      apiToken: {
        secretName: '',
        secretKey: ''
      }
    })

    const [errors, setErrors] = useState<FormErrors>({})
    const [apiTokenTab, setApiTokenTab] = useState<'existing' | 'new'>('new')
    const [newSecretToken, setNewSecretToken] = useState('smp_1234567890abcdef1234567890abcdef')

    return (
      <ApiTokenForm
        integrationData={integrationData}
        setIntegrationData={setIntegrationData}
        errors={errors}
        setErrors={setErrors}
        apiTokenTab={apiTokenTab}
        setApiTokenTab={setApiTokenTab}
        newSecretToken={newSecretToken}
        setNewSecretToken={setNewSecretToken}
        secrets={mockSecrets}
        organizationId="org-123"
        canvasId="canvas-456"
        {...args}
      />
    )
  }
}

export const WithErrors: Story = {
  render: (args) => {
    const [integrationData, setIntegrationData] = useState<IntegrationData>({
      orgUrl: 'https://github.com/myorg',
      name: 'myorg-account',
      apiToken: {
        secretName: '',
        secretKey: ''
      }
    })

    const [errors, setErrors] = useState<FormErrors>({
      apiToken: 'Please select a secret and key',
      secretValue: 'Field cannot be empty'
    })
    const [apiTokenTab, setApiTokenTab] = useState<'existing' | 'new'>('new')
    const [newSecretToken, setNewSecretToken] = useState('')

    return (
      <ApiTokenForm
        integrationData={integrationData}
        setIntegrationData={setIntegrationData}
        errors={errors}
        setErrors={setErrors}
        apiTokenTab={apiTokenTab}
        setApiTokenTab={setApiTokenTab}
        newSecretToken={newSecretToken}
        setNewSecretToken={setNewSecretToken}
        secrets={mockSecrets}
        organizationId="org-123"
        canvasId="canvas-456"
        {...args}
      />
    )
  }
}

export const NoSecretsAvailable: Story = {
  render: (args) => {
    const [integrationData, setIntegrationData] = useState<IntegrationData>({
      orgUrl: 'https://github.com/myorg',
      name: 'myorg-account',
      apiToken: {
        secretName: '',
        secretKey: ''
      }
    })

    const [errors, setErrors] = useState<FormErrors>({})
    const [apiTokenTab, setApiTokenTab] = useState<'existing' | 'new'>('existing')
    const [newSecretToken, setNewSecretToken] = useState('')

    return (
      <ApiTokenForm
        integrationData={integrationData}
        setIntegrationData={setIntegrationData}
        errors={errors}
        setErrors={setErrors}
        apiTokenTab={apiTokenTab}
        setApiTokenTab={setApiTokenTab}
        newSecretToken={newSecretToken}
        setNewSecretToken={setNewSecretToken}
        secrets={[]}
        organizationId="org-123"
        canvasId="canvas-456"
        {...args}
      />
    )
  }
}

export const MultipleKeyOptions: Story = {
  render: (args) => {
    const [integrationData, setIntegrationData] = useState<IntegrationData>({
      orgUrl: 'https://api.example.com',
      name: 'multi-key-integration',
      apiToken: {
        secretName: 'service-keys',
        secretKey: 'primary'
      }
    })

    const [errors, setErrors] = useState<FormErrors>({})
    const [apiTokenTab, setApiTokenTab] = useState<'existing' | 'new'>('existing')
    const [newSecretToken, setNewSecretToken] = useState('')

    return (
      <ApiTokenForm
        integrationData={integrationData}
        setIntegrationData={setIntegrationData}
        errors={errors}
        setErrors={setErrors}
        apiTokenTab={apiTokenTab}
        setApiTokenTab={setApiTokenTab}
        newSecretToken={newSecretToken}
        setNewSecretToken={setNewSecretToken}
        secrets={mockSecrets}
        organizationId="org-123"
        canvasId="canvas-456"
        {...args}
      />
    )
  }
}