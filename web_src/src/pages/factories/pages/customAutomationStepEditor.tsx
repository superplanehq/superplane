import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { usePermissions } from "@/contexts/usePermissions";
import { useSecrets } from "@/hooks/useSecrets";
import { CreateSecretDialog, type CreatedSecretSummary } from "@/ui/CreateSecretDialog";
import { ArrowDown, ArrowUp, KeyRound, Plus, Trash2, X } from "lucide-react";
import { useState } from "react";

import { FlowBadge } from "./customAutomationSetupFlow";
import { ImplementSettingRow } from "./ImplementSettingRow";
import { mergeCheckPromptBody, withMergeCheckHeader } from "./mergeConfidenceSteps";
import { PlanningReviewStepBody } from "./PlanningReviewStepBody";
import type { PlanningReviewStep } from "./planningReviewMockup";

export function StepEditor({
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
      <StepEditorToolbar
        position={position}
        stepCount={stepCount}
        type={step.type}
        onMove={onMove}
        onRemove={onRemove}
      />
      <ImplementSettingRow
        label="Name"
        description="The name shown in the automation flow."
        htmlFor="custom-automation-step-name"
      >
        <Input
          id="custom-automation-step-name"
          value={step.name}
          onChange={(event) => onChange({ ...step, name: event.target.value })}
          placeholder="Step name"
          className="w-full shadow-none"
        />
      </ImplementSettingRow>
      <ImplementSettingRow
        label="Type"
        description="Choose an agent prompt or shell commands."
        htmlFor="custom-automation-step-kind"
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
          <SelectTrigger id="custom-automation-step-kind" className="w-full">
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
          testId={`custom-automation-step-body-${position}`}
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

function StepEditorToolbar({
  position,
  stepCount,
  type,
  onMove,
  onRemove,
}: {
  position: number;
  stepCount: number;
  type: PlanningReviewStep["type"];
  onMove: (offset: -1 | 1) => void;
  onRemove: () => void;
}) {
  return (
    <div className="flex items-center gap-2">
      <FlowBadge kind={type} />
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
        <SecretPickerMenu
          canCreate={canCreate}
          isLoading={isLoading}
          options={options}
          menuOpen={menuOpen}
          onMenuOpenChange={setMenuOpen}
          onCreate={() => setCreateOpen(true)}
          onSelect={(name) => onChange([...selected, name])}
        />
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

function SecretPickerMenu({
  canCreate,
  isLoading,
  options,
  menuOpen,
  onMenuOpenChange,
  onCreate,
  onSelect,
}: {
  canCreate: boolean;
  isLoading: boolean;
  options: string[];
  menuOpen: boolean;
  onMenuOpenChange: (open: boolean) => void;
  onCreate: () => void;
  onSelect: (name: string) => void;
}) {
  return (
    <div className="relative w-fit">
      <Button
        type="button"
        variant="ghost"
        size="xs"
        className="h-7 justify-start gap-1 px-1.5 text-muted-foreground"
        aria-expanded={menuOpen}
        onClick={() => onMenuOpenChange(!menuOpen)}
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
                onMenuOpenChange(false);
                onCreate();
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
                    onSelect(name);
                    onMenuOpenChange(false);
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
  );
}
