import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { useId } from "react";
import type { useCustomAutomationSourceDraft } from "@/hooks/useCustomAutomationSourceDraft";
import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import {
  customAutomationTriggersForRun,
  newCustomAutomationTrigger,
  type CustomAutomationColumn,
} from "../lib/customAutomation";
import {
  automationDefinitionCanvas,
  automationDefinitionSteps,
  automationDefinitionTriggers,
  type AutomationDefinition,
} from "../lib/customAutomationDefinition";
import { CustomAutomationSetupFooter, CustomAutomationSetupForm } from "./CustomAutomationSetupForm";
import type { PlanningReviewDraft, PlanningReviewStep } from "./planningReviewMockup";

export function CustomAutomationSourceEditor({
  model,
  stage,
  saving,
  deleting,
  error,
  onSave,
  onDelete,
  onClose,
}: {
  model: ReturnType<typeof useCustomAutomationSourceDraft>;
  stage: CustomAutomationColumn;
  saving: boolean;
  deleting: boolean;
  error: string;
  onSave: () => void;
  onDelete?: () => void;
  onClose: () => void;
}) {
  const { organizationId, factoryId, routeSegment } = useFactoriesLayout();
  const yamlId = useId();
  const definition = model.definition;
  const form = { saving, deleting, onSave, onDelete, onClose };
  if (model.isYamlOpen || !definition) {
    return (
      <>
        <div className="flex min-h-0 flex-1 flex-col gap-2 p-5">
          <Label htmlFor={yamlId}>Automation YAML</Label>
          <Textarea
            id={yamlId}
            value={model.source}
            onChange={(event) => model.editYaml(event.target.value)}
            readOnly={saving || deleting}
            spellCheck={false}
            data-global-shortcuts="ignore"
            className="min-h-0 flex-1 resize-none font-mono"
          />
        </div>
        <CustomAutomationSetupFooter {...form} error={error || model.problem} />
      </>
    );
  }

  const visual = automationDefinitionCanvas(definition);
  return (
    <CustomAutomationSetupForm
      {...form}
      {...visual}
      organizationId={organizationId}
      factoryId={factoryId}
      factoryKey={routeSegment}
      columnKey={stage}
      asCode
      error={error}
      onNameChange={(name) => model.editCanvas({ name })}
      onIconChange={(icon) => model.editCanvas({ icon })}
      onTriggersChange={(triggers) => model.editCanvas({ triggers: automationDefinitionTriggers(triggers, stage) })}
      onRunsOnTaskChange={(runsOnTask) => {
        const compatible = customAutomationTriggersForRun(visual.triggers, runsOnTask);
        model.editCanvas({
          runs_on_task: runsOnTask,
          triggers: automationDefinitionTriggers(
            compatible.length > 0 ? compatible : [newCustomAutomationTrigger()],
            stage,
          ),
        });
      }}
      onDraftChange={(next) => model.editCanvas(settingsEdit(visual.draft, next))}
    />
  );
}

function settingsEdit(previous: PlanningReviewDraft, next: PlanningReviewDraft): Partial<AutomationDefinition> {
  const before = previous.components[0]?.configuration;
  const after = next.components[0]?.configuration;
  if (!before || !after) return {};
  const patch: Partial<AutomationDefinition> = {};
  if (after.steps !== before.steps && Array.isArray(after.steps)) {
    patch.steps = automationDefinitionSteps(after.steps as PlanningReviewStep[]);
  }
  if (
    after.executionTimeoutSeconds !== before.executionTimeoutSeconds &&
    typeof after.executionTimeoutSeconds === "number"
  ) {
    patch.timeout_seconds = after.executionTimeoutSeconds;
  }
  if (after.model !== before.model && typeof after.model === "string") {
    patch.agent = { harness: "superplane", model: after.model || undefined };
  }
  return patch;
}
