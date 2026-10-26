import { useEffect, useRef, type ReactNode } from "react";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useByokRunnerModelDefault } from "@/hooks/useByokRunnerModelDefault";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { useFactoryAgentResources } from "@/hooks/useFactoryAgentResources";
import { FEATURE_WORKSPACE_SKILLS } from "@/lib/experimentalFeatures";
import { useComponent } from "@/hooks/useComponentData";
import { HOSTED_MODEL_ALL_PROVIDERS } from "@/lib/hostedLLMModels";
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
  showResources = true,
  showConcurrency = true,
  showSteps = true,
  modelLabel,
  modelDescription,
  modelOnly = false,
  appearance = "card",
  view = "full",
  onRestoreDefaultPrompt,
  restoreDefaultPromptDisabled = false,
  restoreError,
  onRetryRestore,
  restoreRetryDisabled = false,
  disabled = false,
  beforeSteps,
}: {
  draft: PlanningReviewDraft;
  onChange: (next: PlanningReviewDraft) => void;
  organizationId?: string;
  factoryId?: string;
  factoryKey?: string;
  showResources?: boolean;
  showConcurrency?: boolean;
  showSteps?: boolean;
  modelLabel?: string;
  modelDescription?: string;
  modelOnly?: boolean;
  /** `fields` matches the pull request step. `card` is the GitHub agent settings. */
  appearance?: "card" | "fields";
  view?: "full" | "implementation" | "planning" | "pr-feedback";
  onRestoreDefaultPrompt?: () => void;
  restoreDefaultPromptDisabled?: boolean;
  restoreError?: string;
  onRetryRestore?: () => void;
  restoreRetryDisabled?: boolean;
  disabled?: boolean;
  beforeSteps?: ReactNode;
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
          showResources={showResources}
          showConcurrency={showConcurrency}
          showSteps={showSteps}
          modelLabel={modelLabel}
          modelDescription={modelDescription}
          modelOnly={modelOnly}
          onRestoreDefaultPrompt={onRestoreDefaultPrompt}
          restoreDefaultPromptDisabled={restoreDefaultPromptDisabled}
          restoreError={restoreError}
          onRetryRestore={onRetryRestore}
          restoreRetryDisabled={restoreRetryDisabled}
          disabled={disabled}
          beforeSteps={beforeSteps}
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
  showResources,
  showConcurrency,
  showSteps,
  modelLabel,
  modelDescription,
  modelOnly,
  onRestoreDefaultPrompt,
  restoreDefaultPromptDisabled,
  restoreError,
  onRetryRestore,
  restoreRetryDisabled,
  disabled,
  beforeSteps,
  onChange,
}: {
  appearance: "card" | "fields";
  view: "full" | "implementation" | "planning" | "pr-feedback";
  component: PlanningReviewComponent;
  organizationId?: string;
  factoryId?: string;
  factoryKey?: string;
  showResources: boolean;
  showConcurrency: boolean;
  showSteps: boolean;
  modelLabel?: string;
  modelDescription?: string;
  modelOnly: boolean;
  onRestoreDefaultPrompt?: () => void;
  restoreDefaultPromptDisabled: boolean;
  restoreError?: string;
  onRetryRestore?: () => void;
  restoreRetryDisabled: boolean;
  disabled: boolean;
  beforeSteps?: ReactNode;
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
      modelOnly={modelOnly}
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
            : "grid grid-cols-2 gap-x-6 gap-y-4 rounded-xl border border-border bg-card px-5 py-4 shadow-sm"
        }
        data-testid="planning-review-settings"
      >
        {view !== "full" && modelControl ? (
          <div role="group" aria-label={modelLabel ?? "Default model"}>
            <ImplementSettingRow
              label={modelLabel ?? "Default model"}
              description={
                modelDescription ??
                (view === "planning"
                  ? "The model used to plan tasks."
                  : view === "pr-feedback"
                    ? "The model used to address pull request comments or failing checks."
                    : "The model used to implement tasks.")
              }
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
      </section>
      {view === "full" && showSteps ? beforeSteps : null}
      {view === "full" && showSteps ? (
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
