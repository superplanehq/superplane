import { useState } from 'react'
import { MaterialSymbol } from '../MaterialSymbol/material-symbol'
import { Button } from '../Button/button'
import { Input } from '../Input/input'
import { Field, Label } from '../Fieldset/fieldset'
import { Textarea } from '../Textarea/textarea'
import {
  Dialog,
  DialogTitle,
  DialogDescription,
  DialogBody,
  DialogActions
} from '../Dialog/dialog'
import { showErrorToast } from '../../utils/toast'

interface CreateWorkflowModalProps {
  isOpen: boolean
  onClose: () => void
  onSubmit: (data: { name: string; description?: string }) => Promise<void>
  isLoading?: boolean
}

const MAX_WORKFLOW_NAME_LENGTH = 50
const MAX_WORKFLOW_DESCRIPTION_LENGTH = 200

export function CreateWorkflowModal({
  isOpen,
  onClose,
  onSubmit,
  isLoading = false
}: CreateWorkflowModalProps) {
  const [workflowName, setWorkflowName] = useState('')
  const [workflowDescription, setWorkflowDescription] = useState('')
  const [nameError, setNameError] = useState('')

  const handleClose = () => {
    setWorkflowName('')
    setWorkflowDescription('')
    setNameError('')
    onClose()
  }

  const handleSubmit = async () => {
    setNameError('')

    if (!workflowName.trim()) {
      setNameError('Workflow name is required')
      return
    }

    if (workflowName.trim().length > MAX_WORKFLOW_NAME_LENGTH) {
      setNameError(`Workflow name must be ${MAX_WORKFLOW_NAME_LENGTH} characters or less`)
      return
    }

    try {
      await onSubmit({
        name: workflowName.trim(),
        description: workflowDescription.trim() || undefined
      })

      // Reset form and close modal
      setWorkflowName('')
      setWorkflowDescription('')
      setNameError('')
      onClose()
    } catch (error) {
      console.error('Error creating workflow:', error)
      const errorMessage = ((error as Error)?.message) || error?.toString() || 'Failed to create workflow'

      showErrorToast(errorMessage)

      if (errorMessage.toLowerCase().includes('already') || errorMessage.toLowerCase().includes('exists')) {
        setNameError('A workflow with this name already exists')
      }
    }
  }

  return (
    <Dialog open={isOpen} onClose={handleClose} size="lg" className="text-left relative">
      <DialogTitle>Create New Workflow</DialogTitle>
      <DialogDescription className="text-sm">
        Create a new workflow that can use both components and custom components as building blocks.
      </DialogDescription>
      <button onClick={handleClose} className="absolute top-4 right-4">
        <MaterialSymbol name="close" size="sm" />
      </button>

      <DialogBody>
        <div className="space-y-6">
          {/* Workflow Name */}
          <Field>
            <Label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
              Workflow name *
            </Label>
            <Input
              type="text"
              value={workflowName}
              onChange={(e) => {
                if (e.target.value.length <= MAX_WORKFLOW_NAME_LENGTH) {
                  setWorkflowName(e.target.value)
                }
                if (nameError) {
                  setNameError('')
                }
              }}
              placeholder="Enter workflow name"
              className={`w-full ${nameError ? 'border-red-500' : ''}`}
              autoFocus
              maxLength={MAX_WORKFLOW_NAME_LENGTH}
            />
            <div className="text-xs text-zinc-500 dark:text-zinc-400 mt-1">
              {workflowName.length}/{MAX_WORKFLOW_NAME_LENGTH} characters
            </div>
            {nameError && (
              <div className="text-xs text-red-600 mt-1">
                {nameError}
              </div>
            )}
          </Field>

          {/* Workflow Description */}
          <Field>
            <Label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
              Description
            </Label>
            <Textarea
              value={workflowDescription}
              onChange={(e) => {
                if (e.target.value.length <= MAX_WORKFLOW_DESCRIPTION_LENGTH) {
                  setWorkflowDescription(e.target.value)
                }
              }}
              placeholder="Describe what this workflow will be used for (optional)"
              rows={3}
              className="w-full"
              maxLength={MAX_WORKFLOW_DESCRIPTION_LENGTH}
            />
            <div className="text-xs text-zinc-500 dark:text-zinc-400 mt-1">
              {workflowDescription.length}/{MAX_WORKFLOW_DESCRIPTION_LENGTH} characters
            </div>
          </Field>
        </div>
      </DialogBody>

      <DialogActions>
        <Button
          color="blue"
          onClick={handleSubmit}
          disabled={!workflowName.trim() || isLoading || !!nameError}
          className="flex items-center gap-2"
        >
          {isLoading ? 'Creating...' : 'Create'}
        </Button>
      </DialogActions>

    </Dialog>
  )
}
