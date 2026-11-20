import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { useId, useState } from "react";
import { useConnectedIntegrations } from "@/hooks/useIntegrations";
import type { useCustomAutomationSourceDraft } from "@/hooks/useCustomAutomationSourceDraft";
import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import {
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
  const { organizationId, factoryId, routeSegment, factory } = useFactoriesLayout();
  const yamlId = useId();
  const definition = model.definition;
  const agent = definition?.agent;
  const onboarding = factory?.onboarding;
  const credentials = agent?.credentials;
  const needsWorkspaceConnection = Boolean(onboarding?.agentIntegrationId && !credentials);
  const connections = useConnectedIntegrations(organizationId, {
    enabled: needsWorkspaceConnection || (agent?.harness === "openrouter" && credentials?.source === "integration"),
  });
  const [formTriggers, setFormTriggers] = useState<CustomAutomationTrigger[]>();

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

  const { modelAgent, ...visual } = automationDefinitionCanvas(definition, onboarding, connections.data);
  const triggers = formTriggers ?? visual.triggers;

  const editTriggers = (next: CustomAutomationTrigger[], runsOnTask?: boolean) => {
    // Keep hidden triggers in the form draft when switching between task and column runs.
    setFormTriggers(next);
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
      showModel={Boolean(modelAgent)}
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
      onDraftChange={(next) => model.editCanvas(settingsEdit(visual.draft, next, modelAgent))}
    />
  );
}

function settingsEdit(
  previous: PlanningReviewDraft,
  next: PlanningReviewDraft,
  agent: AutomationDefinition["agent"],
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
  if (agent && after.model !== before.model && typeof after.model === "string") {
    patch.agent = { ...agent, model: after.model || undefined };
  }
  return patch;
}
