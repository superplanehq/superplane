import { useEffect, useState } from "react";
import { Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
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
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (isDeleting && !next) {
          return;
        }
        if (!next) {
          onClose();
        }
      }}
    >
      <DialogContent data-testid="organization-delete-dialog">
        <DialogHeader>
          <DialogTitle>{`Delete "${organizationName}"?`}</DialogTitle>
          <DialogDescription>
            You lose access now. SuperPlane keeps this organization for at least 30 days, then removes it.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-2 text-sm text-muted-foreground">
          <p>Removal can take longer when workspaces or integrations remain.</p>
          <p>If this organization has a Business plan, SuperPlane cancels it.</p>
          <p>The plan stays active until the end of the current billing period.</p>
        </div>
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
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose} disabled={isDeleting}>
            Keep organization
          </Button>
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
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
