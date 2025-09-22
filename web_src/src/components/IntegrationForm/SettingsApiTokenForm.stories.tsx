import React, { useState } from 'react'
import type { Meta, StoryObj } from '@storybook/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { SettingsApiTokenForm } from './SettingsApiTokenForm'
import type { IntegrationData, FormErrors } from './types'

// Create a query client for stories
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: false,
      staleTime: Infinity,
    },
  },
})

const meta: Meta<typeof SettingsApiTokenForm> = {
  title: 'Components/IntegrationForm/SettingsApiTokenForm',
  component: SettingsApiTokenForm,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
  decorators: [
    (Story) => (
      <QueryClientProvider client={queryClient}>
        <div className="w-[600px] p-6 bg-white dark:bg-zinc-900 rounded-lg">
          <Story />
        </div>
      </QueryClientProvider>
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
  }
]

export const CreateMode: Story = {
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

    return (
      <SettingsApiTokenForm
        integrationData={integrationData}
        setIntegrationData={setIntegrationData}
        errors={errors}
        setErrors={setErrors}
        secrets={mockSecrets}
        organizationId="org-123"
        canvasId="canvas-456"
        isEditMode={false}
      />
    )
  }
}

export const EditMode: Story = {
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
    const [newSecretValue, setNewSecretValue] = useState('')

    return (
      <SettingsApiTokenForm
        integrationData={integrationData}
        setIntegrationData={setIntegrationData}
        errors={errors}
        setErrors={setErrors}
        secrets={mockSecrets}
        organizationId="org-123"
        canvasId="canvas-456"
        isEditMode={true}
        newSecretValue={newSecretValue}
        setNewSecretValue={setNewSecretValue}
      />
    )
  }
}

export const EditModeWithValue: Story = {
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
    const [newSecretValue, setNewSecretValue] = useState('ghp_new_token_value_here')

    return (
      <SettingsApiTokenForm
        integrationData={integrationData}
        setIntegrationData={setIntegrationData}
        errors={errors}
        setErrors={setErrors}
        secrets={mockSecrets}
        organizationId="org-123"
        canvasId="canvas-456"
        isEditMode={true}
        newSecretValue={newSecretValue}
        setNewSecretValue={setNewSecretValue}
      />
    )
  }
}

export const WithSelectedSecret: Story = {
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

    return (
      <SettingsApiTokenForm
        integrationData={integrationData}
        setIntegrationData={setIntegrationData}
        errors={errors}
        setErrors={setErrors}
        secrets={mockSecrets}
        organizationId="org-123"
        canvasId="canvas-456"
        isEditMode={false}
      />
    )
  }
}

