import { Trash2 } from "lucide-react";
import { Dialog, DialogActions, DialogDescription, DialogTitle } from "@/components/Dialog/dialog";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";

type DeleteDraftBranchDialogProps = {
  open: boolean;
  draftName: string;
  deletePending: boolean;
  onConfirm: () => void;
  onClose: () => void;
};

export function DeleteDraftBranchDialog({
  open,
  draftName,
  deletePending,
  onConfirm,
  onClose,
}: DeleteDraftBranchDialogProps) {
  return (
    <Dialog open={open} onClose={onClose} size="lg" className="text-left">
      <DialogTitle className="text-gray-800 dark:text-red-100">Delete "{draftName}"?</DialogTitle>
      <DialogDescription className="text-sm text-gray-800 dark:text-gray-400">
        This cannot be undone. Are you sure you want to continue?
      </DialogDescription>
      <DialogActions>
        <LoadingButton
          variant="destructive"
          onClick={onConfirm}
          loading={deletePending}
          loadingText="Deleting..."
          className="flex items-center gap-2"
        >
          <Trash2 size={16} />
          Delete
        </LoadingButton>
        <Button variant="outline" onClick={onClose}>
          Cancel
        </Button>
      </DialogActions>
    </Dialog>
  );
}
