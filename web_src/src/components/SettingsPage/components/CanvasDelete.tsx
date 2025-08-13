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
import { useDeleteCanvas } from '@/hooks/useOrganizationData'

interface CanvasDeleteProps {
  canvasId: string
  organizationId: string
}

export function CanvasDelete({ canvasId, organizationId }: CanvasDeleteProps) {
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false)
  const [confirmationText, setConfirmationText] = useState('')
  const navigate = useNavigate()
  const deleteCanvasMutation = useDeleteCanvas(organizationId)

  const handleDeleteClick = () => {
    setIsDeleteModalOpen(true)
  }

  const handleCloseModal = () => {
    setIsDeleteModalOpen(false)
    setConfirmationText('')
  }

  const handleConfirmDelete = async () => {
    if (confirmationText === 'DELETE') {
      await deleteCanvasMutation.mutateAsync({ canvasId })
      navigate(`/organization/${organizationId}`)
    }
    handleCloseModal()
  }

  return (
    <div className="max-w-4xl mx-auto space-y-6">
      <Heading level={3} className="text-left text-black dark:text-white sm:text-sm">Danger Zone</Heading>

      <div className="text-left bg-red-50 dark:bg-zinc-800 rounded-lg border border-red-200 dark:border-red-800 p-6">
        <div className="flex items-start gap-4">
          <div className="flex-shrink-0">
            <MaterialSymbol name="warning" className="text-red-500" size="lg" />
          </div>
          <div className="flex-1">
            <Heading level={3} className="sm:text-sm text-black dark:text-white mb-2">
              Delete canvas
            </Heading>
            <div className="space-y-3 mb-6">
              <Text className="text-zinc-600 dark:text-zinc-400">
                Once you delete this canvas, there is no going back. This action cannot be undone. All workflows, configurations, and associated data will be permanently removed
              </Text>
              <Text>This will permanently delete:</Text>
              <ul className="list-disc list-inside space-y-1 text-sm text-red-600 dark:text-zinc-400 ml-4">
                <li>All workflow stages and configurations</li>
                <li>All secrets and environment variables</li>
                <li>All integration connections</li>
                <li>All execution history and logs</li>
                <li>All team member access</li>
              </ul>
            </div>
            <Button color="red" onClick={handleDeleteClick}>
              <MaterialSymbol name="delete" size="sm" />
              Delete Canvas
            </Button>
          </div>
        </div>
      </div>

      {/* Confirmation Dialog */}
      <Dialog className="text-left" open={isDeleteModalOpen} onClose={handleCloseModal} size="lg">
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
                  This will permanently delete the canvas
                </Text>
              </div>
              <Text className="text-sm text-red-700 dark:text-red-300">
                All associated data, stages, execution history, secrets, and configurations will be permanently removed.
              </Text>
            </div>

            <div>
              <label htmlFor="confirm-deletion" className="block text-sm font-medium text-zinc-900 dark:text-zinc-100 mb-2">
                Type <span className="font-mono bg-zinc-100 dark:bg-zinc-800 px-2 py-1 rounded">DELETE</span> to confirm deletion:
              </label>
              <Input
                id="confirm-deletion"
                type="text"
                placeholder="Type DELETE to confirm"
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
            disabled={confirmationText !== 'DELETE' || deleteCanvasMutation.isPending}
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