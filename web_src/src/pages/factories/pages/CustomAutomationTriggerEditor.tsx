import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { cn, resolveIcon } from "@/lib/utils";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { AlarmClock, ChevronDown, GitCommitHorizontal, MousePointerClick, Plus, Trash2 } from "lucide-react";
import { useState } from "react";

import {
  AUTOMATION_ICONS,
  customAutomationColumnLabel,
  newCustomAutomationTrigger,
  type CustomAutomationColumn,
  type CustomAutomationMode,
  type CustomAutomationTrigger,
} from "../lib/customAutomation";
import { AUTOMATION_AGENTS, isAutomationHarness, type AutomationAgent } from "../lib/customAutomationDefinition";
import { CustomAutomationScheduleFields } from "./CustomAutomationScheduleFields";
import { PlanningReviewForm } from "./PlanningReviewForm";
import type { PlanningReviewDraft } from "./planningReviewMockup";

export function TriggerEditor({
  columnKey,
  asCode,
  name,
  icon,
  triggers,
  onTriggersChange,
  draft,
  organizationId,
  factoryId,
  factoryKey,
  onNameChange,
  onIconChange,
  onDraftChange,
}: {
  columnKey: CustomAutomationColumn;
  asCode?: { agent?: AutomationAgent; onAgentChange: (agent?: AutomationAgent) => void };
  name: string;
  icon: string;
  triggers: CustomAutomationTrigger[];
  onTriggersChange: (triggers: CustomAutomationTrigger[]) => void;
  draft: PlanningReviewDraft;
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  onNameChange: (name: string) => void;
  onIconChange: (icon: string) => void;
  onDraftChange: (draft: PlanningReviewDraft) => void;
}) {
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const updateTrigger = (id: string, next: CustomAutomationTrigger) => {
    onTriggersChange(triggers.map((trigger) => (trigger.id === id ? next : trigger)));
  };
  const addTrigger = (mode: CustomAutomationMode) => {
    const trigger = newCustomAutomationTrigger(mode);
    onTriggersChange([...triggers, trigger]);
    setExpandedId(mode === "scheduled" ? trigger.id : null);
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
        {asCode ? <AutomationAgentFields agent={asCode.agent} onChange={asCode.onAgentChange} /> : null}
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
          applyModelDefault={!asCode}
        />
      </section>
      <section className="flex flex-col gap-4">
        <h3 className="text-base font-semibold text-foreground">Triggers</h3>
        {triggers.map((trigger) => (
          <TriggerCard
            key={trigger.id}
            columnKey={columnKey}
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
          />
        ))}
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button type="button" variant="outline" size="sm" className="w-fit gap-1.5">
              <Plus className="size-3.5" aria-hidden />
              Add trigger
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start" className="min-w-72">
            {(["manual", "automatic", "scheduled"] as const).map((mode) => (
              <DropdownMenuItem
                key={mode}
                className="py-2.5"
                disabled={mode !== "scheduled" && triggers.some((trigger) => trigger.mode === mode)}
                onSelect={() => addTrigger(mode)}
              >
                <TriggerOption mode={mode} columnKey={columnKey} />
              </DropdownMenuItem>
            ))}
          </DropdownMenuContent>
        </DropdownMenu>
      </section>
    </div>
  );
}

function TriggerCard({
  columnKey,
  trigger,
  triggers,
  expanded,
  onToggle,
  onRemove,
  onScheduleChange,
}: {
  columnKey: CustomAutomationColumn;
  trigger: CustomAutomationTrigger;
  triggers: CustomAutomationTrigger[];
  expanded: boolean;
  onToggle: () => void;
  onRemove: () => void;
  onScheduleChange: (schedule: CustomAutomationTrigger["schedule"]) => void;
}) {
  const configurable = trigger.mode === "scheduled";
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
            <TriggerOption mode={trigger.mode} columnKey={columnKey} />
            <ChevronDown
              className={cn("ml-auto size-4 shrink-0 text-muted-foreground", !expanded && "-rotate-90")}
              aria-hidden
            />
          </Button>
        ) : (
          <div className="min-w-0 flex-1 px-3 py-2.5">
            <TriggerOption mode={trigger.mode} columnKey={columnKey} />
          </div>
        )}
        {triggers.length > 1 ? (
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            className="text-muted-foreground hover:text-destructive"
            aria-label={removeTriggerLabel(triggers, trigger, columnKey)}
            onClick={onRemove}
          >
            <Trash2 className="size-4" aria-hidden />
          </Button>
        ) : null}
      </div>
      {configurable && expanded ? (
        <div className="border-t border-border px-4 py-4">
          <CustomAutomationScheduleFields
            embedded
            idPrefix={trigger.id}
            schedule={trigger.schedule}
            onChange={onScheduleChange}
          />
        </div>
      ) : null}
    </div>
  );
}

function AutomationAgentFields({
  agent,
  onChange,
}: {
  agent?: AutomationAgent;
  onChange: (agent?: AutomationAgent) => void;
}) {
  return (
    <div className="grid gap-4">
      <div className="grid gap-2">
        <Label htmlFor="custom-automation-agent">Agent</Label>
        <Select
          value={agent?.harness ?? "factory"}
          onValueChange={(harness) => onChange(isAutomationHarness(harness) ? { harness } : undefined)}
        >
          <SelectTrigger id="custom-automation-agent" className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="factory">Factory default</SelectItem>
            {Object.entries(AUTOMATION_AGENTS).map(([harness, { label }]) => (
              <SelectItem key={harness} value={harness}>
                {label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      {agent && agent.harness !== "superplane" ? (
        <AutomationCredentialsFields agent={agent} onChange={onChange} />
      ) : null}
    </div>
  );
}

function AutomationCredentialsFields({
  agent,
  onChange,
}: {
  agent: AutomationAgent;
  onChange: (agent: AutomationAgent) => void;
}) {
  return (
    <div className="grid gap-2">
      <Label htmlFor="automation-credential-source">Credentials</Label>
      <Select
        value={agent.credentials?.source ?? "factory"}
        onValueChange={(value) => {
          if (value === "subscription") {
            onChange({ harness: agent.harness, credentials: { source: "subscription" } });
          } else if (value === "integration") {
            onChange({ ...agent, credentials: { source: "integration", name: "" } });
          } else {
            onChange({ ...agent, credentials: undefined });
          }
        }}
      >
        <SelectTrigger id="automation-credential-source" className="w-full">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="factory">Factory default</SelectItem>
          <SelectItem value="integration">Integration</SelectItem>
          {agent.harness === "codex" ? <SelectItem value="subscription">ChatGPT subscription</SelectItem> : null}
        </SelectContent>
      </Select>
      {agent.credentials?.source === "integration" ? (
        <>
          <Label htmlFor="automation-credential-integration">Integration name</Label>
          <Input
            id="automation-credential-integration"
            value={agent.credentials.name}
            onChange={(event) =>
              onChange({ ...agent, credentials: { source: "integration", name: event.target.value } })
            }
          />
        </>
      ) : null}
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

function removeTriggerLabel(
  triggers: CustomAutomationTrigger[],
  trigger: CustomAutomationTrigger,
  columnKey: CustomAutomationColumn,
): string {
  const title = triggerTitle(trigger.mode, columnKey);
  const sameType = triggers.filter((item) => item.mode === trigger.mode);
  if (sameType.length < 2) {
    return `Remove ${title}`;
  }
  return `Remove ${title} ${sameType.findIndex((item) => item.id === trigger.id) + 1}`;
}

function triggerTitle(mode: CustomAutomationMode, columnKey: CustomAutomationColumn): string {
  if (mode === "scheduled") return "Schedule";
  if (mode === "automatic") {
    return columnKey === "verify"
      ? "Every pull request commit"
      : `When a task enters ${customAutomationColumnLabel(columnKey)}`;
  }
  return "Manual";
}

function TriggerOption({ mode, columnKey }: { mode: CustomAutomationMode; columnKey: CustomAutomationColumn }) {
  const automatic = mode === "automatic";
  const Icon = mode === "scheduled" ? AlarmClock : automatic ? GitCommitHorizontal : MousePointerClick;
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
        <span className="text-sm font-medium text-foreground">
          {mode === "scheduled"
            ? "Schedule"
            : automatic
              ? columnKey === "verify"
                ? "Every pull request commit"
                : `When a task enters ${customAutomationColumnLabel(columnKey)}`
              : "Manual"}
        </span>
        <span className="text-xs font-normal text-muted-foreground">
          {mode === "scheduled"
            ? columnKey === "backlog"
              ? "Run automation on a schedule."
              : "Runs on the repository default branch."
            : automatic
              ? columnKey === "verify"
                ? "Runs after each new commit."
                : `Runs when a task enters ${customAutomationColumnLabel(columnKey)}.`
              : "A person starts each run."}
        </span>
      </span>
    </span>
  );
}
