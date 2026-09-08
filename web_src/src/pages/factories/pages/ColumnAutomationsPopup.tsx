import { cn } from "@/lib/utils";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/ui/dropdownMenu";
import { Bot, Pencil, Sparkles, Workflow } from "lucide-react";
import type { LucideIcon } from "lucide-react";

import { COLUMN_AUTOMATIONS_COPY, type ColumnAutomation, type ColumnAutomationKind } from "../lib/columnAutomations";

export type ColumnAutomationRowAction = "settings" | "edit" | "edit-agent" | "disable" | "enable" | "remove";

const KIND_FALLBACK_ICON: Partial<Record<ColumnAutomationKind, LucideIcon>> = {
  analysis: Sparkles,
  "agent-step": Bot,
  custom: Workflow,
};

interface ColumnAutomationsPopupProps {
  automation: ColumnAutomation;
  onAction: (action: ColumnAutomationRowAction) => void;
  showEditAgent?: boolean;
  showEditAutomation?: boolean;
  /** Open the menu on first render. Used by stories. */
  defaultOpen?: boolean;
}

/** Header icon. Click opens a menu with the automation summary and edit actions. */
export function ColumnAutomationsPopup({
  automation,
  onAction,
  showEditAgent = false,
  showEditAutomation = false,
  defaultOpen = false,
}: ColumnAutomationsPopupProps) {
  const needsRepair = automation.health === "needs-repair";
  const disabled = automation.health === "disabled";
  const hasEditActions = showEditAgent || showEditAutomation;

  return (
    <DropdownMenu defaultOpen={defaultOpen}>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          aria-label={automation.name}
          title={automation.name}
          data-testid={`column-automation-icon-${automation.id}`}
          className={cn(
            "relative flex size-6 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-foreground",
            disabled && "opacity-70",
          )}
        >
          <ColumnAutomationGlyph automation={automation} className="size-3.5" />
          {needsRepair ? (
            <span
              className="absolute -right-0.5 -top-0.5 size-1.5 rounded-full bg-amber-500"
              data-testid={`column-automation-icon-${automation.id}-needs-repair`}
              aria-hidden
            />
          ) : null}
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="min-w-56 w-80 p-0" data-testid="column-automations-popup">
        <div className="p-1">
          <ColumnAutomationInfo automation={automation} onOpenSettings={() => onAction("settings")} />
        </div>
        {hasEditActions ? (
          <>
            <DropdownMenuSeparator className="my-0" />
            <div className="p-1">
              {showEditAgent ? (
                <DropdownMenuItem
                  onSelect={() => onAction("edit-agent")}
                  data-testid={`column-automation-${automation.id}-edit-agent`}
                >
                  <Bot className="h-3.5 w-3.5" aria-hidden />
                  {COLUMN_AUTOMATIONS_COPY.editAgentLabel}
                </DropdownMenuItem>
              ) : null}
              {showEditAutomation ? (
                <DropdownMenuItem
                  onSelect={() => onAction("edit")}
                  data-testid={`column-automation-${automation.id}-edit`}
                >
                  <Pencil className="h-3.5 w-3.5" aria-hidden />
                  {COLUMN_AUTOMATIONS_COPY.editAutomationMenuLabel}
                </DropdownMenuItem>
              ) : null}
            </div>
          </>
        ) : null}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export function ColumnAutomationGlyph({
  automation,
  className,
}: {
  automation: Pick<ColumnAutomation, "iconSrc" | "iconAlt" | "kind">;
  className?: string;
}) {
  if (automation.iconSrc) {
    return (
      <img
        src={automation.iconSrc}
        alt=""
        className={cn(
          "shrink-0 object-contain",
          className,
          automation.iconAlt === "GitHub" && "dark:brightness-0 dark:invert",
        )}
      />
    );
  }
  const Icon = KIND_FALLBACK_ICON[automation.kind] ?? Workflow;
  return <Icon className={cn("shrink-0 text-muted-foreground", className)} aria-hidden />;
}

function ColumnAutomationInfo({
  automation,
  onOpenSettings,
}: {
  automation: ColumnAutomation;
  onOpenSettings: () => void;
}) {
  const disabled = automation.health === "disabled";
  const needsRepair = automation.health === "needs-repair";

  return (
    <DropdownMenuItem
      data-testid={`column-automation-row-${automation.id}`}
      onSelect={onOpenSettings}
      className={cn("items-start gap-3 py-2.5", disabled && "opacity-70")}
    >
      <ColumnAutomationGlyph automation={automation} className="mt-0.5 size-4" />
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
    </DropdownMenuItem>
  );
}

export function automationOffersAgentEdit(automation: ColumnAutomation): boolean {
  return automation.kind === "agent-step" || automation.kind === "analysis";
}
