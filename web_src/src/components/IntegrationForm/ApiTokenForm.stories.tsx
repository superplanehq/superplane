import React, { useState, useRef } from 'react'
import type { Meta, StoryObj } from '@storybook/react'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { ApiTokenForm } from './ApiTokenForm'
import type { IntegrationData, FormErrors } from './types'
import { createMockSecrets, defaultProps } from '../../../test/__mocks__/secrets'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: false,
    },
  },
})

const meta: Meta<typeof ApiTokenForm> = {
  title: 'Components/IntegrationForm/ApiTokenForm',
  component: ApiTokenForm,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
  decorators: [
    (Story) => (
      <QueryClientProvider client={queryClient}>
        <MemoryRouter>
          <div className="w-[600px] p-6 bg-white dark:bg-zinc-900 rounded-lg">
            <Story />
          </div>
        </MemoryRouter>
      </QueryClientProvider>
    ),
  ],
}

export default meta
type Story = StoryObj<typeof meta>

const mockSecrets = createMockSecrets()

export const NewSecretTab: Story = {
  render: () => {
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
    const orgUrlRef = useRef<HTMLInputElement>(null)

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
        orgUrlRef={orgUrlRef}
        secrets={mockSecrets}
        organizationId={defaultProps.organizationId}
        canvasId={defaultProps.canvasId}
      />
    )
  }
}

export const ExistingSecretTab: Story = {
  render: () => {
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
    const orgUrlRef = useRef<HTMLInputElement>(null)

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
        orgUrlRef={orgUrlRef}
        secrets={mockSecrets}
        organizationId={defaultProps.organizationId}
        canvasId={defaultProps.canvasId}
      />
    )
  }
}

export const ExistingSecretSelected: Story = {
  render: () => {
    const [integrationData, setIntegrationData] = useState<IntegrationData>({
      orgUrl: 'https://github.com/myorg',
      name: 'myorg-account',
      apiToken: {
        secretName: 'secret-1',
        secretKey: 'api-token'
      }
    })

    const [errors, setErrors] = useState<FormErrors>({})
    const [apiTokenTab, setApiTokenTab] = useState<'existing' | 'new'>('existing')
    const [newSecretToken, setNewSecretToken] = useState('')
    const orgUrlRef = useRef<HTMLInputElement>(null)

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
        orgUrlRef={orgUrlRef}
        secrets={mockSecrets}
        organizationId={defaultProps.organizationId}
        canvasId={defaultProps.canvasId}
      />
    )
  }
}

export const WithNewSecretValue: Story = {
  render: () => {
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
    const orgUrlRef = useRef<HTMLInputElement>(null)

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
        orgUrlRef={orgUrlRef}
        secrets={mockSecrets}
        organizationId={defaultProps.organizationId}
        canvasId={defaultProps.canvasId}
      />
    )
  }
}

export const WithErrors: Story = {
  render: () => {
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
    const orgUrlRef = useRef<HTMLInputElement>(null)

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
        orgUrlRef={orgUrlRef}
        secrets={mockSecrets}
        organizationId={defaultProps.organizationId}
        canvasId={defaultProps.canvasId}
      />
    )
  }
}

export const NoSecretsAvailable: Story = {
  render: () => {
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
    const orgUrlRef = useRef<HTMLInputElement>(null)

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
        orgUrlRef={orgUrlRef}
        secrets={[]}
        organizationId="org-123"
        canvasId="canvas-456"
      />
    )
  }
}

export const MultipleKeyOptions: Story = {
  render: () => {
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
    const orgUrlRef = useRef<HTMLInputElement>(null)

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
        orgUrlRef={orgUrlRef}
        secrets={mockSecrets}
        organizationId={defaultProps.organizationId}
        canvasId={defaultProps.canvasId}
      />
    )
  }
}

export const EditMode: Story = {
  render: () => {
    const [integrationData, setIntegrationData] = useState<IntegrationData>({
      orgUrl: 'https://api.semaphoreci.com',
      name: 'semaphore-integration',
      apiToken: {
        secretName: 'semaphore-api-key',
        secretKey: 'api-token'
      }
    })

    const [errors, setErrors] = useState<FormErrors>({})
    const [apiTokenTab, setApiTokenTab] = useState<'existing' | 'new'>('existing')
    const [newSecretToken, setNewSecretToken] = useState('')
    const [newSecretValue, setNewSecretValue] = useState('')
    const orgUrlRef = useRef<HTMLInputElement>(null)

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
        orgUrlRef={orgUrlRef}
        isEditMode={true}
        newSecretValue={newSecretValue}
        setNewSecretValue={setNewSecretValue}
        secrets={mockSecrets}
        organizationId={defaultProps.organizationId}
        canvasId={defaultProps.canvasId}
      />
    )
  }
}