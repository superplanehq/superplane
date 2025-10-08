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

interface CreateBlueprintModalProps {
  isOpen: boolean
  onClose: () => void
  onSubmit: (data: { name: string; description?: string }) => Promise<void>
  isLoading?: boolean
}

const MAX_BLUEPRINT_NAME_LENGTH = 50
const MAX_BLUEPRINT_DESCRIPTION_LENGTH = 200

export function CreateBlueprintModal({
  isOpen,
  onClose,
  onSubmit,
  isLoading = false
}: CreateBlueprintModalProps) {
  const [blueprintName, setBlueprintName] = useState('')
  const [blueprintDescription, setBlueprintDescription] = useState('')
  const [nameError, setNameError] = useState('')

  const handleClose = () => {
    setBlueprintName('')
    setBlueprintDescription('')
    setNameError('')
    onClose()
  }

  const handleSubmit = async () => {
    setNameError('')

    if (!blueprintName.trim()) {
      setNameError('Blueprint name is required')
      return
    }

    if (blueprintName.trim().length > MAX_BLUEPRINT_NAME_LENGTH) {
      setNameError(`Blueprint name must be ${MAX_BLUEPRINT_NAME_LENGTH} characters or less`)
      return
    }

    try {
      await onSubmit({
        name: blueprintName.trim(),
        description: blueprintDescription.trim() || undefined
      })

      // Reset form and close modal
      setBlueprintName('')
      setBlueprintDescription('')
      setNameError('')
      onClose()
    } catch (error) {
      console.error('Error creating blueprint:', error)
      const errorMessage = ((error as Error)?.message) || error?.toString() || 'Failed to create blueprint'

      showErrorToast(errorMessage)

      if (errorMessage.toLowerCase().includes('already') || errorMessage.toLowerCase().includes('exists')) {
        setNameError('A blueprint with this name already exists')
      }
    }
  }

  return (
    <Dialog open={isOpen} onClose={handleClose} size="lg" className="text-left relative">
      <DialogTitle>Create New Blueprint</DialogTitle>
      <DialogDescription className="text-sm">
        Create a new blueprint to define reusable workflow patterns using components.
      </DialogDescription>
      <button onClick={handleClose} className="absolute top-4 right-4">
        <MaterialSymbol name="close" size="sm" />
      </button>

      <DialogBody>
        <div className="space-y-6">
          {/* Blueprint Name */}
          <Field>
            <Label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
              Blueprint name *
            </Label>
            <Input
              type="text"
              value={blueprintName}
              onChange={(e) => {
                if (e.target.value.length <= MAX_BLUEPRINT_NAME_LENGTH) {
                  setBlueprintName(e.target.value)
                }
                if (nameError) {
                  setNameError('')
                }
              }}
              placeholder="Enter blueprint name"
              className={`w-full ${nameError ? 'border-red-500' : ''}`}
              autoFocus
              maxLength={MAX_BLUEPRINT_NAME_LENGTH}
            />
            <div className="text-xs text-zinc-500 dark:text-zinc-400 mt-1">
              {blueprintName.length}/{MAX_BLUEPRINT_NAME_LENGTH} characters
            </div>
            {nameError && (
              <div className="text-xs text-red-600 mt-1">
                {nameError}
              </div>
            )}
          </Field>

          {/* Blueprint Description */}
          <Field>
            <Label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
              Description
            </Label>
            <Textarea
              value={blueprintDescription}
              onChange={(e) => {
                if (e.target.value.length <= MAX_BLUEPRINT_DESCRIPTION_LENGTH) {
                  setBlueprintDescription(e.target.value)
                }
              }}
              placeholder="Describe what this blueprint will be used for (optional)"
              rows={3}
              className="w-full"
              maxLength={MAX_BLUEPRINT_DESCRIPTION_LENGTH}
            />
            <div className="text-xs text-zinc-500 dark:text-zinc-400 mt-1">
              {blueprintDescription.length}/{MAX_BLUEPRINT_DESCRIPTION_LENGTH} characters
            </div>
          </Field>
        </div>
      </DialogBody>

      <DialogActions>
        <Button
          color="blue"
          onClick={handleSubmit}
          disabled={!blueprintName.trim() || isLoading || !!nameError}
          className="flex items-center gap-2"
        >
          {isLoading ? 'Creating...' : 'Create'}
        </Button>
      </DialogActions>

    </Dialog>
  )
}
