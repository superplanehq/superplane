import { Dialog, DialogActions, DialogDescription, DialogTitle } from "@/components/Dialog/dialog";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { Trash2 } from "lucide-react";

interface OnboardingCancelDialogProps {
  open: boolean;
  canDelete: boolean;
  isDeleting: boolean;
  onClose: () => void;
  onConfirm: () => Promise<void>;
}

// Onboarding gives the workspace a placeholder name the user never chose, so the
// confirmation must describe the action instead of naming the workspace.
export function OnboardingCancelDialog({
  open,
  canDelete,
  isDeleting,
  onClose,
  onConfirm,
}: OnboardingCancelDialogProps) {
  return (
    <Dialog open={open} onClose={onClose} size="lg" className="text-left">
      <DialogTitle className="text-gray-800 dark:text-red-100">Cancel setup?</DialogTitle>
      <DialogDescription className="text-sm text-gray-800 dark:text-gray-400">
        This deletes the workspace and the setup progress. This cannot be undone.
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
          data-testid="onboarding-cancel-confirm-button"
        >
          <Trash2 size={16} />
          Delete workspace
        </LoadingButton>
        <Button variant="outline" onClick={onClose} disabled={isDeleting}>
          Continue setup
        </Button>
      </DialogActions>
    </Dialog>
  );
}
