import { Button } from "@/components/ui/button";
import { Loader2 } from "lucide-react";

import { type SplitRunFooter, type SplitRunFooterAction, type SplitRunStopChoice } from "./splitRunFooter";

function visibleHeaderActions(
  actions: SplitRunFooter["actions"],
  onStop?: (choice: SplitRunStopChoice) => void | Promise<void>,
) {
  return actions.filter((action) => {
    if (action.kind === "reject" || action.kind === "approve") {
      return false;
    }
    if (action.kind === "reopen") {
      return Boolean(onStop);
    }
    return true;
  });
}

/**
 * Work-order actions in the popup header, left of Close.
 * Start stays on drafts. Reopen stays on closed orders.
 */
export function SplitRunHeaderActions({
  footer,
  onStart,
  onStop,
  startBusy = false,
  stopBusy = false,
  startDisabled = false,
}: {
  footer: SplitRunFooter;
  onStart?: () => void | Promise<void>;
  onStop?: (choice: SplitRunStopChoice) => void | Promise<void>;
  startBusy?: boolean;
  stopBusy?: boolean;
  startDisabled?: boolean;
}) {
  const actions = visibleHeaderActions(footer.actions, onStop);
  if (actions.length === 0) {
    return null;
  }

  const onActionClick = (action: SplitRunFooterAction) => {
    if (action.kind === "start") {
      void onStart?.();
      return;
    }
    if (action.kind === "reopen") {
      void onStop?.("reopen");
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
