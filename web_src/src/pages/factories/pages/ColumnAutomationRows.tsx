import { GripVertical, Loader2, Play, Plus, Settings2 } from "lucide-react";
import { useContext } from "react";
import { AutomationRowContext } from "./AutomationRowContext";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

import { columnAutomationHeadline } from "../lib/columnAutomationHeadline";
import { COLUMN_AUTOMATIONS_COPY, type ColumnAutomation } from "../lib/columnAutomations";
import { ColumnAutomationGlyph, type ColumnAutomationRowAction } from "./ColumnAutomationsPopup";

const ROW_CLASSNAME =
  "flex h-6 w-full min-w-0 items-center gap-1.5 rounded-md pr-1.5 pl-0 text-left text-[12px] leading-4";
const ROW_HOVER_CLASSNAME = "transition-colors hover:bg-black/10 dark:hover:bg-black/20";
// Keep SVGs out of the button's direct children to avoid icon-dependent padding.
const ICON_SLOT_CLASSNAME = "flex size-3.5 shrink-0 items-center justify-center";

/**
 * Header rows that name the automations behind a column. Public boards can
 * reserve a shared rowCount to align card lists with blank slots.
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
    <Button
      variant="ghost"
      size="xs"
      type="button"
      data-testid={`${testId}-empty`}
      aria-label={`${COLUMN_AUTOMATIONS_COPY.addLabel} for ${title}`}
      onClick={onAdd}
      className={cn(ROW_CLASSNAME, ROW_HOVER_CLASSNAME, "font-normal text-muted-foreground hover:text-foreground")}
    >
      <span className={ICON_SLOT_CLASSNAME}>
        <Plus className="size-3.5" aria-hidden />
      </span>
      <span className="min-w-0 flex-1 truncate">{COLUMN_AUTOMATIONS_COPY.addLabel}</span>
    </Button>
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
  const board = useContext(AutomationRowContext);
  const run = board.onRun ?? onRun;
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
      <span className={ICON_SLOT_CLASSNAME}>
        <ColumnAutomationGlyph automation={automation} className="size-3.5" />
      </span>
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
  const showRun = Boolean(run && automation.canRunNow);
  const runningId = board.runningAutomationId ?? runningAutomationId;
  const starting = Boolean(runningId && runningId === automation.canvasId);

  return (
    <div
      role="listitem"
      title={`${automation.name}: ${automation.trigger} → ${automation.action}`}
      className={className}
      data-testid={`${testId}-row-${automation.id}`}
      onClick={onRowAction ? () => onRowAction(automation, "settings") : undefined}
    >
      {board.move && (
        <Button
          type="button"
          draggable
          variant="ghost"
          size="icon-xs"
          className="size-4 shrink-0 cursor-grab"
          aria-label={`Move ${automation.name}`}
          title="Drag to another stage, or use the left and right arrow keys."
          data-automation-handle={automation.id}
          onClick={(event) => event.stopPropagation()}
          onDragStart={(event) => {
            event.stopPropagation();
            event.dataTransfer.effectAllowed = "move";
            event.dataTransfer.setData("application/x-superplane-automation", automation.id);
            board.move?.start(automation);
          }}
          onDragEnd={() => board.move?.end()}
          onKeyDown={(event) => {
            if (event.key === "ArrowLeft" || event.key === "ArrowRight") {
              event.preventDefault();
              board.move?.keyboard(automation, event.key === "ArrowLeft" ? -1 : 1);
            }
          }}
        >
          <GripVertical aria-hidden />
        </Button>
      )}
      {onRowAction ? (
        <Button
          variant="ghost"
          size="xs"
          type="button"
          aria-label={`${COLUMN_AUTOMATIONS_COPY.rowMenu}: ${automation.name}`}
          className="h-auto min-w-0 flex-1 justify-start gap-1.5 rounded-none p-0 text-left text-[12px] font-normal hover:bg-transparent dark:hover:bg-transparent"
        >
          {details}
        </Button>
      ) : (
        details
      )}
      {showRun ? (
        // Disabled buttons ignore pointer events, so stop clicks on their enclosing area.
        <span className="flex shrink-0" onClick={(event) => event.stopPropagation()}>
          <Button
            variant="ghost"
            size="icon-xs"
            type="button"
            aria-label={`Run ${automation.name}`}
            disabled={board.runPending ?? runPending}
            aria-busy={starting}
            data-testid={`${testId}-run-${automation.id}`}
            onClick={(event) => {
              event.stopPropagation();
              run?.(automation);
            }}
            className="size-4 shrink-0 p-0 text-muted-foreground hover:text-foreground"
          >
            {starting ? (
              <Loader2 className="size-3.5 animate-spin" aria-hidden />
            ) : (
              <Play className="size-3.5" aria-hidden />
            )}
          </Button>
        </span>
      ) : null}
      {onRowAction ? (
        <Button
          variant="ghost"
          size="icon-xs"
          type="button"
          tabIndex={-1}
          aria-hidden
          onClick={(event) => {
            event.stopPropagation();
            onRowAction(automation, "settings");
          }}
          className="size-4 shrink-0 p-0 text-muted-foreground"
        >
          <Settings2 className="size-3.5" aria-hidden data-testid={`${testId}-settings-${automation.id}`} />
        </Button>
      ) : null}
    </div>
  );
}
