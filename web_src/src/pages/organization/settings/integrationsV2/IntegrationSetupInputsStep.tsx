import type { IntegrationSetupStepDefinition } from "@/api-client";
import { Button } from "@/components/ui/button";
import { ConfigurationFieldRenderer } from "@/ui/configurationFieldRenderer";
import { IntegrationInstructionsV2 } from "@/ui/IntegrationInstructionsV2";
import { ArrowLeft, MoveRight } from "lucide-react";

interface IntegrationSetupInputsStepProps {
  organizationId: string;
  step: IntegrationSetupStepDefinition;
  values: Record<string, unknown>;
  validationErrors?: Set<string>;
  onBack?: () => void;
  onChange: (fieldName: string, value: unknown) => void;
  onSubmit: () => void;
  isReverting?: boolean;
  isSubmitting?: boolean;
}

export function IntegrationSetupInputsStep({
  organizationId,
  step,
  values,
  validationErrors,
  onBack,
  onChange,
  onSubmit,
  isReverting,
  isSubmitting,
}: IntegrationSetupInputsStepProps) {
  const fields = (step.inputs || []).filter((field) => Boolean(field.name));
  const hasInstructions = Boolean(step.instructions?.trim());

  return (
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

      <div className="flex w-fit max-w-full items-center gap-4 pt-2">
        {onBack ? (
          <Button
            type="button"
            variant="link"
            onClick={onBack}
            disabled={Boolean(isSubmitting || isReverting)}
            className="group h-auto shrink-0 gap-1.5 px-0 py-1 font-normal hover:!no-underline"
          >
            <ArrowLeft
              aria-hidden
              className="size-4 shrink-0 transition-transform duration-200 ease-out group-hover:-translate-x-1 motion-reduce:transition-none motion-reduce:group-hover:translate-x-0"
            />
            {isReverting ? "Going back..." : "Previous"}
          </Button>
        ) : null}
        <Button
          onClick={onSubmit}
          disabled={Boolean(isSubmitting || isReverting)}
          className="group justify-center gap-2 text-sm !px-7 hover:!bg-primary"
        >
          {isSubmitting ? "Saving..." : "Next"}
          <MoveRight
            aria-hidden
            className="size-4 shrink-0 transition-transform duration-200 ease-out group-hover:translate-x-1 motion-reduce:transition-none motion-reduce:group-hover:translate-x-0"
          />
        </Button>
      </div>

      {hasInstructions ? <hr className="my-8 border-0 border-t border-gray-300 dark:border-gray-600" /> : null}

      <IntegrationInstructionsV2 description={step.instructions} />
    </div>
  );
}
