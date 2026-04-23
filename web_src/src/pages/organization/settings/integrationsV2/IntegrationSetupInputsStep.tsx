import type { IntegrationSetupStepDefinition } from "@/api-client";
import { Button } from "@/components/ui/button";
import { ConfigurationFieldRenderer } from "@/ui/configurationFieldRenderer";
import { IntegrationInstructionsV2 } from "@/ui/IntegrationInstructionsV2";

interface IntegrationSetupInputsStepProps {
  organizationId: string;
  step: IntegrationSetupStepDefinition;
  values: Record<string, unknown>;
  validationErrors?: Set<string>;
  onChange: (fieldName: string, value: unknown) => void;
  onSubmit: () => void;
  isSubmitting?: boolean;
}

export function IntegrationSetupInputsStep({
  organizationId,
  step,
  values,
  validationErrors,
  onChange,
  onSubmit,
  isSubmitting,
}: IntegrationSetupInputsStepProps) {
  const fields = (step.inputs || []).filter((field) => Boolean(field.name));
  const hasInstructions = Boolean(step.instructions?.trim());

  return (
    <div className="space-y-4">
      <div>
        <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">{step.label}</h2>
        <IntegrationInstructionsV2 description={step.instructions} className="mt-3" />
      </div>

      {hasInstructions && <div className="border-t border-gray-200 dark:border-gray-700" />}

      <div className="space-y-4">
        {fields.map((field) => {
          const fieldName = field.name!;
          return (
            <ConfigurationFieldRenderer
              key={fieldName}
              field={field}
              value={values[fieldName]}
              onChange={(value) => onChange(fieldName, value)}
              allValues={values}
              domainId={organizationId}
              domainType="DOMAIN_TYPE_ORGANIZATION"
              organizationId={organizationId}
              validationErrors={validationErrors}
            />
          );
        })}
      </div>

      <div className="flex items-center gap-3 pt-2">
        <Button onClick={onSubmit} disabled={Boolean(isSubmitting)}>
          {isSubmitting ? "Saving..." : "Next"}
        </Button>
      </div>
    </div>
  );
}
