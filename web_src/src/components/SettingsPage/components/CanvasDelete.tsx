import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Heading } from '../../Heading/heading'
import { Text } from '../../Text/text'
import { Button } from '../../Button/button'
import { MaterialSymbol } from '../../MaterialSymbol/material-symbol'
import { Input } from '../../Input/input'
import {
  Dialog,
  DialogTitle,
  DialogDescription,
  DialogBody,
  DialogActions
} from '../../Dialog/dialog'
import { useMutation } from '@tanstack/react-query'

interface CanvasDeleteProps {
  canvasId: string
  canvasName: string
  organizationId: string
}

export function CanvasDelete({ canvasId, canvasName, organizationId }: CanvasDeleteProps) {
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false)
  const [confirmationText, setConfirmationText] = useState('')
  const navigate = useNavigate()

  // Mock delete function - replace with actual API call when available
  const mockDeleteCanvas = async (): Promise<void> => {
    // Simulate API call delay
    await new Promise(resolve => setTimeout(resolve, 2000))
    console.log('Mock: Deleting canvas', canvasId, 'from organization', organizationId)
    // In real implementation, this would call the actual API
    // return await superplaneDeleteCanvas({ path: { id: canvasId }, query: { organizationId } })
  }

  const deleteCanvasMutation = useMutation({
    mutationFn: mockDeleteCanvas,
    onSuccess: () => {
      // Navigate back to organization page after successful deletion
      navigate(`/organization/${organizationId}`)
    }
  })

  const handleDeleteClick = () => {
    setIsDeleteModalOpen(true)
  }

  const handleCloseModal = () => {
    setIsDeleteModalOpen(false)
    setConfirmationText('')
  }

  const handleConfirmDelete = async () => {
    if (confirmationText === canvasName) {
      try {
        await deleteCanvasMutation.mutateAsync()
      } catch (err) {
        console.error('Error deleting canvas:', err)
      }
    }
  }

  const isConfirmationValid = confirmationText === canvasName

  return (
    <div className="space-y-6">
      <Heading level={2}>Delete Canvas</Heading>
      
      <div className="bg-white dark:bg-zinc-800 rounded-lg border border-red-200 dark:border-red-800 p-6">
        <div className="flex items-start gap-4">
          <div className="flex-shrink-0">
            <MaterialSymbol name="warning" className="text-red-500" size="lg" />
          </div>
          <div className="flex-1">
            <Heading level={3} className="text-lg text-red-900 dark:text-red-100 mb-2">
              Delete "{canvasName}"
            </Heading>
            <div className="space-y-3 mb-6">
              <Text className="text-zinc-600 dark:text-zinc-400">
                Once you delete a canvas, there is no going back. This action will permanently delete:
              </Text>
              <ul className="list-disc list-inside space-y-1 text-sm text-zinc-600 dark:text-zinc-400 ml-4">
                <li>All stages and workflow configurations</li>
                <li>Historical execution data and logs</li>
                <li>Associated secrets and environment variables</li>
                <li>Canvas permissions and member access</li>
                <li>All connected integrations and webhooks</li>
              </ul>
              <Text className="text-zinc-600 dark:text-zinc-400 font-medium">
                This action cannot be undone.
              </Text>
            </div>
            <Button color="red" onClick={handleDeleteClick}>
              <MaterialSymbol name="delete" size="sm" />
              Delete Canvas
            </Button>
          </div>
        </div>
      </div>

      {/* Confirmation Dialog */}
      <Dialog open={isDeleteModalOpen} onClose={handleCloseModal} size="lg">
        <DialogTitle className="text-red-900 dark:text-red-100">
          Delete Canvas
        </DialogTitle>
        <DialogDescription>
          This action cannot be undone. Please confirm that you want to permanently delete this canvas.
        </DialogDescription>
        
        <DialogBody>
          <div className="space-y-4">
            <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
              <div className="flex items-center gap-3 mb-3">
                <MaterialSymbol name="warning" className="text-red-500" size="sm" />
                <Text className="font-medium text-red-900 dark:text-red-100">
                  This will permanently delete the canvas "{canvasName}"
                </Text>
              </div>
              <Text className="text-sm text-red-700 dark:text-red-300">
                All associated data, stages, execution history, secrets, and configurations will be permanently removed.
              </Text>
            </div>
            
            <div>
              <label htmlFor="confirm-deletion" className="block text-sm font-medium text-zinc-900 dark:text-zinc-100 mb-2">
                Type <span className="font-mono bg-zinc-100 dark:bg-zinc-800 px-2 py-1 rounded">{canvasName}</span> to confirm deletion:
              </label>
              <Input
                id="confirm-deletion"
                type="text"
                placeholder={`Type "${canvasName}" to confirm`}
                value={confirmationText}
                onChange={(e) => setConfirmationText(e.target.value)}
                className="w-full"
                autoComplete="off"
              />
            </div>
          </div>
        </DialogBody>

        <DialogActions>
          <Button plain onClick={handleCloseModal}>
            Cancel
          </Button>
          <Button 
            color="red" 
            onClick={handleConfirmDelete}
            disabled={!isConfirmationValid || deleteCanvasMutation.isPending}
            className="flex items-center gap-2"
          >
            <MaterialSymbol name="delete" size="sm" />
            {deleteCanvasMutation.isPending ? 'Deleting...' : 'Delete Canvas'}
          </Button>
        </DialogActions>
      </Dialog>
    </div>
  )
}