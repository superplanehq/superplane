import { cn } from "@/lib/utils";
import { Workflow } from "lucide-react";

import { columnAutomationsNeedRepair, type ColumnAutomation, type ColumnKey } from "../lib/columnAutomations";
import { ColumnAutomationsPopup, type ColumnAutomationRowAction } from "./ColumnAutomationsPopup";

interface ColumnAutomationsIndicatorProps {
  title: string;
  count: number;
  needsRepair: boolean;
  testId: string;
  open?: boolean;
}

/** Compact count in the column header. Opens the Automations menu. */
export function ColumnAutomationsIndicator({
  title,
  count,
  needsRepair,
  testId,
  open = false,
}: ColumnAutomationsIndicatorProps) {
  return (
    <button
      type="button"
      aria-label={`${title} automations`}
      title={`${title} automations`}
      data-testid={testId}
      className={cn(
        "relative flex h-6 shrink-0 items-center gap-1 rounded-md px-1.5 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground",
        open && "bg-accent text-foreground",
      )}
    >
      <Workflow className="size-3.5" aria-hidden />
      <span className="text-[11px] font-medium tabular-nums">{count}</span>
      {needsRepair ? (
        <span
          className="absolute -right-0.5 -top-0.5 size-1.5 rounded-full bg-amber-500"
          data-testid={`${testId}-needs-repair`}
          aria-hidden
        />
      ) : null}
    </button>
  );
}

export function ColumnAutomationsHeaderSlot({
  title,
  columnKey,
  automations,
  open = false,
  onOpen,
  onClose,
  onAdd,
  onRowAction,
  addDisabled,
  testId,
}: {
  title: string;
  columnKey: ColumnKey;
  automations?: ColumnAutomation[];
  open?: boolean;
  onOpen?: () => void;
  onClose?: () => void;
  onAdd?: () => void;
  onRowAction?: (automation: ColumnAutomation, action: ColumnAutomationRowAction) => void;
  addDisabled?: boolean;
  testId: string;
}) {
  if (!onOpen || !onClose || !onAdd || !onRowAction || !automations) {
    return null;
  }
  return (
    <ColumnAutomationsPopup
      columnTitle={title}
      columnKey={columnKey}
      automations={automations}
      open={open}
      addDisabled={addDisabled}
      onOpen={onOpen}
      onClose={onClose}
      onAdd={onAdd}
      onRowAction={onRowAction}
      trigger={
        <ColumnAutomationsIndicator
          title={title}
          count={automations.length}
          needsRepair={columnAutomationsNeedRepair(automations)}
          testId={testId}
          open={open}
        />
      }
    />
  );
}
