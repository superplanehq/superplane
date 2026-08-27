import { Button } from "@/components/ui/button";
import { Loader2 } from "lucide-react";
import { useState } from "react";

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

import {
  splitRunCloseNeedsConfirm,
  type SplitRunFooter,
  type SplitRunFooterAction,
  type SplitRunFooterActionKind,
  type SplitRunStopChoice,
} from "./splitRunFooter";

type CloseKind = Extract<SplitRunFooterActionKind, "reject" | "approve">;

function visibleHeaderActions(
  actions: SplitRunFooter["actions"],
  onStop?: (choice: SplitRunStopChoice) => void | Promise<void>,
  onReject?: () => void | Promise<void>,
) {
  return actions.filter((action) => {
    if (action.kind === "reopen" || action.kind === "approve") {
      return Boolean(onStop);
    }
    if (action.kind === "reject") {
      return Boolean(onReject) || Boolean(onStop);
    }
    return true;
  });
}

/**
 * Work-order actions in the popup header, left of Close.
 */
export function SplitRunHeaderActions({
  footer,
  onStart,
  onStop,
  onReject,
  startBusy = false,
  stopBusy = false,
  startDisabled = false,
}: {
  footer: SplitRunFooter;
  onStart?: () => void | Promise<void>;
  onStop?: (choice: SplitRunStopChoice) => void | Promise<void>;
  onReject?: () => void | Promise<void>;
  startBusy?: boolean;
  stopBusy?: boolean;
  startDisabled?: boolean;
}) {
  const [pendingKind, setPendingKind] = useState<CloseKind | null>(null);
  const actions = visibleHeaderActions(footer.actions, onStop, onReject);
  if (actions.length === 0) {
    return null;
  }

  const runClose = (kind: CloseKind) => {
    if (kind === "approve") {
      void onStop?.("completed");
      return;
    }
    if (footer.kind === "draft") {
      void onReject?.();
      return;
    }
    void onStop?.("canceled");
  };

  const onActionClick = (action: SplitRunFooterAction) => {
    if (action.kind === "start") {
      void onStart?.();
      return;
    }
    if (action.kind === "reopen") {
      void onStop?.("reopen");
      return;
    }
    if (action.kind === "reject" || action.kind === "approve") {
      if (splitRunCloseNeedsConfirm(footer.kind)) {
        setPendingKind(action.kind);
        return;
      }
      runClose(action.kind);
    }
  };

  return (
    <div className="flex shrink-0 items-center gap-2" data-testid="split-run-header-actions">
      {actions.map((action) => (
        <HeaderAction
          key={action.id}
          action={action}
          startBusy={startBusy}
          stopBusy={stopBusy}
          startDisabled={startDisabled}
          onClick={() => onActionClick(action)}
        />
      ))}
      <AlertDialog open={pendingKind !== null} onOpenChange={(open) => !open && setPendingKind(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Stop running automations?</AlertDialogTitle>
            <AlertDialogDescription>
              This action stops all running automations on this work order. Then it closes the work order.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (pendingKind) {
                  runClose(pendingKind);
                }
              }}
            >
              {pendingKind === "approve" ? "Approve" : "Reject"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}

function HeaderAction({
  action,
  startBusy,
  stopBusy,
  startDisabled,
  onClick,
}: {
  action: SplitRunFooterAction;
  startBusy: boolean;
  stopBusy: boolean;
  startDisabled: boolean;
  onClick: () => void;
}) {
  const primary = action.emphasis === "primary";
  const busy = action.kind === "start" ? startBusy : stopBusy;
  const disabled = action.kind === "start" ? startDisabled || startBusy : stopBusy;

  return (
    <Button
      type="button"
      size="sm"
      variant={primary ? "default" : "outline"}
      disabled={disabled}
      onClick={onClick}
      data-testid={primary ? "split-run-review-cta" : `split-run-footer-${action.id}`}
    >
      {busy ? <Loader2 className="size-3.5 animate-spin" aria-hidden /> : null}
      {action.label}
    </Button>
  );
}
