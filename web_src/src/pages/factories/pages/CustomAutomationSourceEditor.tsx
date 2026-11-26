import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { useId, useState } from "react";
import { useCustomAutomationEditorAgent } from "@/hooks/useCustomAutomationEditorAgent";
import { useFactoryCustomAutomation } from "@/hooks/useFactoryCustomAutomationData";
import type { useCustomAutomationSourceDraft } from "@/hooks/useCustomAutomationSourceDraft";
import { useSelectableLLMModels } from "@/hooks/useSelectableLLMModels";
import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import {
  customAutomationRunContext,
  customAutomationTriggersForRun,
  newCustomAutomationTrigger,
  type CustomAutomationColumn,
  type CustomAutomationTrigger,
} from "../lib/customAutomation";
import {
  automationDefinitionCanvas,
  automationDefinitionSteps,
  automationDefinitionTriggers,
  type AutomationDefinition,
} from "../lib/customAutomationDefinition";
import { hostedModelForYaml, type AutomationHostedModels } from "../lib/customAutomationHostedModels";
import { CustomAutomationSetupFooter, CustomAutomationSetupForm } from "./CustomAutomationSetupForm";
import type { PlanningReviewDraft, PlanningReviewStep } from "./planningReviewMockup";

export function CustomAutomationSourceEditor({
  model,
  automationId,
  stage,
  saving,
  deleting,
  error,
  onSave,
  onDelete,
  onClose,
}: {
  model: ReturnType<typeof useCustomAutomationSourceDraft>;
  automationId?: string;
  stage: CustomAutomationColumn;
  saving: boolean;
  deleting: boolean;
  error: string;
  onSave: () => void;
  onDelete?: () => void;
  onClose: () => void;
}) {
  const { organizationId, factoryId, routeSegment, factory } = useFactoriesLayout();
  const yamlId = useId();
  const definition = model.definition;
  const live = useFactoryCustomAutomation(organizationId, factoryId, automationId);
  const { agent, customProvider, ready } = useCustomAutomationEditorAgent(
    organizationId,
    factory?.onboarding,
    automationId,
    live.isFetchedAfterMount && !live.isError ? live.data?.settings?.agent : undefined,
  );
  const [formTriggers, setFormTriggers] = useState<CustomAutomationTrigger[]>();

  const isHosted = agent?.component === "runnerSuperPlane";
  const { data: hostedModels } = useSelectableLLMModels(organizationId, {
    factoryId,
    sources: ["hosted"],
    enabled: isHosted && !model.isYamlOpen,
  });
  const hosted = isHosted ? { models: hostedModels ?? [], savedKey: agent.model } : undefined;

  const form = { saving, deleting, onSave, onDelete, onClose };
  if (model.isYamlOpen || !definition) {
    return (
      <>
        <div className="flex min-h-0 flex-1 flex-col gap-2 p-5">
          <Label htmlFor={yamlId}>Automation YAML</Label>
          <Textarea
            id={yamlId}
            value={model.source}
            onChange={(event) => {
              setFormTriggers(undefined);
              model.editYaml(event.target.value);
            }}
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

  const visual = automationDefinitionCanvas(definition, agent, customProvider, hosted);
  const triggers = formTriggers ?? visual.triggers;

  const editTriggers = (next: CustomAutomationTrigger[], selected?: boolean) => {
    // Keep hidden triggers in the form draft when switching between task and column runs.
    setFormTriggers(next);
    const runsOnTask = customAutomationRunContext(stage, next, selected);
    const active = customAutomationTriggersForRun(next, runsOnTask);
    model.editCanvas({
      runs_on_task: runsOnTask,
      triggers: automationDefinitionTriggers(active.length ? active : [newCustomAutomationTrigger()], stage),
    });
  };
  return (
    <CustomAutomationSetupForm
      {...form}
      {...visual}
      triggers={triggers}
      showModel={ready}
      organizationId={organizationId}
      factoryId={factoryId}
      factoryKey={routeSegment}
      columnKey={stage}
      asCode
      error={error}
      onNameChange={(name) => model.editCanvas({ name })}
      onIconChange={(icon) => model.editCanvas({ icon })}
      onTriggersChange={(next) => editTriggers(next, visual.runsOnTask)}
      onRunsOnTaskChange={(runsOnTask) => editTriggers(triggers, runsOnTask)}
      onDraftChange={(next) => model.editCanvas(settingsEdit(visual.draft, next, definition.agent, hosted))}
    />
  );
}

function settingsEdit(
  previous: PlanningReviewDraft,
  next: PlanningReviewDraft,
  agent: AutomationDefinition["agent"],
  hosted?: AutomationHostedModels,
): Partial<AutomationDefinition> {
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
  if (after.skipSetupCommands !== before.skipSetupCommands) {
    patch.skip_setup_commands = after.skipSetupCommands === true;
  }
  if (after.skipRepositoryCheckout !== before.skipRepositoryCheckout) {
    patch.skip_repository_checkout = after.skipRepositoryCheckout === true;
  }
  if (after.model !== before.model) {
    const model = hostedModelForYaml(after.model, hosted);
    patch.agent = model || agent?.harness ? { ...agent, model } : undefined;
  }
  return patch;
}
