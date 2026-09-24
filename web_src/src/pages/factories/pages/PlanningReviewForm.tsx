import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { useComponent } from "@/hooks/useComponentData";
import { cn } from "@/lib/utils";
import { ConfigurationFieldRenderer } from "@/ui/configurationFieldRenderer";

import type { PlanningReviewComponent, PlanningReviewDraft, PlanningReviewStep } from "./planningReviewMockup";
import { planningReviewModelUsedField } from "./planningReviewRunnerFields";
import { disabledAgentResourceIds } from "./disabledAgentResourceIds";
import { PlanningReviewResourcesCard } from "./PlanningReviewResourcesCard";
import { PlanningReviewStepList } from "./PlanningReviewStepList";

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
}: {
  draft: PlanningReviewDraft;
  onChange: (next: PlanningReviewDraft) => void;
  organizationId?: string;
  factoryId?: string;
  factoryKey?: string;
  showVisualEvidenceSetting?: boolean;
}) {
  const updateComponent = (id: string, next: PlanningReviewComponent) => {
    onChange({
      ...draft,
      components: draft.components.map((component) => (component.id === id ? next : component)),
    });
  };

  return (
    <div className="flex flex-col gap-4">
      {draft.components.map((component) => (
        <AgentPanel
          key={component.id}
          component={component}
          organizationId={organizationId}
          factoryId={factoryId}
          factoryKey={factoryKey}
          showVisualEvidenceSetting={showVisualEvidenceSetting}
          onChange={(next) => updateComponent(component.id, next)}
        />
      ))}
    </div>
  );
}

function AgentPanel({
  component,
  organizationId,
  factoryId,
  factoryKey,
  showVisualEvidenceSetting,
  onChange,
}: {
  component: PlanningReviewComponent;
  organizationId?: string;
  factoryId?: string;
  factoryKey?: string;
  showVisualEvidenceSetting: boolean;
  onChange: (next: PlanningReviewComponent) => void;
}) {
  const setConfigurationField = (name: string, value: unknown) => {
    onChange({
      ...component,
      configuration: { ...component.configuration, [name]: value },
    });
  };
  const { data: action } = useComponent(organizationId ?? "", component.component ?? "");
  const modelUsedField = planningReviewModelUsedField(component.component, action?.configuration);

  return (
    <div className="flex flex-col gap-4" data-testid={`planning-review-component-${component.id}`}>
      <section
        className={cn(
          "grid gap-x-6 gap-y-4 rounded-xl border border-border bg-card px-5 py-4 shadow-sm",
          showVisualEvidenceSetting ? "grid-cols-3" : "grid-cols-2",
        )}
        data-testid="planning-review-settings"
      >
        <div className="flex flex-col gap-2">
          <Label htmlFor={`planning-review-concurrency-max-${component.id}`}>Concurrency</Label>
          <Input
            id={`planning-review-concurrency-max-${component.id}`}
            data-testid={`planning-review-concurrency-max-${component.id}`}
            type="number"
            min={1}
            value={component.concurrency.max}
            onChange={(event) =>
              onChange({
                ...component,
                concurrency: { ...component.concurrency, max: event.target.value },
              })
            }
            className="shadow-none"
          />
        </div>
        {modelUsedField ? (
          <ConfigurationFieldRenderer
            field={modelUsedField}
            value={component.configuration.model}
            onChange={(value) => setConfigurationField("model", value)}
            allValues={component.configuration}
            organizationId={organizationId}
            allowExpressions
            autocompleteExampleObj={EXPRESSION_CONTEXT}
            fieldPath="model"
          />
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
      <PlanningReviewStepList
        steps={(component.configuration.steps as PlanningReviewStep[]) ?? []}
        onChange={(steps) => setConfigurationField("steps", steps)}
      />
      <PlanningReviewResourcesCard
        organizationId={organizationId}
        factoryId={factoryId}
        factoryKey={factoryKey}
        disabledIds={disabledAgentResourceIds(component.configuration)}
        onDisabledIdsChange={(ids) => setConfigurationField("disabledAgentResourceIds", ids)}
      />
    </div>
  );
}
