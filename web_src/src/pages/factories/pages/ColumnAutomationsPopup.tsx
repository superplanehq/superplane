import { cn } from "@/lib/utils";
import { Popover, PopoverAnchor, PopoverContent } from "@/ui/popover";
import { Separator } from "@/ui/separator";
import { Plus, Workflow } from "lucide-react";
import { useEffect, useState, type ReactNode } from "react";

import {
  COLUMN_AUTOMATIONS_COPY,
  columnAutomationsEmptyCopy,
  type ColumnAutomation,
  type ColumnKey,
} from "../lib/columnAutomations";

export type ColumnAutomationRowAction = "settings" | "edit" | "disable" | "enable" | "remove";

/** Same hover surface for an automation row and Add automation. */
const AUTOMATION_ITEM_CLASS =
  "flex w-full cursor-pointer items-start gap-3 rounded-md px-2.5 py-2.5 text-left hover:bg-accent";

interface ColumnAutomationsPopupProps {
  columnTitle: string;
  columnKey: ColumnKey;
  automations: ColumnAutomation[];
  onClose: () => void;
  onAdd: () => void;
  onRowAction: (automation: ColumnAutomation, action: ColumnAutomationRowAction) => void;
  open?: boolean;
  onOpen?: () => void;
  addDisabled?: boolean;
  trigger?: ReactNode;
}

/** Large column menu, same chrome as the Backlog create dropdown. */
export function ColumnAutomationsPopup({
  columnTitle,
  columnKey,
  automations,
  onClose,
  onAdd,
  onRowAction,
  open = true,
  onOpen,
  addDisabled = false,
  trigger,
}: ColumnAutomationsPopupProps) {
  const title = `${columnTitle} automations`;
  const [menuOpen, setMenuOpen] = useState(open);

  useEffect(() => {
    setMenuOpen(open);
  }, [open]);

  return (
    <Popover
      open={menuOpen}
      onOpenChange={(next) => {
        setMenuOpen(next);
        if (next) {
          onOpen?.();
          return;
        }
        onClose();
      }}
    >
      {trigger ? (
        <PopoverAnchor asChild>
          <span
            className="inline-flex"
            onClick={() => {
              setMenuOpen(true);
              onOpen?.();
            }}
          >
            {trigger}
          </span>
        </PopoverAnchor>
      ) : null}
      <PopoverContent
        side="bottom"
        align="end"
        sideOffset={6}
        className="w-96 p-2"
        data-testid="column-automations-popup"
      >
        <h2 className="px-2.5 pb-1.5 pt-1 text-sm font-medium tracking-[-0.01em]">{title}</h2>
        {automations.length === 0 ? (
          <p className="px-2.5 py-4 text-[13px] text-muted-foreground" data-testid="column-automations-empty">
            {columnAutomationsEmptyCopy(columnTitle, columnKey)}
          </p>
        ) : (
          <ul className="flex flex-col" data-testid="column-automations-list">
            {automations.map((automation) => (
              <ColumnAutomationRow
                key={automation.id}
                automation={automation}
                onAction={(action) => onRowAction(automation, action)}
              />
            ))}
          </ul>
        )}
        <Separator className="my-1" data-testid="column-automations-divider" />
        <button
          type="button"
          onClick={onAdd}
          disabled={addDisabled}
          data-testid="column-automations-add"
          className={cn(AUTOMATION_ITEM_CLASS, "mt-0.5", addDisabled && "cursor-not-allowed opacity-60")}
        >
          <Plus className="mt-0.5 size-4 shrink-0 text-muted-foreground" aria-hidden />
          <span className="min-w-0">
            <span className="block text-sm font-medium">{COLUMN_AUTOMATIONS_COPY.addLabel}</span>
            <span className="mt-0.5 block text-[13px] text-muted-foreground">{COLUMN_AUTOMATIONS_COPY.addHint}</span>
          </span>
        </button>
      </PopoverContent>
    </Popover>
  );
}

function ColumnAutomationRow({
  automation,
  onAction,
}: {
  automation: ColumnAutomation;
  onAction: (action: ColumnAutomationRowAction) => void;
}) {
  const disabled = automation.health === "disabled";
  const needsRepair = automation.health === "needs-repair";

  return (
    <li>
      <button
        type="button"
        data-testid={`column-automation-row-${automation.id}`}
        onClick={() => onAction("settings")}
        className={cn(AUTOMATION_ITEM_CLASS, disabled && "opacity-70")}
      >
        {automation.iconSrc ? (
          <img
            src={automation.iconSrc}
            alt=""
            className={cn(
              "mt-0.5 size-4 shrink-0 object-contain",
              automation.iconAlt === "GitHub" && "dark:brightness-0 dark:invert",
            )}
          />
        ) : (
          <Workflow className="mt-0.5 size-4 shrink-0 text-muted-foreground" aria-hidden />
        )}
        <span className="min-w-0">
          <span className="block text-sm font-medium">{automation.name}</span>
          <span className="mt-0.5 block text-[13px] text-muted-foreground">
            {automation.trigger} → {automation.action}
          </span>
          {needsRepair || disabled ? (
            <span className="mt-1 flex flex-wrap items-center gap-1.5">
              {needsRepair ? (
                <span className="text-[11px] font-medium text-amber-700 dark:text-amber-400">
                  {COLUMN_AUTOMATIONS_COPY.needsRepairLabel}
                </span>
              ) : null}
              {disabled ? (
                <span className="text-[11px] font-medium text-muted-foreground">
                  {COLUMN_AUTOMATIONS_COPY.disabledLabel}
                </span>
              ) : null}
            </span>
          ) : null}
        </span>
      </button>
    </li>
  );
}
