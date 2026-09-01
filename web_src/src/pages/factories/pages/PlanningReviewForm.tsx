import type { ConfigurationField } from "@/api-client";
import { AutoCompleteInput } from "@/components/AutoCompleteInput/AutoCompleteInput";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ConfigurationFieldRenderer } from "@/ui/configurationFieldRenderer";

import type { PlanningReviewComponent, PlanningReviewDraft, PlanningReviewStep } from "./planningReviewMockup";
import { PLANNING_REVIEW_RUNNER_FIELDS } from "./planningReviewRunnerFields";
import { PLANNING_REVIEW_SECTIONS, type PlanningReviewSectionId } from "./planningReviewSections";
import { PlanningReviewStepList } from "./PlanningReviewStepList";

const FIELDS_BY_NAME = new Map(
  PLANNING_REVIEW_RUNNER_FIELDS.filter((field) => field.name).map((field) => [field.name!, field]),
);

const EXPRESSION_CONTEXT = {
  data: { branch: "feature/planning-review" },
  order: { description: "Add a simple planning review editor." },
  previous: { data: { result: { branch: "feature/planning-review" } } },
};

/** Fields that read better side by side than stacked full width. */
const PAIRED_FIELDS = ["machineType", "model", "workingDirectory", "executionTimeoutSeconds"];

export function PlanningReviewForm({
  draft,
  onChange,
  organizationId,
  section,
}: {
  draft: PlanningReviewDraft;
  onChange: (next: PlanningReviewDraft) => void;
  organizationId?: string;
  section: PlanningReviewSectionId;
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
        <SectionPanel
          key={component.id}
          component={component}
          section={section}
          organizationId={organizationId}
          onChange={(next) => updateComponent(component.id, next)}
        />
      ))}
    </div>
  );
}

/**
 * One group of settings. The left column already names the group, so the panel
 * leads with what the group is for instead of repeating the label.
 */
function SectionPanel({
  component,
  section,
  organizationId,
  onChange,
}: {
  component: PlanningReviewComponent;
  section: PlanningReviewSectionId;
  organizationId?: string;
  onChange: (next: PlanningReviewComponent) => void;
}) {
  const definition = PLANNING_REVIEW_SECTIONS.find((entry) => entry.id === section);

  const setConfigurationField = (name: string, value: unknown) => {
    onChange({
      ...component,
      configuration: { ...component.configuration, [name]: value },
    });
  };

  return (
    <div
      className="flex flex-col gap-4"
      data-testid={`planning-review-component-${component.id}`}
      data-section={section}
    >
      {definition ? (
        <p className="px-1 text-[13px] leading-6 text-muted-foreground" data-testid="planning-review-section-intro">
          {definition.description}
        </p>
      ) : null}
      {section === "steps" ? (
        <PlanningReviewStepList
          steps={(component.configuration.steps as PlanningReviewStep[]) ?? []}
          onChange={(steps) => setConfigurationField("steps", steps)}
        />
      ) : null}
      {section === "concurrency" ? <ConcurrencyFields component={component} onChange={onChange} /> : null}
      {definition && definition.fieldNames.length > 0 ? (
        <FieldCard>
          <FieldGrid
            fieldNames={definition.fieldNames}
            component={component}
            organizationId={organizationId}
            onChange={setConfigurationField}
          />
        </FieldCard>
      ) : null}
    </div>
  );
}

function FieldCard({ children }: { children: React.ReactNode }) {
  return <section className="rounded-xl border border-border bg-card px-5 py-5 shadow-sm">{children}</section>;
}

function FieldGrid({
  fieldNames,
  component,
  organizationId,
  onChange,
}: {
  fieldNames: string[];
  component: PlanningReviewComponent;
  organizationId?: string;
  onChange: (name: string, value: unknown) => void;
}) {
  return (
    <div className="grid grid-cols-2 gap-x-6 gap-y-5" data-testid="planning-review-environment-model-row">
      {fieldsNamed(fieldNames).map((field) => (
        <div key={field.name} className={PAIRED_FIELDS.includes(field.name!) ? undefined : "col-span-2"}>
          <RunnerField field={field} component={component} organizationId={organizationId} onChange={onChange} />
        </div>
      ))}
    </div>
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
    <FieldCard>
      <div className="space-y-6">
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
    </FieldCard>
  );
}
