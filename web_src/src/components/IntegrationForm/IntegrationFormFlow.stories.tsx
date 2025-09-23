import React, { useState, useRef } from 'react'
import type { Meta, StoryObj } from '@storybook/react'
import { MemoryRouter } from 'react-router-dom'
import { GitHubIntegrationForm } from './GitHubIntegrationForm'
import { SemaphoreIntegrationForm } from './SemaphoreIntegrationForm'
import { ApiTokenForm } from './ApiTokenForm'
import { useIntegrationForm } from './useIntegrationForm'
import { Button } from '../Button/button'
import { MaterialSymbol } from '../MaterialSymbol/material-symbol'
import type { IntegrationData, FormErrors } from './types'
import { createFlowMockSecrets, createMockIntegrations, defaultProps } from './__mocks__/storyFactory'

const meta: Meta = {
  title: 'Components/IntegrationForm/Complete Flow',
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
  decorators: [
    (Story) => (
      <MemoryRouter>
        <div className="w-[700px] p-8 bg-white dark:bg-zinc-900 rounded-lg">
          <Story />
        </div>
      </MemoryRouter>
    ),
  ],
}

export default meta
type Story = StoryObj<typeof meta>

const mockSecrets = createFlowMockSecrets()
const mockIntegrations = createMockIntegrations()

export const GitHubIntegrationFlow: Story = {
  render: () => {
    const [integrationType] = useState('github')
    const orgUrlRef = useRef<HTMLInputElement>(null)

    const {
      integrationData,
      setIntegrationData,
      apiTokenTab,
      setApiTokenTab,
      newSecretToken,
      setNewSecretToken,
      errors,
      setErrors,
      validateForm,
      resetForm
    } = useIntegrationForm({
      integrationType,
      integrations: mockIntegrations
    })

    const [isSubmitting, setIsSubmitting] = useState(false)
    const [submitResult, setSubmitResult] = useState<string | null>(null)

    const handleSubmit = async () => {
      if (!validateForm()) {
        setSubmitResult('❌ Validation failed. Please fix the errors above.')
        return
      }

      setIsSubmitting(true)
      setSubmitResult('⏳ Creating integration...')

      // Simulate API call
      await new Promise(resolve => setTimeout(resolve, 2000))

      setIsSubmitting(false)
      setSubmitResult('✅ Integration created successfully!')

      // Reset form after 3 seconds
      setTimeout(() => {
        resetForm()
        setSubmitResult(null)
      }, 3000)
    }

    return (
      <div className="space-y-6">
        <div className="border-b border-zinc-200 dark:border-zinc-700 pb-4">
          <h2 className="text-xl font-semibold text-zinc-900 dark:text-zinc-100">
            New GitHub Integration
          </h2>
          <p className="text-sm text-zinc-600 dark:text-zinc-400 mt-1">
            This story demonstrates the complete integration creation flow.
          </p>
        </div>

        <GitHubIntegrationForm
          integrationData={integrationData}
          setIntegrationData={setIntegrationData}
          errors={errors}
          setErrors={setErrors}
          apiTokenTab={apiTokenTab}
          setApiTokenTab={setApiTokenTab}
          newSecretToken={newSecretToken}
          setNewSecretToken={setNewSecretToken}
          secrets={mockSecrets}
          orgUrlRef={orgUrlRef}
        />

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
          organizationId={defaultProps.organizationId}
          canvasId={defaultProps.canvasId}
          orgUrlRef={orgUrlRef}
        />

        <div className="flex gap-3 pt-4 border-t border-zinc-200 dark:border-zinc-700">
          <Button
            onClick={() => {
              resetForm()
              setSubmitResult(null)
            }}
            disabled={isSubmitting}
          >
            <MaterialSymbol name="refresh" size="sm" />
            Reset Form
          </Button>

          <Button
            color="blue"
            onClick={handleSubmit}
            disabled={isSubmitting}
          >
            {isSubmitting ? (
              <>
                <MaterialSymbol name="progress_activity" className="animate-spin" size="sm" />
                Creating...
              </>
            ) : (
              <>
                <MaterialSymbol name="add" size="sm" />
                Create Integration
              </>
            )}
          </Button>
        </div>

        {submitResult && (
          <div className="p-4 rounded-lg bg-zinc-100 dark:bg-zinc-800 text-sm">
            {submitResult}
          </div>
        )}

        <div className="mt-8 p-4 rounded-lg bg-blue-50 dark:bg-blue-900/20 text-sm">
          <h3 className="font-medium text-blue-900 dark:text-blue-100 mb-2">
            📋 Form State (for debugging)
          </h3>
          <pre className="text-xs text-blue-800 dark:text-blue-200 overflow-auto">
            {JSON.stringify({
              integrationData,
              apiTokenTab,
              newSecretToken: newSecretToken ? '***hidden***' : '',
              errors
            }, null, 2)}
          </pre>
        </div>
      </div>
    )
  }
}

export const SemaphoreIntegrationFlow: Story = {
  render: () => {
    const [integrationType] = useState('semaphore')
    const orgUrlRef = useRef<HTMLInputElement>(null)

    const {
      integrationData,
      setIntegrationData,
      apiTokenTab,
      setApiTokenTab,
      newSecretToken,
      setNewSecretToken,
      errors,
      setErrors,
      validateForm,
      resetForm
    } = useIntegrationForm({
      integrationType,
      integrations: mockIntegrations
    })

    const [isSubmitting, setIsSubmitting] = useState(false)
    const [submitResult, setSubmitResult] = useState<string | null>(null)

    const handleSubmit = async () => {
      if (!validateForm()) {
        setSubmitResult('❌ Validation failed. Please fix the errors above.')
        return
      }

      setIsSubmitting(true)
      setSubmitResult('⏳ Creating integration...')

      // Simulate API call
      await new Promise(resolve => setTimeout(resolve, 2000))

      setIsSubmitting(false)
      setSubmitResult('✅ Integration created successfully!')

      // Reset form after 3 seconds
      setTimeout(() => {
        resetForm()
        setSubmitResult(null)
      }, 3000)
    }

    return (
      <div className="space-y-6">
        <div className="border-b border-zinc-200 dark:border-zinc-700 pb-4">
          <h2 className="text-xl font-semibold text-zinc-900 dark:text-zinc-100">
            New Semaphore Integration
          </h2>
          <p className="text-sm text-zinc-600 dark:text-zinc-400 mt-1">
            This story demonstrates the complete integration creation flow.
          </p>
        </div>

        <SemaphoreIntegrationForm
          integrationData={integrationData}
          setIntegrationData={setIntegrationData}
          errors={errors}
          setErrors={setErrors}
          apiTokenTab={apiTokenTab}
          setApiTokenTab={setApiTokenTab}
          newSecretToken={newSecretToken}
          setNewSecretToken={setNewSecretToken}
          secrets={mockSecrets}
          orgUrlRef={orgUrlRef}
        />

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
          organizationId={defaultProps.organizationId}
          canvasId={defaultProps.canvasId}
          orgUrlRef={orgUrlRef}
        />

        <div className="flex gap-3 pt-4 border-t border-zinc-200 dark:border-zinc-700">
          <Button
            onClick={() => {
              resetForm()
              setSubmitResult(null)
            }}
            disabled={isSubmitting}
          >
            <MaterialSymbol name="refresh" size="sm" />
            Reset Form
          </Button>

          <Button
            color="blue"
            onClick={handleSubmit}
            disabled={isSubmitting}
          >
            {isSubmitting ? (
              <>
                <MaterialSymbol name="progress_activity" className="animate-spin" size="sm" />
                Creating...
              </>
            ) : (
              <>
                <MaterialSymbol name="add" size="sm" />
                Create Integration
              </>
            )}
          </Button>
        </div>

        {submitResult && (
          <div className="p-4 rounded-lg bg-zinc-100 dark:bg-zinc-800 text-sm">
            {submitResult}
          </div>
        )}
      </div>
    )
  }
}

export const EditIntegrationFlow: Story = {
  render: () => {
    const [integrationType] = useState('github')
    const orgUrlRef = useRef<HTMLInputElement>(null)

    // Simulate editing an existing integration
    const [integrationData, setIntegrationData] = useState<IntegrationData>({
      orgUrl: 'https://github.com/mycompany',
      name: 'mycompany-production',
      apiToken: {
        secretName: 'github-pat-production',
        secretKey: 'api-token'
      }
    })

    const [errors, setErrors] = useState<FormErrors>({})
    const [apiTokenTab, setApiTokenTab] = useState<'existing' | 'new'>('existing')
    const [newSecretToken, setNewSecretToken] = useState('')
    const [isSubmitting, setIsSubmitting] = useState(false)
    const [submitResult, setSubmitResult] = useState<string | null>(null)

    const handleSubmit = async () => {
      // Basic validation
      if (!integrationData.name.trim() || !integrationData.orgUrl.trim()) {
        setSubmitResult('❌ Please fill in all required fields.')
        return
      }

      setIsSubmitting(true)
      setSubmitResult('⏳ Updating integration...')

      // Simulate API call
      await new Promise(resolve => setTimeout(resolve, 1500))

      setIsSubmitting(false)
      setSubmitResult('✅ Integration updated successfully!')
    }

    return (
      <div className="space-y-6">
        <div className="border-b border-zinc-200 dark:border-zinc-700 pb-4">
          <h2 className="text-xl font-semibold text-zinc-900 dark:text-zinc-100">
            Edit GitHub Integration
          </h2>
          <p className="text-sm text-zinc-600 dark:text-zinc-400 mt-1">
            This story demonstrates editing an existing integration.
          </p>
        </div>

        <GitHubIntegrationForm
          integrationData={integrationData}
          setIntegrationData={setIntegrationData}
          errors={errors}
          setErrors={setErrors}
          apiTokenTab={apiTokenTab}
          setApiTokenTab={setApiTokenTab}
          newSecretToken={newSecretToken}
          setNewSecretToken={setNewSecretToken}
          secrets={mockSecrets}
          orgUrlRef={orgUrlRef}
        />

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
          organizationId={defaultProps.organizationId}
          canvasId={defaultProps.canvasId}
          orgUrlRef={orgUrlRef}
        />

        <div className="flex gap-3 pt-4 border-t border-zinc-200 dark:border-zinc-700">
          <Button
            disabled={isSubmitting}
          >
            <MaterialSymbol name="close" size="sm" />
            Cancel
          </Button>

          <Button
            color="blue"
            onClick={handleSubmit}
            disabled={isSubmitting}
          >
            {isSubmitting ? (
              <>
                <MaterialSymbol name="progress_activity" className="animate-spin" size="sm" />
                Updating...
              </>
            ) : (
              <>
                <MaterialSymbol name="save" size="sm" />
                Update Integration
              </>
            )}
          </Button>
        </div>

        {submitResult && (
          <div className="p-4 rounded-lg bg-zinc-100 dark:bg-zinc-800 text-sm">
            {submitResult}
          </div>
        )}
      </div>
    )
  }
}