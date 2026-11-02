import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { cn, resolveIcon } from "@/lib/utils";
import { CircleAlert, GitCommitHorizontal, MousePointerClick } from "lucide-react";
import { useState } from "react";

import type { CustomAutomationMode, CustomAutomationColumn } from "../lib/customAutomation";
import { AutomationFlow } from "./customAutomationSetupFlow";
import { StepEditor } from "./customAutomationStepEditor";
import { PlanningReviewForm } from "./PlanningReviewForm";
import type { PlanningReviewDraft, PlanningReviewStep } from "./planningReviewMockup";

type CustomAutomationSetupFormProps = {
  name: string;
  icon: string;
  mode: CustomAutomationMode;
  columnKey?: CustomAutomationColumn;
  draft: PlanningReviewDraft;
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  saving: boolean;
  deleting: boolean;
  error: string;
  onNameChange: (name: string) => void;
  onIconChange: (icon: string) => void;
  onModeChange: (mode: CustomAutomationMode) => void;
  onDraftChange: (draft: PlanningReviewDraft) => void;
  onSave: () => void;
  onDelete?: () => void;
  onClose: () => void;
};

export function CustomAutomationSetupForm(props: CustomAutomationSetupFormProps) {
  const {
    name,
    icon,
    mode,
    columnKey = "verify",
    draft,
    organizationId,
    factoryId,
    factoryKey,
    saving,
    deleting,
    error,
    onNameChange,
    onIconChange,
    onModeChange,
    onDraftChange,
    onSave,
    onDelete,
    onClose,
  } = props;
  const [selectedStep, setSelectedStep] = useState<number | null>(null);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const component = draft.components[0];
  const steps = (component?.configuration.steps as PlanningReviewStep[] | undefined) ?? [];
  const activeStep = selectedStep === null ? undefined : steps[selectedStep];
  const { updateStep, addStep, removeStepAt, moveStepAt } = useStepList(
    draft,
    component,
    selectedStep,
    setSelectedStep,
    onDraftChange,
  );

  return (
    <div className="flex min-h-0 flex-1 flex-col" data-testid="custom-automation-setup">
      <div className="flex min-h-0 flex-1">
        <AutomationFlow
          columnKey={columnKey}
          mode={mode}
          steps={steps}
          selectedStep={selectedStep}
          onSelectTrigger={() => setSelectedStep(null)}
          onSelectStep={setSelectedStep}
          onMoveStep={moveStepAt}
          onRemoveStep={removeStepAt}
          onAddStep={addStep}
        />
        <div className="min-h-0 min-w-0 flex-1 overflow-y-auto bg-background">
          {activeStep ? (
            <StepEditor
              step={activeStep}
              position={selectedStep ?? 0}
              stepCount={steps.length}
              onChange={updateStep}
              onMove={(offset) => selectedStep !== null && moveStepAt(selectedStep, offset)}
              onRemove={() => selectedStep !== null && removeStepAt(selectedStep)}
              organizationId={organizationId}
            />
          ) : (
            <TriggerEditor
              columnKey={columnKey}
              name={name}
              icon={icon}
              mode={mode}
              draft={draft}
              organizationId={organizationId}
              factoryId={factoryId}
              factoryKey={factoryKey}
              onNameChange={onNameChange}
              onIconChange={onIconChange}
              onModeChange={onModeChange}
              onDraftChange={onDraftChange}
            />
          )}
        </div>
      </div>
      <SetupFormFooter
        error={error}
        saving={saving}
        deleting={deleting}
        confirmDelete={confirmDelete}
        onConfirmDeleteChange={setConfirmDelete}
        onSave={onSave}
        onDelete={onDelete}
        onClose={onClose}
      />
    </div>
  );
}

function useStepList(
  draft: PlanningReviewDraft,
  component: PlanningReviewDraft["components"][number] | undefined,
  selectedStep: number | null,
  setSelectedStep: (value: number | null) => void,
  onDraftChange: (draft: PlanningReviewDraft) => void,
) {
  const steps = (component?.configuration.steps as PlanningReviewStep[] | undefined) ?? [];
  const updateSteps = (nextSteps: PlanningReviewStep[]) => {
    if (!component) return;
    onDraftChange({
      ...draft,
      components: draft.components.map((entry) =>
        entry.id === component.id ? { ...entry, configuration: { ...entry.configuration, steps: nextSteps } } : entry,
      ),
    });
  };
  return {
    updateStep: (nextStep: PlanningReviewStep) => {
      if (selectedStep === null) return;
      updateSteps(steps.map((step, index) => (index === selectedStep ? nextStep : step)));
    },
    addStep: () => {
      updateSteps([...steps, { name: "New step", type: "bash", command: "" }]);
      setSelectedStep(steps.length);
    },
    removeStepAt: (position: number) => {
      updateSteps(steps.filter((_, index) => index !== position));
      if (selectedStep === position) {
        setSelectedStep(null);
      } else if (selectedStep !== null && selectedStep > position) {
        setSelectedStep(selectedStep - 1);
      }
    },
    moveStepAt: (position: number, offset: -1 | 1) => {
      const destination = position + offset;
      if (destination < 0 || destination >= steps.length) return;
      const nextSteps = [...steps];
      [nextSteps[position], nextSteps[destination]] = [nextSteps[destination], nextSteps[position]];
      updateSteps(nextSteps);
      if (selectedStep === position) {
        setSelectedStep(destination);
      } else if (selectedStep === destination) {
        setSelectedStep(position);
      }
    },
  };
}

function SetupFormFooter({
  error,
  saving,
  deleting,
  confirmDelete,
  onConfirmDeleteChange,
  onSave,
  onDelete,
  onClose,
}: {
  error: string;
  saving: boolean;
  deleting: boolean;
  confirmDelete: boolean;
  onConfirmDeleteChange: (value: boolean) => void;
  onSave: () => void;
  onDelete?: () => void;
  onClose: () => void;
}) {
  return (
    <footer className="flex shrink-0 flex-col gap-3 border-t border-border bg-popover px-6 py-4">
      {error ? (
        <div
          className="flex w-full items-center gap-2 rounded-lg border border-destructive/30 bg-destructive/10 px-3 py-2.5 text-sm font-medium text-destructive"
          role="alert"
        >
          <CircleAlert className="size-4 shrink-0" aria-hidden />
          <p>{error}</p>
        </div>
      ) : null}
      <div className="flex w-full items-center justify-between gap-3">
        <div>
          {onDelete ? (
            confirmDelete ? (
              <div className="flex items-center gap-2">
                <span className="text-sm text-destructive">Delete this automation?</span>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  disabled={deleting}
                  onClick={() => onConfirmDeleteChange(false)}
                >
                  Keep
                </Button>
                <LoadingButton
                  type="button"
                  variant="destructive"
                  size="sm"
                  loading={deleting}
                  loadingText="Deleting…"
                  onClick={onDelete}
                >
                  Delete
                </LoadingButton>
              </div>
            ) : (
              <Button
                type="button"
                variant="ghost"
                size="sm"
                disabled={saving}
                onClick={() => onConfirmDeleteChange(true)}
              >
                Delete
              </Button>
            )
          ) : null}
        </div>
        <div className="flex gap-3">
          <Button type="button" variant="outline" onClick={onClose} disabled={saving || deleting}>
            Cancel
          </Button>
          <LoadingButton
            type="button"
            loading={saving}
            loadingText="Saving…"
            disabled={deleting}
            onClick={onSave}
            data-testid="custom-automation-save"
          >
            Save
          </LoadingButton>
        </div>
      </div>
    </footer>
  );
}

function TriggerEditor({
  columnKey,
  name,
  icon,
  mode,
  draft,
  organizationId,
  factoryId,
  factoryKey,
  onNameChange,
  onIconChange,
  onModeChange,
  onDraftChange,
}: {
  columnKey: CustomAutomationColumn;
  name: string;
  icon: string;
  mode: CustomAutomationMode;
  draft: PlanningReviewDraft;
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  onNameChange: (name: string) => void;
  onIconChange: (icon: string) => void;
  onModeChange: (mode: CustomAutomationMode) => void;
  onDraftChange: (draft: PlanningReviewDraft) => void;
}) {
  return (
    <div className="mx-auto flex w-full max-w-3xl flex-col gap-8 px-8 py-10">
      <h3 className="text-base font-semibold text-foreground">Edit Trigger</h3>
      <div>
        <Label htmlFor="custom-automation-mode" className="sr-only">
          Trigger
        </Label>
        <Select value={mode} onValueChange={(value) => onModeChange(value === "automatic" ? "automatic" : "manual")}>
          <SelectTrigger id="custom-automation-mode" className="min-h-14 w-full !h-auto py-2.5">
            <SelectValue>
              <TriggerOption mode={mode} columnKey={columnKey} />
            </SelectValue>
          </SelectTrigger>
          <SelectContent className="min-w-[var(--radix-select-trigger-width)]">
            <SelectItem value="manual" className="py-2.5">
              <TriggerOption mode="manual" columnKey={columnKey} />
            </SelectItem>
            <SelectItem value="automatic" className="py-2.5">
              <TriggerOption mode="automatic" columnKey={columnKey} />
            </SelectItem>
          </SelectContent>
        </Select>
      </div>
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
      />
    </div>
  );
}

const AUTOMATION_ICONS = [
  "sparkles",
  "shield-check",
  "search",
  "bug",
  "file-code",
  "code",
  "terminal",
  "eye",
  "gauge",
  "git-pull-request",
  "cpu",
  "circle-check",
  "activity",
  "alarm-clock",
  "archive",
  "bell",
  "bolt",
  "book",
  "box",
  "boxes",
  "braces",
  "calendar",
  "camera",
  "chart-line",
  "check",
  "clock",
  "cloud",
  "cog",
  "component",
  "container",
  "database",
  "factory",
  "file-json",
  "file-text",
  "fingerprint",
  "flag",
  "folder",
  "git-branch",
  "git-commit",
  "git-merge",
  "globe",
  "key",
  "layers",
  "lock",
  "message-square",
  "network",
  "package",
  "rocket",
] as const;

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

function TriggerOption({ mode, columnKey }: { mode: CustomAutomationMode; columnKey: CustomAutomationColumn }) {
  const automatic = mode === "automatic";
  const Icon = automatic ? GitCommitHorizontal : MousePointerClick;
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
          {automatic
            ? columnKey === "backlog"
              ? "When a task enters Backlog"
              : "Every pull request commit"
            : "Manual"}
        </span>
        <span className="text-xs font-normal text-muted-foreground">
          {automatic
            ? columnKey === "backlog"
              ? "Runs when a task enters Backlog."
              : "Runs after each new commit."
            : "Runs only when you start it."}
        </span>
      </span>
    </span>
  );
}
