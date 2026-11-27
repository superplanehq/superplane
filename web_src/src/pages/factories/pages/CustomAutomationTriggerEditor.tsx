import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { cn, resolveIcon } from "@/lib/utils";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { SegmentedNav } from "@/ui/SegmentedNav";
import {
  AlarmClock,
  ChevronDown,
  GitCommitHorizontal,
  LogOut,
  MousePointerClick,
  Plus,
  Trash2,
  Webhook,
} from "lucide-react";
import { useState } from "react";

import {
  AUTOMATION_ICONS,
  customAutomationColumnLabel,
  customAutomationTriggersForRun,
  newCustomAutomationTrigger,
  type CustomAutomationColumn,
  type CustomAutomationMode,
  type CustomAutomationTrigger,
} from "../lib/customAutomation";
import { CustomAutomationScheduleFields } from "./CustomAutomationScheduleFields";
import { CustomAutomationWebhookFields } from "./CustomAutomationWebhookFields";
import { PlanningReviewForm } from "./PlanningReviewForm";
import type { PlanningReviewDraft } from "./planningReviewMockup";
import { useCustomAutomationStageLabel } from "./useCustomAutomationStageLabel";

type TriggerEditorProps = {
  showModel?: boolean;
  columnKey: CustomAutomationColumn;
  name: string;
  icon: string;
  triggers: CustomAutomationTrigger[];
  runsOnTask?: boolean;
  onTriggersChange: (triggers: CustomAutomationTrigger[]) => void;
  onRunsOnTaskChange: (runsOnTask: boolean) => void;
  draft: PlanningReviewDraft;
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  onNameChange: (name: string) => void;
  onIconChange: (icon: string) => void;
  onDraftChange: (draft: PlanningReviewDraft) => void;
};

export function TriggerEditor({
  columnKey,
  showModel = true,
  name,
  icon,
  triggers,
  runsOnTask,
  onTriggersChange,
  onRunsOnTaskChange,
  draft,
  organizationId,
  factoryId,
  factoryKey,
  onNameChange,
  onIconChange,
  onDraftChange,
}: TriggerEditorProps) {
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const visibleTriggers = customAutomationTriggersForRun(triggers, runsOnTask);
  const triggerModes = availableTriggerModes(columnKey, runsOnTask);
  const updateTrigger = (id: string, next: CustomAutomationTrigger) => {
    onTriggersChange(triggers.map((trigger) => (trigger.id === id ? next : trigger)));
  };
  const addTrigger = (mode: CustomAutomationMode) => {
    const trigger = newCustomAutomationTrigger(mode);
    onTriggersChange([...triggers, trigger]);
    setExpandedId(repeatableTriggerMode(mode) ? trigger.id : null);
  };
  return (
    <div className="mx-auto flex w-full max-w-3xl flex-col gap-10 px-8 py-10">
      <section className="flex flex-col gap-8">
        <h3 className="text-base font-semibold text-foreground">Automation</h3>
        <div className="flex flex-col gap-2">
          <Label htmlFor="automation-name">Automation name</Label>
          <div className="flex items-center gap-2">
            <AutomationIconPicker icon={icon} onChange={onIconChange} />
            <Input
              id="automation-name"
              value={name}
              onChange={(event) => onNameChange(event.target.value)}
              placeholder="Automation name"
              className="min-w-0 flex-1 shadow-none"
            />
          </div>
        </div>
      </section>
      <section className="flex flex-col gap-4">
        <div className="flex flex-col gap-1.5">
          <h3 className="text-base font-semibold text-foreground">Triggers</h3>
          <TriggerContext
            columnKey={columnKey}
            triggers={triggers}
            runsOnTask={runsOnTask}
            onRunsOnTaskChange={onRunsOnTaskChange}
          />
        </div>
        {visibleTriggers.map((trigger) => (
          <TriggerCard
            key={trigger.id}
            columnKey={columnKey}
            runsOnTask={runsOnTask !== false}
            trigger={trigger}
            triggers={triggers}
            expanded={expandedId === trigger.id}
            onToggle={() => setExpandedId(expandedId === trigger.id ? null : trigger.id)}
            onRemove={() => {
              if (expandedId === trigger.id) {
                setExpandedId(null);
              }
              onTriggersChange(triggers.filter((item) => item.id !== trigger.id));
            }}
            onScheduleChange={(schedule) => updateTrigger(trigger.id, { ...trigger, schedule })}
            onWebhookChange={(webhookId) => updateTrigger(trigger.id, { ...trigger, webhookId })}
            organizationId={organizationId}
            factoryId={factoryId}
            factoryKey={factoryKey}
          />
        ))}
        <AddTriggerMenu
          modes={triggerModes}
          triggers={triggers}
          columnKey={columnKey}
          runsOnTask={runsOnTask}
          onAdd={addTrigger}
        />
      </section>
      <section className="flex flex-col gap-8">
        <h3 className="text-base font-semibold text-foreground">Agent</h3>
        <PlanningReviewForm
          draft={draft}
          onChange={onDraftChange}
          organizationId={organizationId}
          factoryId={factoryId}
          factoryKey={factoryKey}
          appearance="fields"
          view="implementation"
          showConcurrency={false}
          showResources={false}
          showSteps={false}
          modelOnly
          applyModelDefault={false}
          showModel={showModel}
          modelDescription="The model used for this automation."
        />
      </section>
    </div>
  );
}

function AddTriggerMenu({
  modes,
  triggers,
  columnKey,
  runsOnTask,
  onAdd,
}: {
  modes: CustomAutomationMode[];
  triggers: CustomAutomationTrigger[];
  columnKey: CustomAutomationColumn;
  runsOnTask?: boolean;
  onAdd: (mode: CustomAutomationMode) => void;
}) {
  if (modes.length === 0) return null;
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button type="button" variant="outline" size="sm" className="w-fit gap-1.5">
          <Plus className="size-3.5" aria-hidden />
          Add trigger
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="min-w-72">
        {modes.map((mode) => (
          <DropdownMenuItem
            key={mode}
            className="py-2.5"
            disabled={!repeatableTriggerMode(mode) && triggers.some((trigger) => trigger.mode === mode)}
            onSelect={() => onAdd(mode)}
          >
            <TriggerOption mode={mode} columnKey={columnKey} runsOnTask={runsOnTask !== false} />
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function TriggerCard({
  columnKey,
  runsOnTask,
  trigger,
  triggers,
  expanded,
  onToggle,
  onRemove,
  onScheduleChange,
  onWebhookChange,
  organizationId,
  factoryId,
  factoryKey,
}: {
  columnKey: CustomAutomationColumn;
  runsOnTask: boolean;
  trigger: CustomAutomationTrigger;
  triggers: CustomAutomationTrigger[];
  expanded: boolean;
  onToggle: () => void;
  onRemove: () => void;
  onScheduleChange: (schedule: CustomAutomationTrigger["schedule"]) => void;
  onWebhookChange: (webhookId: string) => void;
  organizationId: string;
  factoryId: string;
  factoryKey: string;
}) {
  const configurable = repeatableTriggerMode(trigger.mode);
  const stageLabel = useCustomAutomationStageLabel(columnKey);
  return (
    <div className="rounded-lg border border-border bg-background">
      <div className="flex items-center gap-1 pr-2">
        {configurable ? (
          <Button
            type="button"
            variant="ghost"
            className="h-auto min-w-0 flex-1 justify-start gap-2 px-3 py-2.5 text-left font-normal"
            aria-expanded={expanded}
            onClick={onToggle}
          >
            <TriggerOption mode={trigger.mode} columnKey={columnKey} runsOnTask={runsOnTask} />
            <ChevronDown
              className={cn("ml-auto size-4 shrink-0 text-muted-foreground", !expanded && "-rotate-90")}
              aria-hidden
            />
          </Button>
        ) : (
          <div className="min-w-0 flex-1 px-3 py-2.5">
            <TriggerOption mode={trigger.mode} columnKey={columnKey} runsOnTask={runsOnTask} />
          </div>
        )}
        {triggers.length > 1 ? (
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            className="text-muted-foreground hover:text-destructive"
            aria-label={removeTriggerLabel(triggers, trigger, columnKey, stageLabel)}
            onClick={onRemove}
          >
            <Trash2 className="size-4" aria-hidden />
          </Button>
        ) : null}
      </div>
      {configurable && expanded ? (
        <div className="border-t border-border px-4 py-4">
          {trigger.mode === "webhook" ? (
            <CustomAutomationWebhookFields
              idPrefix={trigger.id}
              organizationId={organizationId}
              factoryId={factoryId}
              factoryKey={factoryKey}
              helper={
                columnKey === "backlog"
                  ? "The agent gets the JSON request body and can create a task."
                  : "The agent gets the JSON request body. The run has no task."
              }
              webhookId={trigger.webhookId}
              usedWebhookIds={triggers
                .filter((item) => item.id !== trigger.id && item.mode === "webhook" && item.webhookId)
                .map((item) => item.webhookId!)}
              onChange={onWebhookChange}
            />
          ) : (
            <CustomAutomationScheduleFields
              embedded
              idPrefix={trigger.id}
              schedule={trigger.schedule}
              onChange={onScheduleChange}
            />
          )}
        </div>
      ) : null}
    </div>
  );
}

/** Chooses which triggers this automation can run. The choice is saved. */
function TriggerContext({
  columnKey,
  triggers,
  runsOnTask,
  onRunsOnTaskChange,
}: {
  columnKey: CustomAutomationColumn;
  triggers: CustomAutomationTrigger[];
  runsOnTask?: boolean;
  onRunsOnTaskChange: (runsOnTask: boolean) => void;
}) {
  if (columnKey.startsWith("custom-")) {
    return (
      <p className="text-xs text-muted-foreground">
        {triggers.some((trigger) => trigger.mode === "automatic" || trigger.mode === "manual")
          ? "This automation runs on a task in this stage. Every required automation must pass before the task advances. A webhook can also start it with no task."
          : "A webhook starts this automation with no task. Tasks wait in this stage only when auto advance is off. A webhook run can move them."}
      </p>
    );
  }
  return (
    <div className="flex items-center gap-2" data-testid="custom-automation-runs-on-task">
      <span className="text-xs text-muted-foreground">Runs</span>
      <SegmentedNav
        ariaLabel="Runs"
        size="xs"
        value={runsOnTask === undefined ? "" : runsOnTask ? "task" : "none"}
        onValueChange={(value) => onRunsOnTaskChange(value === "task")}
        options={[
          { value: "task", label: "on a task" },
          { value: "none", label: "with no task" },
        ]}
      />
    </div>
  );
}

function AutomationIconPicker({ icon, onChange }: { icon: string; onChange: (icon: string) => void }) {
  const [open, setOpen] = useState(false);
  const Icon = resolveIcon(icon);
  return (
    <div className="relative">
      <Button
        type="button"
        variant="outline"
        size="icon-sm"
        className="size-8 shrink-0 rounded-md border-[#e5e5e5] shadow-none dark:border-gray-600/70"
        aria-label="Choose automation icon"
        aria-expanded={open}
        onClick={() => setOpen((current) => !current)}
      >
        <Icon className="size-4 text-muted-foreground" aria-hidden />
      </Button>
      {open ? (
        <div className="absolute top-full left-0 z-30 mt-1.5 w-56 rounded-md border bg-popover p-2 text-popover-foreground shadow-sm">
          <div className="grid grid-cols-6 gap-1">
            {AUTOMATION_ICONS.map((option) => {
              const OptionIcon = resolveIcon(option);
              const selected = option === icon;
              return (
                <Button
                  key={option}
                  type="button"
                  variant="ghost"
                  size="icon-xs"
                  className={cn(
                    "size-7 text-muted-foreground hover:text-foreground",
                    selected && "bg-muted text-foreground",
                  )}
                  aria-label={option.replace(/-/g, " ")}
                  aria-pressed={selected}
                  onClick={() => {
                    onChange(option);
                    setOpen(false);
                  }}
                >
                  <OptionIcon className="size-3.5" aria-hidden />
                </Button>
              );
            })}
          </div>
        </div>
      ) : null}
    </div>
  );
}

/** A custom stage starts from Manual, its line trigger, webhooks, or a mix. Other stages offer every trigger for the run context. */
function availableTriggerModes(columnKey: CustomAutomationColumn, runsOnTask?: boolean): CustomAutomationMode[] {
  if (columnKey.startsWith("custom-")) return ["manual", "automatic", "webhook"];
  const modes: CustomAutomationMode[] = ["manual"];
  if (runsOnTask !== false) modes.push("automatic", "leave");
  if (runsOnTask !== true) modes.push("scheduled", "webhook");
  return modes;
}

/** Schedules and webhooks can repeat with different settings. Event triggers cannot. */
function repeatableTriggerMode(mode: CustomAutomationMode): boolean {
  return mode === "scheduled" || mode === "webhook";
}

function removeTriggerLabel(
  triggers: CustomAutomationTrigger[],
  trigger: CustomAutomationTrigger,
  columnKey: CustomAutomationColumn,
  stageLabel?: string,
): string {
  const title = triggerTitle(trigger.mode, columnKey, stageLabel);
  const sameType = triggers.filter((item) => item.mode === trigger.mode);
  if (sameType.length < 2) {
    return `Remove ${title}`;
  }
  return `Remove ${title} ${sameType.findIndex((item) => item.id === trigger.id) + 1}`;
}

function triggerTitle(
  mode: CustomAutomationMode,
  columnKey: CustomAutomationColumn,
  stageLabel = customAutomationColumnLabel(columnKey),
): string {
  if (mode === "scheduled") return "Schedule";
  if (mode === "webhook") return "Webhook";
  if (mode === "leave") return `When a task leaves ${stageLabel}`;
  if (mode === "automatic") {
    return columnKey === "verify" ? "Every pull request commit" : `When a task enters ${stageLabel}`;
  }
  return "Manual";
}

function TriggerOption({
  mode,
  columnKey,
  runsOnTask,
}: {
  mode: CustomAutomationMode;
  columnKey: CustomAutomationColumn;
  runsOnTask: boolean;
}) {
  const automatic = mode === "automatic" || mode === "leave";
  const stageLabel = useCustomAutomationStageLabel(columnKey);
  const Icon =
    mode === "scheduled"
      ? AlarmClock
      : mode === "webhook"
        ? Webhook
        : mode === "leave"
          ? LogOut
          : automatic
            ? GitCommitHorizontal
            : MousePointerClick;
  return (
    <span className="flex min-w-0 items-center gap-3 text-left">
      <span
        className={cn(
          "flex size-8 shrink-0 items-center justify-center rounded-lg",
          automatic
            ? "bg-violet-500/10 text-violet-700 dark:text-violet-400"
            : "bg-amber-500/10 text-amber-700 dark:text-amber-400",
        )}
      >
        <Icon className="size-4" aria-hidden />
      </span>
      <span className="flex min-w-0 flex-col">
        <span className="text-sm font-medium text-foreground">{triggerTitle(mode, columnKey, stageLabel)}</span>
        <span className="text-xs font-normal text-muted-foreground">
          {mode === "webhook"
            ? "Runs when a webhook receives a request."
            : mode === "scheduled"
              ? columnKey === "backlog"
                ? "Run automation on a schedule."
                : "Runs on the repository default branch."
              : mode === "leave"
                ? `Runs when a task leaves ${stageLabel}.`
                : automatic
                  ? columnKey === "verify"
                    ? "Runs after each new commit."
                    : `Runs when a task enters ${stageLabel}.`
                  : runsOnTask
                    ? "A person starts each run from a task."
                    : "A person starts each run from the column."}
        </span>
      </span>
    </span>
  );
}
