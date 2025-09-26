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

interface CreateCanvasModalProps {
  isOpen: boolean
  onClose: () => void
  onSubmit: (data: { name: string; description?: string }) => Promise<void>
  isLoading?: boolean
}

const MAX_CANVAS_NAME_LENGTH = 50
const MAX_CANVAS_DESCRIPTION_LENGTH = 200

export function CreateCanvasModal({
  isOpen,
  onClose,
  onSubmit,
  isLoading = false
}: CreateCanvasModalProps) {
  const [canvasName, setCanvasName] = useState('')
  const [canvasDescription, setCanvasDescription] = useState('')
  const [nameError, setNameError] = useState('')

  const handleClose = () => {
    setCanvasName('')
    setCanvasDescription('')
    setNameError('')
    onClose()
  }

  const handleSubmit = async () => {
    setNameError('')

    if (!canvasName.trim()) {
      setNameError('Canvas name is required')
      return
    }

    if (canvasName.trim().length > MAX_CANVAS_NAME_LENGTH) {
      setNameError(`Canvas name must be ${MAX_CANVAS_NAME_LENGTH} characters or less`)
      return
    }

    try {
      await onSubmit({
        name: canvasName.trim(),
        description: canvasDescription.trim() || undefined
      })

      // Reset form and close modal
      setCanvasName('')
      setCanvasDescription('')
      setNameError('')
      onClose()
    } catch (error) {
      console.error('Error creating canvas:', error)
      const errorMessage = ((error as Error)?.message) || error?.toString() || 'Failed to create canvas'

      showErrorToast(errorMessage)

      if (errorMessage.toLowerCase().includes('already') || errorMessage.toLowerCase().includes('exists')) {
        setNameError('A canvas with this name already exists')
      }
    }
  }

  return (
    <Dialog open={isOpen} onClose={handleClose} size="lg" className="text-left relative">
      <DialogTitle>Create New Canvas</DialogTitle>
      <DialogDescription className="text-sm">
        Create a new interactive canvas to build and manage your workflows.
      </DialogDescription>
      <button onClick={handleClose} className="absolute top-4 right-4">
        <MaterialSymbol name="close" size="sm" />
      </button>

      <DialogBody>
        <div className="space-y-6">
          {/* Canvas Name */}
          <Field>
            <Label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
              Canvas name *
            </Label>
            <Input
              type="text"
              value={canvasName}
              onChange={(e) => {
                if (e.target.value.length <= MAX_CANVAS_NAME_LENGTH) {
                  setCanvasName(e.target.value)
                }
                if (nameError) {
                  setNameError('')
                }
              }}
              placeholder="Enter canvas name"
              className={`w-full ${nameError ? 'border-red-500' : ''}`}
              autoFocus
              maxLength={MAX_CANVAS_NAME_LENGTH}
            />
            <div className="text-xs text-zinc-500 dark:text-zinc-400 mt-1">
              {canvasName.length}/{MAX_CANVAS_NAME_LENGTH} characters
            </div>
            {nameError && (
              <div className="text-xs text-red-600 mt-1">
                {nameError}
              </div>
            )}
          </Field>

          {/* Canvas Description */}
          <Field>
            <Label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
              Description
            </Label>
            <Textarea
              value={canvasDescription}
              onChange={(e) => {
                if (e.target.value.length <= MAX_CANVAS_DESCRIPTION_LENGTH) {
                  setCanvasDescription(e.target.value)
                }
              }}
              placeholder="Describe what this canvas will be used for (optional)"
              rows={3}
              className="w-full"
              maxLength={MAX_CANVAS_DESCRIPTION_LENGTH}
            />
            <div className="text-xs text-zinc-500 dark:text-zinc-400 mt-1">
              {canvasDescription.length}/{MAX_CANVAS_DESCRIPTION_LENGTH} characters
            </div>
          </Field>
        </div>
      </DialogBody>

      <DialogActions>
        <Button
          color="blue"
          onClick={handleSubmit}
          disabled={!canvasName.trim() || isLoading || !!nameError}
          className="flex items-center gap-2"
        >
          {isLoading ? 'Creating...' : 'Create'}
        </Button>
      </DialogActions>

    </Dialog>
  )
}