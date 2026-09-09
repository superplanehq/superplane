import type { ConfigurationField } from "@/api-client";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ConfigurationFieldRenderer } from "@/ui/configurationFieldRenderer";

import type { PlanningReviewComponent, PlanningReviewDraft, PlanningReviewStep } from "./planningReviewMockup";
import { PLANNING_REVIEW_RUNNER_FIELDS } from "./planningReviewRunnerFields";
import { PlanningReviewStepList } from "./PlanningReviewStepList";

const MODEL_FIELD: ConfigurationField | undefined = PLANNING_REVIEW_RUNNER_FIELDS.find(
  (field) => field.name === "model",
);

const MODEL_USED_FIELD: ConfigurationField | undefined = MODEL_FIELD
  ? { ...MODEL_FIELD, label: "Model used", description: "" }
  : undefined;

const EXPRESSION_CONTEXT = {
  data: { branch: "feature/planning-review" },
  order: { description: "Add a simple planning review editor." },
  previous: { data: { result: { branch: "feature/planning-review" } } },
};

export function PlanningReviewForm({
  draft,
  onChange,
  organizationId,
}: {
  draft: PlanningReviewDraft;
  onChange: (next: PlanningReviewDraft) => void;
  organizationId?: string;
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
          onChange={(next) => updateComponent(component.id, next)}
        />
      ))}
    </div>
  );
}

function AgentPanel({
  component,
  organizationId,
  onChange,
}: {
  component: PlanningReviewComponent;
  organizationId?: string;
  onChange: (next: PlanningReviewComponent) => void;
}) {
  const setConfigurationField = (name: string, value: unknown) => {
    onChange({
      ...component,
      configuration: { ...component.configuration, [name]: value },
    });
  };

  return (
    <div className="flex flex-col gap-4" data-testid={`planning-review-component-${component.id}`}>
      <section
        className="grid grid-cols-2 gap-x-6 gap-y-4 rounded-xl border border-border bg-card px-5 py-4 shadow-sm"
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
        {MODEL_USED_FIELD ? (
          <ConfigurationFieldRenderer
            field={MODEL_USED_FIELD}
            value={component.configuration.model}
            onChange={(value) => setConfigurationField("model", value)}
            allValues={component.configuration}
            organizationId={organizationId}
            allowExpressions
            autocompleteExampleObj={EXPRESSION_CONTEXT}
            fieldPath="model"
          />
        ) : null}
      </section>
      <PlanningReviewStepList
        steps={(component.configuration.steps as PlanningReviewStep[]) ?? []}
        onChange={(steps) => setConfigurationField("steps", steps)}
      />
    </div>
  );
}
