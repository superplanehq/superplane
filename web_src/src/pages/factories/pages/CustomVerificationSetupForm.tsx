import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { usePermissions } from "@/contexts/usePermissions";
import { useSecrets } from "@/hooks/useSecrets";
import { cn, resolveIcon } from "@/lib/utils";
import { CreateSecretDialog, type CreatedSecretSummary } from "@/ui/CreateSecretDialog";
import {
  ArrowDown,
  ArrowUp,
  CircleAlert,
  GitCommitHorizontal,
  KeyRound,
  MessageSquareText,
  MousePointerClick,
  Plus,
  Terminal,
  Trash2,
  X,
  Zap,
} from "lucide-react";
import { useState, type ReactNode } from "react";

import type { CustomVerificationMode } from "../lib/customVerification";
import { ImplementSettingRow } from "./ImplementSettingRow";
import { mergeCheckPromptBody, withMergeCheckHeader } from "./mergeConfidenceSteps";
import { PlanningReviewForm } from "./PlanningReviewForm";
import { PlanningReviewStepBody } from "./PlanningReviewStepBody";
import type { PlanningReviewDraft, PlanningReviewStep, PlanningReviewStepKind } from "./planningReviewMockup";

type CustomVerificationSetupFormProps = {
  name: string;
  icon: string;
  mode: CustomVerificationMode;
  draft: PlanningReviewDraft;
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  saving: boolean;
  deleting: boolean;
  error: string;
  onNameChange: (name: string) => void;
  onIconChange: (icon: string) => void;
  onModeChange: (mode: CustomVerificationMode) => void;
  onDraftChange: (draft: PlanningReviewDraft) => void;
  onSave: () => void;
  onDelete?: () => void;
  onClose: () => void;
};

export function CustomVerificationSetupForm(props: CustomVerificationSetupFormProps) {
  const {
    name,
    icon,
    mode,
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

  const updateSteps = (nextSteps: PlanningReviewStep[]) => {
    if (!component) return;
    onDraftChange({
      ...draft,
      components: draft.components.map((entry) =>
        entry.id === component.id ? { ...entry, configuration: { ...entry.configuration, steps: nextSteps } } : entry,
      ),
    });
  };

  const updateStep = (nextStep: PlanningReviewStep) => {
    if (selectedStep === null) return;
    updateSteps(steps.map((step, index) => (index === selectedStep ? nextStep : step)));
  };

  const addStep = () => {
    const nextStep: PlanningReviewStep = { name: "New step", type: "bash", command: "" };
    updateSteps([...steps, nextStep]);
    setSelectedStep(steps.length);
  };

  const removeStepAt = (position: number) => {
    updateSteps(steps.filter((_, index) => index !== position));
    if (selectedStep === position) {
      setSelectedStep(null);
    } else if (selectedStep !== null && selectedStep > position) {
      setSelectedStep(selectedStep - 1);
    }
  };

  const moveStepAt = (position: number, offset: -1 | 1) => {
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
  };

  return (
    <div className="flex min-h-0 flex-1 flex-col" data-testid="custom-verification-setup">
      <div className="flex min-h-0 flex-1">
        <AutomationFlow
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
                    onClick={() => setConfirmDelete(false)}
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
                  onClick={() => setConfirmDelete(true)}
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
              data-testid="custom-verification-save"
            >
              Save
            </LoadingButton>
          </div>
        </div>
      </footer>
    </div>
  );
}

function AutomationFlow({
  mode,
  steps,
  selectedStep,
  onSelectTrigger,
  onSelectStep,
  onMoveStep,
  onRemoveStep,
  onAddStep,
}: {
  mode: CustomVerificationMode;
  steps: PlanningReviewStep[];
  selectedStep: number | null;
  onSelectTrigger: () => void;
  onSelectStep: (index: number) => void;
  onMoveStep: (index: number, offset: -1 | 1) => void;
  onRemoveStep: (index: number) => void;
  onAddStep: () => void;
}) {
  return (
    <aside className="relative min-h-0 w-[42%] shrink-0 overflow-y-auto border-r border-border bg-muted/20">
      <div className="absolute inset-0 bg-[radial-gradient(circle,var(--border)_1px,transparent_1px)] [background-size:18px_18px] opacity-50" />
      <div className="relative mx-auto flex w-full max-w-lg flex-col items-center px-8 py-10">
        <FlowCard selected={selectedStep === null} onClick={onSelectTrigger}>
          <FlowBadge kind="trigger" />
          <p className="mt-3 text-sm font-semibold text-foreground">
            {mode === "automatic" ? "Every pull request commit" : "Manual"}
          </p>
          <p className="mt-1 text-xs text-muted-foreground">
            {mode === "automatic" ? "Runs for every new pull request commit." : "Runs when you start it from a task."}
          </p>
        </FlowCard>
        {steps.map((step, index) => (
          <div className="flex w-full flex-col items-center" key={index}>
            <FlowConnector />
            <FlowCard
              selected={selectedStep === index}
              onClick={() => onSelectStep(index)}
              actions={
                <StepCardActions
                  position={index}
                  stepCount={steps.length}
                  onMove={(offset) => onMoveStep(index, offset)}
                  onRemove={() => onRemoveStep(index)}
                />
              }
            >
              <FlowBadge kind={step.type} />
              <p className="mt-3 truncate text-sm font-semibold text-foreground">{step.name || `Step ${index + 1}`}</p>
              <p className="mt-1 truncate text-xs text-muted-foreground">
                {step.type === "prompt" ? mergeCheckPromptBody(step.prompt) : step.command || "No commands yet."}
              </p>
            </FlowCard>
          </div>
        ))}
        <FlowConnector />
        <Button type="button" variant="outline" size="sm" className="gap-1.5 bg-background" onClick={onAddStep}>
          <Plus className="size-3.5" aria-hidden />
          Add step
        </Button>
      </div>
    </aside>
  );
}

function FlowCard({
  selected,
  onClick,
  actions,
  children,
}: {
  selected: boolean;
  onClick: () => void;
  actions?: ReactNode;
  children: ReactNode;
}) {
  return (
    <div className="group/flow-card relative w-full">
      <button
        type="button"
        onClick={onClick}
        className={cn(
          "w-full rounded-xl border bg-card px-4 py-4 text-left shadow-sm transition-colors hover:border-foreground/30",
          selected ? "border-primary ring-2 ring-primary/15" : "border-border",
        )}
      >
        {children}
      </button>
      {actions ? (
        <div className="absolute top-3 right-3 flex items-center opacity-0 transition-opacity group-focus-within/flow-card:opacity-100 group-hover/flow-card:opacity-100">
          {actions}
        </div>
      ) : null}
    </div>
  );
}

function StepCardActions({
  position,
  stepCount,
  onMove,
  onRemove,
}: {
  position: number;
  stepCount: number;
  onMove: (offset: -1 | 1) => void;
  onRemove: () => void;
}) {
  return (
    <>
      <Button
        type="button"
        variant="ghost"
        size="icon-xs"
        className="size-6 text-muted-foreground/60 hover:bg-muted hover:text-foreground"
        aria-label="Move step up"
        disabled={position === 0}
        onClick={() => onMove(-1)}
      >
        <ArrowUp className="size-3.5" aria-hidden />
      </Button>
      <Button
        type="button"
        variant="ghost"
        size="icon-xs"
        className="size-6 text-muted-foreground/60 hover:bg-muted hover:text-foreground"
        aria-label="Move step down"
        disabled={position === stepCount - 1}
        onClick={() => onMove(1)}
      >
        <ArrowDown className="size-3.5" aria-hidden />
      </Button>
      <Button
        type="button"
        variant="ghost"
        size="icon-xs"
        className="size-6 text-muted-foreground/60 hover:bg-destructive/10 hover:text-destructive"
        aria-label="Remove step"
        onClick={onRemove}
      >
        <Trash2 className="size-3.5" aria-hidden />
      </Button>
    </>
  );
}

function FlowConnector() {
  return (
    <div className="flex h-8 w-6 items-center justify-center" aria-hidden>
      <span className="h-full w-0.5 bg-foreground/30" />
    </div>
  );
}

function FlowBadge({ kind }: { kind: PlanningReviewStepKind | "trigger" }) {
  const Icon = kind === "trigger" ? Zap : kind === "prompt" ? MessageSquareText : Terminal;
  const label = kind === "trigger" ? "Trigger" : kind === "prompt" ? "Prompt" : "Bash";
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-[11px] font-medium",
        kind === "trigger" && "border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-400",
        kind === "prompt" && "border-violet-500/30 bg-violet-500/10 text-violet-700 dark:text-violet-400",
        kind === "bash" && "border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-400",
      )}
    >
      <Icon className="size-3" aria-hidden />
      {label}
    </span>
  );
}

function TriggerEditor({
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
  name: string;
  icon: string;
  mode: CustomVerificationMode;
  draft: PlanningReviewDraft;
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  onNameChange: (name: string) => void;
  onIconChange: (icon: string) => void;
  onModeChange: (mode: CustomVerificationMode) => void;
  onDraftChange: (draft: PlanningReviewDraft) => void;
}) {
  return (
    <div className="mx-auto flex w-full max-w-3xl flex-col gap-8 px-8 py-10">
      <h3 className="text-base font-semibold text-foreground">Edit Trigger</h3>
      <div>
        <Label htmlFor="custom-verification-mode" className="sr-only">
          Trigger
        </Label>
        <Select value={mode} onValueChange={(value) => onModeChange(value === "automatic" ? "automatic" : "manual")}>
          <SelectTrigger id="custom-verification-mode" className="min-h-14 w-full !h-auto py-2.5">
            <SelectValue>
              <TriggerOption mode={mode} />
            </SelectValue>
          </SelectTrigger>
          <SelectContent className="min-w-[var(--radix-select-trigger-width)]">
            <SelectItem value="manual" className="py-2.5">
              <TriggerOption mode="manual" />
            </SelectItem>
            <SelectItem value="automatic" className="py-2.5">
              <TriggerOption mode="automatic" />
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
        showConcurrency
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

function TriggerOption({ mode }: { mode: CustomVerificationMode }) {
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
          {automatic ? "Every pull request commit" : "Manual"}
        </span>
        <span className="text-xs font-normal text-muted-foreground">
          {automatic ? "Runs after each new commit." : "Runs only when you start it."}
        </span>
      </span>
    </span>
  );
}

function StepEditor({
  step,
  position,
  stepCount,
  onChange,
  onMove,
  onRemove,
  organizationId,
}: {
  step: PlanningReviewStep;
  position: number;
  stepCount: number;
  onChange: (step: PlanningReviewStep) => void;
  onMove: (offset: -1 | 1) => void;
  onRemove: () => void;
  organizationId: string;
}) {
  const value = step.type === "prompt" ? mergeCheckPromptBody(step.prompt) : (step.command ?? "");
  return (
    <div className="mx-auto flex w-full max-w-3xl flex-col gap-8 px-8 py-10">
      <div className="flex items-center gap-2">
        <FlowBadge kind={step.type} />
        <h3 className="text-base font-semibold text-foreground">Step settings</h3>
        <div className="ml-auto flex items-center gap-1">
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            aria-label="Move step up"
            disabled={position === 0}
            onClick={() => onMove(-1)}
          >
            <ArrowUp aria-hidden />
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            aria-label="Move step down"
            disabled={position === stepCount - 1}
            onClick={() => onMove(1)}
          >
            <ArrowDown aria-hidden />
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            className="text-muted-foreground hover:text-destructive"
            aria-label="Remove step"
            onClick={onRemove}
          >
            <Trash2 aria-hidden />
          </Button>
        </div>
      </div>
      <ImplementSettingRow
        label="Name"
        description="The name shown in the automation flow."
        htmlFor="custom-verification-step-name"
      >
        <Input
          id="custom-verification-step-name"
          value={step.name}
          onChange={(event) => onChange({ ...step, name: event.target.value })}
          placeholder="Step name"
          className="w-full shadow-none"
        />
      </ImplementSettingRow>
      <ImplementSettingRow
        label="Type"
        description="Choose an agent prompt or shell commands."
        htmlFor="custom-verification-step-kind"
      >
        <Select
          value={step.type}
          onValueChange={(value) =>
            onChange({
              ...step,
              type: value === "prompt" ? "prompt" : "bash",
              ...(value === "prompt"
                ? { prompt: step.prompt ?? step.command ?? "" }
                : { command: step.command ?? step.prompt ?? "" }),
            })
          }
        >
          <SelectTrigger id="custom-verification-step-kind" className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="prompt">Prompt</SelectItem>
            <SelectItem value="bash">Bash</SelectItem>
          </SelectContent>
        </Select>
      </ImplementSettingRow>
      <div className="flex flex-col gap-2">
        <div className="flex flex-col gap-1.5">
          <Label>{step.type === "prompt" ? "Prompt" : "Commands"}</Label>
          <p className="text-xs text-muted-foreground">
            {step.type === "prompt" ? "Tell the agent what to do in this step." : "Add shell commands for the runner."}
          </p>
        </div>
        <PlanningReviewStepBody
          kind={step.type}
          value={value}
          onChange={(next) =>
            onChange(
              step.type === "prompt"
                ? { ...step, prompt: withMergeCheckHeader(step.prompt, next) }
                : { ...step, command: next },
            )
          }
          label={`${step.type === "prompt" ? "Prompt" : "Commands"} for ${step.name || `Step ${position + 1}`}`}
          testId={`custom-verification-step-body-${position}`}
        />
      </div>
      {step.type === "bash" ? (
        <BashStepSecrets
          organizationId={organizationId}
          selected={step.secrets ?? []}
          onChange={(secrets) => onChange({ ...step, secrets })}
        />
      ) : null}
    </div>
  );
}

function BashStepSecrets({
  organizationId,
  selected,
  onChange,
}: {
  organizationId: string;
  selected: string[];
  onChange: (secrets: string[]) => void;
}) {
  const { canAct } = usePermissions();
  const canCreate = canAct("secrets", "create");
  const [createOpen, setCreateOpen] = useState(false);
  const [menuOpen, setMenuOpen] = useState(false);
  const { data: secrets = [], isLoading } = useSecrets(organizationId, "DOMAIN_TYPE_ORGANIZATION", organizationId);
  const options = secrets
    .map((secret) => secret.metadata?.name?.trim())
    .filter((name): name is string => Boolean(name))
    .filter((name) => !selected.includes(name))
    .sort((left, right) => left.localeCompare(right));
  const handleCreated = (created: CreatedSecretSummary) => {
    if (created.name && !selected.includes(created.name)) {
      onChange([...selected, created.name]);
    }
  };

  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-col gap-1.5">
        <Label>Secrets</Label>
        <p className="text-xs text-muted-foreground">Give this step access to stored credentials.</p>
      </div>
      <div className="flex flex-col gap-1">
        {selected.map((name) => (
          <div key={name} className="flex h-8 items-center gap-2 rounded-md bg-muted/40 px-2.5 text-sm">
            <KeyRound className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />
            <span className="min-w-0 flex-1 truncate">{name}</span>
            <Button
              type="button"
              variant="ghost"
              size="icon-xs"
              className="size-6 text-muted-foreground hover:text-foreground"
              aria-label={`Remove ${name}`}
              onClick={() => onChange(selected.filter((entry) => entry !== name))}
            >
              <X className="size-3.5" aria-hidden />
            </Button>
          </div>
        ))}
        <div className="relative w-fit">
          <Button
            type="button"
            variant="ghost"
            size="xs"
            className="h-7 justify-start gap-1 px-1.5 text-muted-foreground"
            aria-expanded={menuOpen}
            onClick={() => setMenuOpen((current) => !current)}
          >
            <Plus className="size-3" aria-hidden />
            Add
          </Button>
          {menuOpen ? (
            <div className="absolute top-full left-0 z-30 mt-1 min-w-56 rounded-md border bg-popover p-1 text-popover-foreground shadow-sm">
              {isLoading ? <p className="px-2 py-1.5 text-sm text-muted-foreground">Loading secrets…</p> : null}
              {canCreate ? (
                <Button
                  type="button"
                  variant="ghost"
                  className="h-8 w-full justify-start gap-2 px-2 font-normal"
                  onClick={() => {
                    setMenuOpen(false);
                    setCreateOpen(true);
                  }}
                >
                  <Plus className="size-3.5" aria-hidden />
                  Create new secret
                </Button>
              ) : null}
              {canCreate && options.length > 0 ? <div className="my-1 border-t border-border" /> : null}
              {!isLoading
                ? options.map((name) => (
                    <Button
                      key={name}
                      type="button"
                      variant="ghost"
                      className="h-8 w-full justify-start gap-2 px-2 font-normal"
                      onClick={() => {
                        onChange([...selected, name]);
                        setMenuOpen(false);
                      }}
                    >
                      <KeyRound className="size-3.5 text-muted-foreground" aria-hidden />
                      {name}
                    </Button>
                  ))
                : null}
              {!isLoading && options.length === 0 && !canCreate ? (
                <p className="px-2 py-1.5 text-sm text-muted-foreground">No organization secrets</p>
              ) : null}
            </div>
          ) : null}
        </div>
        {canCreate ? (
          <CreateSecretDialog
            open={createOpen}
            onOpenChange={setCreateOpen}
            organizationId={organizationId}
            onCreated={handleCreated}
          />
        ) : null}
      </div>
    </div>
  );
}
