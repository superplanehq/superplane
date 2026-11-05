import { Loader2, Play, Plus, Settings2 } from "lucide-react";

import { cn } from "@/lib/utils";

import { columnAutomationHeadline } from "../lib/columnAutomationHeadline";
import { COLUMN_AUTOMATIONS_COPY, type ColumnAutomation } from "../lib/columnAutomations";
import { ColumnAutomationGlyph, type ColumnAutomationRowAction } from "./ColumnAutomationsPopup";

const ROW_CLASSNAME = "flex h-6 w-full min-w-0 items-center gap-1.5 rounded-md px-1.5 text-left text-[12px] leading-4";
const ROW_HOVER_CLASSNAME = "transition-colors hover:bg-black/10 dark:hover:bg-black/20";

/**
 * Header rows that name the automations behind a column. Every column on
 * the board reserves the same `rowCount`, so short columns pad with blank
 * slots and the card lists start at one height.
 */
export function ColumnAutomationRows({
  title,
  automations,
  rowCount,
  onRowAction,
  onRun,
  runPending = false,
  runningAutomationId,
  onAdd,
  testId,
}: {
  title: string;
  automations: ColumnAutomation[];
  rowCount: number;
  onRowAction?: (automation: ColumnAutomation, action: ColumnAutomationRowAction) => void;
  onRun?: (automation: ColumnAutomation) => void;
  runPending?: boolean;
  runningAutomationId?: string;
  onAdd?: () => void;
  testId: string;
}) {
  const blankCount = Math.max(0, rowCount - Math.max(1, automations.length));

  return (
    <div role="list" aria-label={`${title} automations`} className="flex flex-col gap-0.5" data-testid={testId}>
      {automations.length === 0 ? (
        <EmptyAutomationRow title={title} onAdd={onAdd} testId={testId} />
      ) : (
        automations.map((automation) => (
          <ColumnAutomationRow
            key={automation.id}
            automation={automation}
            onRowAction={onRowAction}
            onRun={onRun}
            runPending={runPending}
            runningAutomationId={runningAutomationId}
            testId={testId}
          />
        ))
      )}
      {Array.from({ length: blankCount }, (_, index) => (
        <div key={`blank-${index}`} className={ROW_CLASSNAME} aria-hidden data-testid={`${testId}-blank`} />
      ))}
    </div>
  );
}

function EmptyAutomationRow({ title, onAdd, testId }: { title: string; onAdd?: () => void; testId: string }) {
  if (!onAdd) {
    return (
      <p className={cn(ROW_CLASSNAME, "text-muted-foreground")} data-testid={`${testId}-empty`}>
        {COLUMN_AUTOMATIONS_COPY.rowsEmpty}
      </p>
    );
  }

  return (
    <button
      type="button"
      data-testid={`${testId}-empty`}
      aria-label={`${COLUMN_AUTOMATIONS_COPY.addLabel} for ${title}`}
      onClick={onAdd}
      className={cn(ROW_CLASSNAME, ROW_HOVER_CLASSNAME, "text-muted-foreground hover:text-foreground")}
    >
      <Plus className="size-3.5 shrink-0" aria-hidden />
      <span className="min-w-0 flex-1 truncate">{COLUMN_AUTOMATIONS_COPY.addLabel}</span>
    </button>
  );
}

function ColumnAutomationRow({
  automation,
  onRowAction,
  onRun,
  runPending,
  runningAutomationId,
  testId,
}: {
  automation: ColumnAutomation;
  onRowAction?: (automation: ColumnAutomation, action: ColumnAutomationRowAction) => void;
  onRun?: (automation: ColumnAutomation) => void;
  runPending: boolean;
  runningAutomationId?: string;
  testId: string;
}) {
  const disabled = automation.health === "disabled";
  const needsRepair = automation.health === "needs-repair";
  const headline = columnAutomationHeadline(automation);
  const className = cn(
    ROW_CLASSNAME,
    "text-foreground",
    disabled && "line-through opacity-60",
    onRowAction && ROW_HOVER_CLASSNAME,
  );
  const details = (
    <>
      <ColumnAutomationGlyph automation={automation} className="size-3.5" />
      <span className="min-w-0 flex-1 truncate">{headline}</span>
      {needsRepair ? (
        <span
          className="size-1.5 shrink-0 rounded-full bg-amber-500"
          title={COLUMN_AUTOMATIONS_COPY.needsRepairLabel}
          data-testid={`${testId}-needs-repair-${automation.id}`}
        />
      ) : null}
    </>
  );
  const showRun = Boolean(onRun && automation.canRunNow);
  const starting = Boolean(runningAutomationId && runningAutomationId === automation.canvasId);

  return (
    <div
      role="listitem"
      title={`${automation.name}: ${automation.trigger} → ${automation.action}`}
      className={className}
      data-testid={`${testId}-row-${automation.id}`}
      onClick={onRowAction ? () => onRowAction(automation, "settings") : undefined}
    >
      {onRowAction ? (
        <button
          type="button"
          aria-label={`${COLUMN_AUTOMATIONS_COPY.rowMenu}: ${automation.name}`}
          className="flex min-w-0 flex-1 items-center gap-1.5 text-left"
        >
          {details}
        </button>
      ) : (
        details
      )}
      {showRun ? (
        <button
          type="button"
          aria-label={`Run ${automation.name}`}
          disabled={runPending}
          aria-busy={starting}
          data-testid={`${testId}-run-${automation.id}`}
          onClick={(event) => {
            event.stopPropagation();
            onRun?.(automation);
          }}
          className="flex size-4 shrink-0 items-center justify-center text-muted-foreground hover:text-foreground disabled:opacity-50"
        >
          {starting ? (
            <Loader2 className="size-3.5 animate-spin" aria-hidden />
          ) : (
            <Play className="size-3.5" aria-hidden />
          )}
        </button>
      ) : null}
      {onRowAction ? (
        <button
          type="button"
          tabIndex={-1}
          aria-hidden
          onClick={(event) => {
            event.stopPropagation();
            onRowAction(automation, "settings");
          }}
          className="flex size-4 shrink-0 items-center justify-center text-muted-foreground"
        >
          <Settings2 className="size-3.5" aria-hidden data-testid={`${testId}-settings-${automation.id}`} />
        </button>
      ) : null}
    </div>
  );
}
