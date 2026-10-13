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
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { Ellipsis } from "lucide-react";

import { type SplitConfigDeleteCopy } from "./splitConfigDeleteCopy";

const HEADER_ICON_BUTTON =
  "flex h-6 w-6 items-center justify-center rounded-full text-foreground hover:bg-slate-950/5 dark:hover:bg-white/10";

export function MergeConfidenceActionsMenu({ label, onDelete }: { label: string; onDelete: () => void }) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          className={HEADER_ICON_BUTTON}
          aria-label="More actions"
          data-testid="merge-confidence-config-actions"
        >
          <Ellipsis className="h-4 w-4" aria-hidden />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-48">
        <DropdownMenuItem
          className="text-destructive focus:text-destructive"
          onSelect={onDelete}
          data-testid="merge-confidence-config-delete"
        >
          {label}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

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
