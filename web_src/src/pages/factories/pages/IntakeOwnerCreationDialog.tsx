import { useState } from "react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { getApiErrorMessage } from "@/lib/errors";
import { IntakeSetupDefaultOwner } from "./IntakeDefaultOwnerField";
import { useIntakeOwnerDraft } from "./useIntakeOwnerDraft";

export function IntakeOwnerCreationDialog({
  organizationId,
  factoryId,
  onClose,
  onCreate,
}: {
  organizationId: string;
  factoryId: string;
  onClose: () => void;
  onCreate: (defaultOwnerId: string) => Promise<void>;
}) {
  const owner = useIntakeOwnerDraft(organizationId, factoryId);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string>();
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !pending) onClose();
      }}
    >
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Set default owner</DialogTitle>
          <DialogDescription>Select who owns tasks from this intake.</DialogDescription>
        </DialogHeader>
        <IntakeSetupDefaultOwner owner={owner} disabled={pending} />
        {error ? (
          <p role="alert" className="text-sm text-destructive">
            {error}
          </p>
        ) : null}
        <DialogFooter>
          <Button variant="outline" disabled={pending} onClick={onClose}>
            Cancel
          </Button>
          <Button
            disabled={!owner.value || pending}
            onClick={async () => {
              setPending(true);
              setError(undefined);
              try {
                await onCreate(owner.value);
                onClose();
              } catch (cause) {
                setError(getApiErrorMessage(cause, "SuperPlane could not create the intake. Try again."));
              } finally {
                setPending(false);
              }
            }}
          >
            {pending ? "Creating…" : "Create intake"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
