import { useEffect, useState } from "react";
import { Trash2 } from "lucide-react";
import { Dialog, DialogActions, DialogBody, DialogDescription, DialogTitle } from "@/components/Dialog/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";

interface OrganizationDeleteDialogProps {
  open: boolean;
  organizationName: string;
  canDelete: boolean;
  isDeleting: boolean;
  onClose: () => void;
  onConfirm: () => Promise<void>;
}

export function OrganizationDeleteDialog({
  open,
  organizationName,
  canDelete,
  isDeleting,
  onClose,
  onConfirm,
}: OrganizationDeleteDialogProps) {
  const [confirmation, setConfirmation] = useState("");
  const nameMatches = organizationName.length > 0 && confirmation.trim() === organizationName;

  useEffect(() => {
    if (!open) {
      setConfirmation("");
    }
  }, [open]);

  return (
    <Dialog open={open} onClose={onClose} size="lg" className="text-left">
      <DialogTitle className="text-gray-800 dark:text-red-100">{`Delete "${organizationName}"?`}</DialogTitle>
      <DialogDescription className="space-y-2 text-sm text-gray-800 dark:text-gray-400">
        <p>SuperPlane permanently removes all workspaces, members, and settings in this organization.</p>
        <p>If this organization has a Business plan, SuperPlane cancels it.</p>
        <p>The plan stays active until the end of the current billing period.</p>
      </DialogDescription>
      <DialogBody>
        <div className="space-y-2">
          <Label htmlFor="organization-delete-confirmation">{`Type "${organizationName}" to confirm`}</Label>
          <Input
            id="organization-delete-confirmation"
            data-testid="organization-delete-confirmation-input"
            value={confirmation}
            onChange={(event) => setConfirmation(event.target.value)}
            autoComplete="off"
            disabled={!canDelete || isDeleting}
          />
        </div>
      </DialogBody>
      <DialogActions>
        <LoadingButton
          variant="destructive"
          onClick={() => {
            if (!nameMatches) return;
            void (async () => {
              try {
                await onConfirm();
                onClose();
              } catch {
                return;
              }
            })();
          }}
          disabled={!canDelete || !nameMatches}
          loading={isDeleting}
          loadingText="Deleting..."
          className="flex items-center gap-2"
          data-testid="organization-delete-confirm-button"
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
