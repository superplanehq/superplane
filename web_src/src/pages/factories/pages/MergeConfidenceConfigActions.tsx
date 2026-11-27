import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/ui/alertDialog";

import { type SplitConfigDeleteCopy } from "./splitConfigDeleteCopy";

export function MergeConfidenceDeleteDialog({
  open,
  pending,
  error,
  copy,
  onOpenChange,
  onConfirm,
}: {
  open: boolean;
  pending: boolean;
  error?: string;
  copy: SplitConfigDeleteCopy;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => Promise<void> | void;
}) {
  return (
    <AlertDialog
      open={open}
      onOpenChange={(next) => {
        if (!pending) {
          onOpenChange(next);
        }
      }}
    >
      <AlertDialogContent data-testid="merge-confidence-config-delete-dialog">
        <AlertDialogHeader>
          <AlertDialogTitle>{copy.title}</AlertDialogTitle>
          <AlertDialogDescription>{copy.description}</AlertDialogDescription>
          {error ? (
            <p className="text-sm text-destructive" data-testid="merge-confidence-config-delete-error">
              {error}
            </p>
          ) : null}
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>{copy.keepLabel}</AlertDialogCancel>
          <AlertDialogAction
            className="bg-destructive text-white hover:bg-destructive/90"
            disabled={pending}
            data-testid="merge-confidence-config-delete-confirm"
            onClick={(event) => {
              event.preventDefault();
              void confirmDelete(onConfirm);
            }}
          >
            {pending ? copy.pendingLabel : copy.confirmLabel}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

async function confirmDelete(onConfirm: () => Promise<void> | void) {
  try {
    await onConfirm();
  } catch {
    // The parent supplies the actionable error message.
  }
}
