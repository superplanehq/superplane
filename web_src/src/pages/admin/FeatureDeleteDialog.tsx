import { useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { billingCatalogRequest, type FeatureDefinition } from "./billingPlansApi";

export function FeatureDeleteDialog({
  feature,
  returnFocus,
  survivorFocus,
  onDeleted,
  onClose,
}: {
  feature: FeatureDefinition;
  returnFocus: HTMLElement | null;
  survivorFocus: HTMLElement | null;
  onDeleted: () => Promise<void>;
  onClose: () => void;
}) {
  const [deleting, setDeleting] = useState(false);
  const [error, setError] = useState("");
  const succeeded = useRef(false);
  const remove = async () => {
    setDeleting(true);
    setError("");
    try {
      await billingCatalogRequest(`feature-definitions/${encodeURIComponent(feature.key)}`, "DELETE");
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "Failed to delete feature");
      setDeleting(false);
      return;
    }
    succeeded.current = true;
    await onDeleted();
    onClose();
  };
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !deleting) onClose();
      }}
    >
      <DialogContent
        showCloseButton={!deleting}
        onCloseAutoFocus={(event) => {
          event.preventDefault();
          // The row trigger is removed after a successful delete, so focus a control that remains.
          (succeeded.current ? survivorFocus : returnFocus)?.focus();
        }}
      >
        <DialogHeader>
          <DialogTitle>Delete “{feature.label}”?</DialogTitle>
          <DialogDescription>
            This removes the feature from all plans and organization grants. Organizations lose access immediately. This
            cannot be undone.
          </DialogDescription>
        </DialogHeader>
        {error && (
          <p role="alert" className="text-sm text-destructive">
            {error}
          </p>
        )}
        <DialogFooter className="border-t pt-4">
          <Button variant="outline" disabled={deleting} onClick={() => onClose()}>
            Keep feature
          </Button>
          <Button variant="destructive" disabled={deleting} onClick={() => void remove()}>
            {deleting ? "Deleting..." : "Delete feature"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
