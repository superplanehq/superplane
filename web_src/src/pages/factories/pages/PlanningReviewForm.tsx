import { useEffect, useRef } from "react";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { useByokRunnerModelDefault } from "@/hooks/useByokRunnerModelDefault";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { useFactoryAgentResources } from "@/hooks/useFactoryAgentResources";
import { FEATURE_WORKSPACE_SKILLS } from "@/lib/experimentalFeatures";
import { useComponent } from "@/hooks/useComponentData";
import { HOSTED_MODEL_ALL_PROVIDERS } from "@/lib/hostedLLMModels";
import { cn } from "@/lib/utils";
import { ConfigurationFieldRenderer } from "@/ui/configurationFieldRenderer";

import { factoryAgentSkillEditorPath } from "../lib/factoryPagePaths";
import { isMergeConfidenceSteps, newMergeConfidenceStep } from "./mergeConfidenceSteps";
import type { PlanningReviewComponent, PlanningReviewDraft, PlanningReviewStep } from "./planningReviewMockup";
import { planningReviewModelUsedField } from "./planningReviewRunnerFields";
import { disabledAgentResourceIds } from "./disabledAgentResourceIds";
import { disabledAgentResourceTools, enabledAgentResourceTools } from "./PlanningReviewDisabledTools";
import { PlanningReviewResourcesCard } from "./PlanningReviewResourcesCard";
import { PlanningReviewStepList } from "./PlanningReviewStepList";
import { ImplementRunDuration } from "./ImplementRunDuration";
import { ImplementSettingRow } from "./ImplementSettingRow";

const FLAT_FIELD_CONTROL =
  "h-auto min-h-10 !rounded-lg border-foreground/20 bg-card px-3 py-2.5 shadow-none hover:bg-card dark:border-foreground/20 dark:bg-card";

const EXPRESSION_CONTEXT = {
  data: { branch: "feature/planning-review" },
  order: { description: "Add a simple planning review editor." },
  previous: { data: { result: { branch: "feature/planning-review" } } },
};

export function PlanningReviewForm({
  draft,
  onChange,
  organizationId,
  factoryId,
  factoryKey,
  showVisualEvidenceSetting = false,
  showResources = true,
  showConcurrency = true,
  appearance = "card",
  view = "full",
  onRestoreDefaultPrompt,
  restoreDefaultPromptDisabled = false,
  restoreError,
  onRetryRestore,
  restoreRetryDisabled = false,
  disabled = false,
}: {
  draft: PlanningReviewDraft;
  onChange: (next: PlanningReviewDraft) => void;
  organizationId?: string;
  factoryId?: string;
  factoryKey?: string;
  showVisualEvidenceSetting?: boolean;
  showResources?: boolean;
  showConcurrency?: boolean;
  /** `fields` matches the pull request step. `card` is the GitHub agent settings. */
  appearance?: "card" | "fields";
  view?: "full" | "implementation" | "planning";
  onRestoreDefaultPrompt?: () => void;
  restoreDefaultPromptDisabled?: boolean;
  restoreError?: string;
  onRetryRestore?: () => void;
  restoreRetryDisabled?: boolean;
  disabled?: boolean;
}) {
  const updateComponent = (id: string, next: PlanningReviewComponent) => {
    onChange({
      ...draft,
      components: draft.components.map((component) => (component.id === id ? next : component)),
    });
  };

  return (
    <div className={appearance === "fields" ? "flex flex-col gap-8" : "flex flex-col gap-4"}>
      {draft.components.map((component) => (
        <AgentPanel
          key={component.id}
          appearance={appearance}
          view={view}
          component={component}
          organizationId={organizationId}
          factoryId={factoryId}
          factoryKey={factoryKey}
          showVisualEvidenceSetting={showVisualEvidenceSetting}
          showResources={showResources}
          showConcurrency={showConcurrency}
          onRestoreDefaultPrompt={onRestoreDefaultPrompt}
          restoreDefaultPromptDisabled={restoreDefaultPromptDisabled}
          restoreError={restoreError}
          onRetryRestore={onRetryRestore}
          restoreRetryDisabled={restoreRetryDisabled}
          disabled={disabled}
          onChange={(next) => updateComponent(component.id, next)}
        />
      ))}
    </div>
  );
}

function AgentPanel({
  appearance,
  view,
  component,
  organizationId,
  factoryId,
  factoryKey,
  showVisualEvidenceSetting,
  showResources,
  showConcurrency,
  onRestoreDefaultPrompt,
  restoreDefaultPromptDisabled,
  restoreError,
  onRetryRestore,
  restoreRetryDisabled,
  disabled,
  onChange,
}: {
  appearance: "card" | "fields";
  view: "full" | "implementation" | "planning";
  component: PlanningReviewComponent;
  organizationId?: string;
  factoryId?: string;
  factoryKey?: string;
  showVisualEvidenceSetting: boolean;
  showResources: boolean;
  showConcurrency: boolean;
  onRestoreDefaultPrompt?: () => void;
  restoreDefaultPromptDisabled: boolean;
  restoreError?: string;
  onRetryRestore?: () => void;
  restoreRetryDisabled: boolean;
  disabled: boolean;
  onChange: (next: PlanningReviewComponent) => void;
}) {
  const features = useExperimentalFeature(organizationId);
  const skillsEnabled = Boolean(
    view === "full" && organizationId && factoryId && features.has(FEATURE_WORKSPACE_SKILLS),
  );
  const skills = useFactoryAgentResources(organizationId ?? "", factoryId ?? "", "KIND_SKILL", skillsEnabled);
  const disabledSkills = disabledAgentResourceIds(component.configuration);
  const promptSkills = skillsEnabled
    ? (skills.data ?? []).filter(
        (skill) =>
          skill.enabled !== false &&
          skill.name &&
          skill.name !== "superplane" &&
          (skill.systemSkill || !disabledSkills.includes(skill.id ?? "")),
      )
    : undefined;
  const componentRef = useRef(component);
  const setConfigurationField = (name: string, value: unknown) => {
    const current = componentRef.current;
    const next = {
      ...current,
      configuration: { ...current.configuration, [name]: value },
    };
    componentRef.current = next;
    onChange(next);
  };
  const { data: action } = useComponent(organizationId ?? "", component.component ?? "");
  const modelUsedField = planningReviewModelUsedField(component.component, action?.configuration);
  const modelProvider = modelUsedField?.typeOptions?.hostedModel?.provider ?? "";
  const currentModel = typeof component.configuration.model === "string" ? component.configuration.model : "";
  const byokModelDefault = useByokRunnerModelDefault({
    enabled:
      modelUsedField?.type === "hosted-model" && modelProvider !== "" && modelProvider !== HOSTED_MODEL_ALL_PROVIDERS,
    organizationId,
    factoryId,
    provider: modelProvider,
    current: currentModel,
  });

  componentRef.current = component;
  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;

  useEffect(() => {
    if (!byokModelDefault) {
      return;
    }
    const current = componentRef.current;
    onChangeRef.current({
      ...current,
      configuration: { ...current.configuration, model: byokModelDefault },
    });
  }, [byokModelDefault]);

  const modelControl = modelUsedField ? (
    <ConfigurationFieldRenderer
      field={view !== "full" ? { ...modelUsedField, label: "Default model" } : modelUsedField}
      hideLabel={appearance === "fields"}
      triggerClassName={
        view !== "full" ? "h-8 w-full shadow-none" : appearance === "fields" ? FLAT_FIELD_CONTROL : undefined
      }
      value={byokModelDefault ?? component.configuration.model}
      onChange={(value) => setConfigurationField("model", value)}
      onValuesChange={(patch) => {
        const configuration = { ...component.configuration, ...patch };
        if (!patch.thinkingLevel) {
          delete configuration.thinkingLevel;
        }
        onChange({ ...component, configuration });
      }}
      allValues={component.configuration}
      organizationId={organizationId}
      factoryId={factoryId}
      allowExpressions={view === "full"}
      autocompleteExampleObj={EXPRESSION_CONTEXT}
      fieldPath="model"
      readOnly={disabled}
    />
  ) : null;

  const concurrencyControl = (
    <Input
      id={`planning-review-concurrency-max-${component.id}`}
      name="implementationConcurrency"
      data-testid={`planning-review-concurrency-max-${component.id}`}
      type="number"
      min={1}
      value={component.concurrency.max}
      disabled={disabled}
      onChange={(event) =>
        onChange({ ...component, concurrency: { ...component.concurrency, max: event.target.value } })
      }
      className={view !== "full" ? "w-24 tabular-nums shadow-none max-sm:text-base" : "shadow-none"}
      aria-describedby={view !== "full" ? `planning-review-concurrency-max-${component.id}-help` : undefined}
    />
  );

  return (
    <div
      className={appearance === "fields" ? "flex flex-col gap-8" : "flex flex-col gap-4"}
      data-testid={`planning-review-component-${component.id}`}
    >
      <section
        className={
          appearance === "fields"
            ? view !== "full"
              ? "flex flex-col gap-8"
              : "flex flex-col gap-6"
            : cn(
                "grid gap-x-6 gap-y-4 rounded-xl border border-border bg-card px-5 py-4 shadow-sm",
                showVisualEvidenceSetting ? "grid-cols-3" : "grid-cols-2",
              )
        }
        data-testid="planning-review-settings"
      >
        {view !== "full" && modelControl ? (
          <div role="group" aria-label="Default model">
            <ImplementSettingRow
              label="Default model"
              description={view === "planning" ? "The model used to plan tasks." : "The model used to implement tasks."}
            >
              <div className="w-full min-w-0">{modelControl}</div>
            </ImplementSettingRow>
          </div>
        ) : null}
        {showConcurrency ? (
          view !== "full" ? (
            <ImplementSettingRow
              label="Maximum simultaneous runs"
              description="Extra runs wait until a running task finishes."
              htmlFor={`planning-review-concurrency-max-${component.id}`}
            >
              {concurrencyControl}
              <p aria-hidden className="w-14 text-base text-muted-foreground sm:text-sm">
                runs
              </p>
            </ImplementSettingRow>
          ) : (
            <div className="flex flex-col gap-2">
              <Label htmlFor={`planning-review-concurrency-max-${component.id}`}>Concurrency</Label>
              {concurrencyControl}
            </div>
          )
        ) : null}
        {view === "implementation" ? (
          <ImplementRunDuration
            seconds={component.configuration.executionTimeoutSeconds}
            disabled={disabled}
            onChange={(seconds) => setConfigurationField("executionTimeoutSeconds", seconds)}
          />
        ) : view === "full" && modelControl ? (
          <div className={appearance === "fields" ? "flex flex-col gap-2" : undefined}>
            {appearance === "fields" ? <h3 className="workspace-section-title">Model</h3> : null}
            {modelControl}
          </div>
        ) : null}
        {showVisualEvidenceSetting ? (
          <div className="flex flex-col gap-2">
            <Label htmlFor={`planning-review-visual-evidence-${component.id}`}>Include visual evidence</Label>
            <div className="flex h-9 items-center">
              <Switch
                id={`planning-review-visual-evidence-${component.id}`}
                checked={component.configuration.includeVisualEvidence === true}
                onCheckedChange={(checked) => setConfigurationField("includeVisualEvidence", checked)}
              />
            </div>
          </div>
        ) : null}
      </section>
      {view === "full" ? (
        <PlanningReviewStepList
          skills={promptSkills}
          skillHref={(skill) =>
            organizationId && factoryKey && skill.id
              ? factoryAgentSkillEditorPath(organizationId, factoryKey, skill.id)
              : undefined
          }
          appearance={appearance}
          steps={(component.configuration.steps as PlanningReviewStep[]) ?? []}
          onChange={(steps) => setConfigurationField("steps", steps)}
          createStep={
            isMergeConfidenceSteps((component.configuration.steps as PlanningReviewStep[]) ?? [])
              ? newMergeConfidenceStep
              : undefined
          }
          onRestoreDefaultPrompt={onRestoreDefaultPrompt}
          restoreDefaultPromptDisabled={restoreDefaultPromptDisabled}
          restoreError={restoreError}
          onRetryRestore={onRetryRestore}
          restoreRetryDisabled={restoreRetryDisabled}
        />
      ) : null}
      {showResources ? (
        <PlanningReviewResourcesCard
          appearance={appearance}
          organizationId={organizationId}
          factoryId={factoryId}
          factoryKey={factoryKey}
          disabledIds={disabledAgentResourceIds(component.configuration)}
          disabledTools={disabledAgentResourceTools(component.configuration)}
          enabledTools={enabledAgentResourceTools(component.configuration)}
          onDisabledIdsChange={(ids) => setConfigurationField("disabledAgentResourceIds", ids)}
          onDisabledToolsChange={(tools) => setConfigurationField("disabledAgentResourceTools", tools)}
          onEnabledToolsChange={(tools) => setConfigurationField("enabledAgentResourceTools", tools)}
        />
      ) : null}
    </div>
  );
}
