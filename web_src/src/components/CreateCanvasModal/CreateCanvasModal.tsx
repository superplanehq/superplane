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

interface CreateCanvasModalProps {
  isOpen: boolean
  onClose: () => void
  onSubmit: (data: { name: string; description?: string }) => Promise<void>
  isLoading?: boolean
}

export function CreateCanvasModal({
  isOpen,
  onClose,
  onSubmit,
  isLoading = false
}: CreateCanvasModalProps) {
  const [canvasName, setCanvasName] = useState('')
  const [canvasDescription, setCanvasDescription] = useState('')

  const handleClose = () => {
    setCanvasName('')
    setCanvasDescription('')
    onClose()
  }

  const handleSubmit = async () => {
    if (canvasName.trim()) {
      try {
        await onSubmit({
          name: canvasName.trim(),
          description: canvasDescription.trim() || undefined
        })

        // Reset form and close modal
        setCanvasName('')
        setCanvasDescription('')
        onClose()
      } catch (error) {
        console.error('Error creating canvas:', error)
      }
    }
  }

  return (
    <Dialog open={isOpen} onClose={handleClose} size="lg" className="text-left">
      <DialogTitle>Create New Canvas</DialogTitle>
      <DialogDescription className="text-sm">
        Create a new interactive canvas to build and manage your workflows.
      </DialogDescription>

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
              onChange={(e) => setCanvasName(e.target.value)}
              placeholder="Enter canvas name"
              className="w-full"
              autoFocus
            />
          </Field>

          {/* Canvas Description */}
          <Field>
            <Label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
              Description
            </Label>
            <Textarea
              value={canvasDescription}
              onChange={(e) => setCanvasDescription(e.target.value)}
              placeholder="Describe what this canvas will be used for (optional)"
              rows={3}
              className="w-full"
            />
          </Field>
        </div>
      </DialogBody>

      <DialogActions>
        <Button plain onClick={handleClose}>
          Cancel
        </Button>
        <Button
          color="blue"
          onClick={handleSubmit}
          disabled={!canvasName.trim() || isLoading}
          className="flex items-center gap-2"
        >
          <MaterialSymbol name="add" size="sm" />
          {isLoading ? 'Creating...' : 'Create Canvas'}
        </Button>
      </DialogActions>
    </Dialog>
  )
}