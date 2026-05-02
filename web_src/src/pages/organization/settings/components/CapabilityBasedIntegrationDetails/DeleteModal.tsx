import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { LoadingButton } from "@/components/ui/loading-button";

export interface DeleteModalProps {
  open: boolean;
  integrationName?: string;
  canDeleteIntegrations: boolean;
  isDeleting: boolean;
  hasDeleteError: boolean;
  onDelete: () => void;
  onClose: () => void;
}

export function DeleteModal({
  open,
  integrationName,
  canDeleteIntegrations,
  isDeleting,
  hasDeleteError,
  onDelete,
  onClose,
}: DeleteModalProps) {
  return (
    <Dialog
      open={open}
      onOpenChange={(nextOpen) => {
        if (!nextOpen && !isDeleting) {
          onClose();
        }
      }}
    >
      <DialogContent showCloseButton={false} className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Delete {integrationName || "integration"}?</DialogTitle>
          <DialogDescription>This cannot be undone. All data will be permanently deleted.</DialogDescription>
        </DialogHeader>
        <DialogFooter className="flex-row justify-start gap-3 sm:justify-start">
          <LoadingButton
            color="blue"
            onClick={onDelete}
            disabled={!canDeleteIntegrations}
            loading={isDeleting}
            loadingText="Deleting..."
            className="bg-red-600 hover:bg-red-700 dark:bg-red-600 dark:hover:bg-red-700"
          >
            Delete
          </LoadingButton>
          <Button variant="outline" onClick={onClose} disabled={isDeleting}>
            Cancel
          </Button>
        </DialogFooter>
        {hasDeleteError && (
          <div className="mt-4 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md">
            <p className="text-sm text-red-800 dark:text-red-200">Failed to delete integration. Please try again.</p>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
