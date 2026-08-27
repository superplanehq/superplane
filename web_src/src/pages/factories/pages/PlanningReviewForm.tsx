import type { ConfigurationField } from "@/api-client";
import { AutoCompleteInput } from "@/components/AutoCompleteInput/AutoCompleteInput";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";
import { ConfigurationFieldRenderer } from "@/ui/configurationFieldRenderer";
import { ChevronDown } from "lucide-react";
import { useState } from "react";

import type { PlanningReviewComponent, PlanningReviewDraft, PlanningReviewStep } from "./planningReviewMockup";
import {
  PLANNING_REVIEW_ADVANCED_FIELD_NAMES,
  PLANNING_REVIEW_ENVIRONMENT_MODEL_FIELDS,
  PLANNING_REVIEW_RUNNER_FIELDS,
} from "./planningReviewRunnerFields";
import { PlanningReviewStepList } from "./PlanningReviewStepList";

const FIELDS_BY_NAME = new Map(
  PLANNING_REVIEW_RUNNER_FIELDS.filter((field) => field.name).map((field) => [field.name!, field]),
);

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
    <div className="flex flex-col gap-3">
      {draft.components.map((component) => (
        <ComponentBlock
          key={component.id}
          component={component}
          organizationId={organizationId}
          onChange={(next) => updateComponent(component.id, next)}
        />
      ))}
    </div>
  );
}

function ComponentBlock({
  component,
  organizationId,
  onChange,
}: {
  component: PlanningReviewComponent;
  organizationId?: string;
  onChange: (next: PlanningReviewComponent) => void;
}) {
  return (
    <section
      className="overflow-hidden rounded-xl border border-border bg-card shadow-sm"
      data-testid={`planning-review-component-${component.id}`}
    >
      <button
        type="button"
        className={cn(
          "flex w-full items-center gap-3 px-4 py-3 text-left transition-colors hover:bg-muted",
          component.expanded && "bg-muted",
        )}
        aria-expanded={component.expanded}
        aria-controls={`planning-review-component-body-${component.id}`}
        data-testid={`planning-review-component-toggle-${component.id}`}
        onClick={() => onChange({ ...component, expanded: !component.expanded })}
      >
        <div className="min-w-0 flex-1">
          <h3 className="truncate text-[14px] font-semibold text-foreground">{component.title}</h3>
          <p className="mt-0.5 truncate text-[12px] text-muted-foreground">{component.description}</p>
        </div>
        <ChevronDown
          className={cn(
            "size-4 shrink-0 text-muted-foreground transition-transform",
            component.expanded && "rotate-180",
          )}
          aria-hidden
        />
      </button>
      {component.expanded ? (
        <div
          id={`planning-review-component-body-${component.id}`}
          className="space-y-6 border-t border-border bg-card px-4 py-5"
        >
          <RunnerConfiguration component={component} organizationId={organizationId} onChange={onChange} />
        </div>
      ) : null}
    </section>
  );
}

function RunnerConfiguration({
  component,
  organizationId,
  onChange,
}: {
  component: PlanningReviewComponent;
  organizationId?: string;
  onChange: (next: PlanningReviewComponent) => void;
}) {
  const [moreSettingsOpen, setMoreSettingsOpen] = useState(false);
  const setConfigurationField = (name: string, value: unknown) => {
    onChange({
      ...component,
      configuration: { ...component.configuration, [name]: value },
    });
  };

  return (
    <>
      <PlanningReviewStepList
        steps={(component.configuration.steps as PlanningReviewStep[]) ?? []}
        onChange={(steps) => setConfigurationField("steps", steps)}
      />
      <div className="grid grid-cols-2 gap-x-4 gap-y-3" data-testid="planning-review-environment-model-row">
        {fieldsNamed(PLANNING_REVIEW_ENVIRONMENT_MODEL_FIELDS).map((field) => (
          <RunnerField
            key={field.name}
            field={field}
            component={component}
            organizationId={organizationId}
            onChange={setConfigurationField}
          />
        ))}
      </div>
      <div className="border-t border-border pt-4">
        <button
          type="button"
          className="flex w-full items-center justify-between py-1 text-left text-[13px] font-medium text-foreground hover:text-foreground/80"
          aria-expanded={moreSettingsOpen}
          data-testid="planning-review-more-settings-toggle"
          onClick={() => setMoreSettingsOpen((open) => !open)}
        >
          More settings
          <ChevronDown
            className={cn(
              "size-4 shrink-0 text-muted-foreground transition-transform",
              moreSettingsOpen && "rotate-180",
            )}
            aria-hidden
          />
        </button>
        {moreSettingsOpen ? (
          <div className="mt-4 space-y-6">
            {fieldsNamed(PLANNING_REVIEW_ADVANCED_FIELD_NAMES).map((field) => (
              <RunnerField
                key={field.name}
                field={field}
                component={component}
                organizationId={organizationId}
                onChange={setConfigurationField}
              />
            ))}
            <ConcurrencyFields component={component} onChange={onChange} />
          </div>
        ) : null}
      </div>
    </>
  );
}

function fieldsNamed(names: string[]): ConfigurationField[] {
  return names.flatMap((name) => {
    const field = FIELDS_BY_NAME.get(name);
    return field ? [field] : [];
  });
}

function RunnerField({
  field,
  component,
  organizationId,
  onChange,
}: {
  field: ConfigurationField;
  component: PlanningReviewComponent;
  organizationId?: string;
  onChange: (name: string, value: unknown) => void;
}) {
  const name = field.name;
  if (!name) {
    return null;
  }
  return (
    <ConfigurationFieldRenderer
      field={field}
      value={component.configuration[name]}
      onChange={(value) => onChange(name, value)}
      allValues={component.configuration}
      organizationId={organizationId}
      allowExpressions
      autocompleteExampleObj={EXPRESSION_CONTEXT}
      fieldPath={name}
    />
  );
}

function ConcurrencyFields({
  component,
  onChange,
}: {
  component: PlanningReviewComponent;
  onChange: (next: PlanningReviewComponent) => void;
}) {
  return (
    <div className="space-y-4 border-t border-border pt-6">
      <h4 className="text-sm font-semibold text-foreground">Concurrency</h4>
      <div className="flex flex-col gap-2">
        <Label htmlFor={`planning-review-concurrency-max-${component.id}`}>Max parallel executions</Label>
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
        <p className="text-xs text-muted-foreground">
          Executions above this limit wait in the queue. Default: one execution at a time.
        </p>
      </div>
      <div className="flex flex-col gap-2">
        <Label>Key</Label>
        <AutoCompleteInput
          data-testid={`planning-review-concurrency-key-${component.id}`}
          exampleObj={EXPRESSION_CONTEXT}
          value={component.concurrency.key}
          onChange={(key) =>
            onChange({
              ...component,
              concurrency: { ...component.concurrency, key },
            })
          }
          placeholder="ci-{{ $.data.branch }}"
          startWord="{{"
          prefix="{{ "
          suffix=" }}"
          inputSize="md"
          quickTip="Tip: type `{{` to start an expression."
          className="shadow-none"
        />
        <p className="text-xs text-muted-foreground">
          Optional expression that splits the backlog. Each value is a separate queue.
        </p>
      </div>
    </div>
  );
}
