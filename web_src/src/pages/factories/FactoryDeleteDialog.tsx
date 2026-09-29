import { Dialog, DialogActions, DialogDescription, DialogTitle } from "@/components/Dialog/dialog";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { Trash2 } from "lucide-react";

interface FactoryDeleteDialogProps {
  open: boolean;
  factoryName: string;
  title?: string;
  description?: string;
  canDelete: boolean;
  isDeleting: boolean;
  onClose: () => void;
  onConfirm: () => Promise<void>;
}

export function FactoryDeleteDialog({
  open,
  factoryName,
  title,
  description,
  canDelete,
  isDeleting,
  onClose,
  onConfirm,
}: FactoryDeleteDialogProps) {
  return (
    <Dialog open={open} onClose={onClose} size="lg" className="text-left">
      <DialogTitle className="text-gray-800 dark:text-red-100">{title ?? `Delete "${factoryName}"?`}</DialogTitle>
      <DialogDescription className="text-sm text-gray-800 dark:text-gray-400">
        {description ?? "This cannot be undone. Are you sure you want to continue?"}
      </DialogDescription>
      <DialogActions>
        <LoadingButton
          variant="destructive"
          onClick={() => {
            void (async () => {
              try {
                await onConfirm();
                onClose();
              } catch {
                // Toast handled by caller; keep dialog open for retry.
              }
            })();
          }}
          disabled={!canDelete}
          loading={isDeleting}
          loadingText="Deleting..."
          className="flex items-center gap-2"
          data-testid="factory-delete-confirm-button"
        >
          <Trash2 size={16} />
          Delete
        </LoadingButton>
        <Button variant="outline" onClick={onClose} disabled={isDeleting}>
          Cancel
        </Button>
      </DialogActions>
    </Dialog>
  );
}
