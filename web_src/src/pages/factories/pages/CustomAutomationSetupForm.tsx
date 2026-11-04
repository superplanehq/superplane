import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { CircleAlert } from "lucide-react";
import { useState } from "react";

import { type CustomAutomationColumn, type CustomAutomationTrigger } from "../lib/customAutomation";
import { AutomationFlow } from "./customAutomationSetupFlow";
import { StepEditor } from "./customAutomationStepEditor";
import { TriggerEditor } from "./CustomAutomationTriggerEditor";
import type { PlanningReviewDraft, PlanningReviewStep } from "./planningReviewMockup";

type CustomAutomationSetupFormProps = {
  name: string;
  icon: string;
  triggers: CustomAutomationTrigger[];
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
  onTriggersChange: (triggers: CustomAutomationTrigger[]) => void;
  onDraftChange: (draft: PlanningReviewDraft) => void;
  onSave: () => void;
  onDelete?: () => void;
  onClose: () => void;
};

export function CustomAutomationSetupForm(props: CustomAutomationSetupFormProps) {
  const {
    name,
    icon,
    triggers,
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
    onTriggersChange,
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
          triggers={triggers}
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
              triggers={triggers}
              onTriggersChange={onTriggersChange}
              draft={draft}
              organizationId={organizationId}
              factoryId={factoryId}
              factoryKey={factoryKey}
              onNameChange={onNameChange}
              onIconChange={onIconChange}
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
