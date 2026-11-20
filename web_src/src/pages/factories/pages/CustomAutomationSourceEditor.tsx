import { Label } from "@/components/ui/label";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import { useId } from "react";
import type { useCustomAutomationSourceDraft } from "@/hooks/useCustomAutomationSourceDraft";
import { useConnectedIntegrations } from "@/hooks/useIntegrations";
import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import {
  customAutomationTriggersForRun,
  newCustomAutomationTrigger,
  workspaceAgentFromOnboarding,
  type CustomAutomationColumn,
} from "../lib/customAutomation";
import {
  automationDefinitionCanvas,
  automationDefinitionSteps,
  automationDefinitionTriggers,
  automationHarnessForComponent,
  type AutomationDefinition,
  type AutomationHarness,
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
  const connections = useConnectedIntegrations(organizationId, {
    enabled: agent?.harness === "openrouter" && agent.credentials?.source === "integration",
  });
  const workspaceComponent = workspaceAgentFromOnboarding(factory?.onboarding)?.component;
  const visual = definition ? automationDefinitionCanvas(definition, workspaceComponent, connections.data) : undefined;
  const form = { saving, deleting, onSave, onDelete, onClose };
  return (
    <Tabs value={model.tab} onValueChange={model.switchTab} activationMode="manual" className="min-h-0 flex-1">
      <div className="shrink-0 border-b border-border px-5 py-3">
        <TabsList aria-label="Automation editor">
          <TabsTrigger value="canvas" disabled={saving || deleting}>
            Canvas
          </TabsTrigger>
          <TabsTrigger value="yaml" disabled={saving || deleting}>
            YAML
          </TabsTrigger>
        </TabsList>
      </div>
      <TabsContent value="canvas" className="min-h-0 flex flex-col data-[state=inactive]:hidden">
        {model.tab === "canvas" && definition && visual ? (
          <CustomAutomationSetupForm
            {...form}
            {...visual}
            organizationId={organizationId}
            factoryId={factoryId}
            factoryKey={routeSegment}
            columnKey={stage}
            asCode={{ agent: definition.agent, onAgentChange: (agent) => model.editCanvas({ agent }) }}
            error={error}
            onNameChange={(name) => model.editCanvas({ name })}
            onIconChange={(icon) => model.editCanvas({ icon })}
            onTriggersChange={(triggers) =>
              model.editCanvas({ triggers: automationDefinitionTriggers(triggers, stage) })
            }
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
            onDraftChange={(next) =>
              model.editCanvas(
                settingsEdit(definition, visual.draft, next, automationHarnessForComponent(workspaceComponent)),
              )
            }
          />
        ) : null}
      </TabsContent>
      <TabsContent value="yaml" className="min-h-0 flex flex-col data-[state=inactive]:hidden">
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
      </TabsContent>
    </Tabs>
  );
}

function settingsEdit(
  definition: AutomationDefinition,
  previous: PlanningReviewDraft,
  next: PlanningReviewDraft,
  workspaceHarness?: AutomationHarness,
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
  const harness = definition.agent?.harness ?? workspaceHarness;
  if (harness && after.model !== before.model && typeof after.model === "string") {
    patch.agent = { ...definition.agent, harness, model: after.model || undefined };
  }
  return patch;
}
