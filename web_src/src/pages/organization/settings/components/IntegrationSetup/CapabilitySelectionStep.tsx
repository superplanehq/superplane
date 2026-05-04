import type {
  IntegrationSetupStepDefinition,
  IntegrationsCapabilityDefinition,
  IntegrationsIntegrationDefinition,
} from "@/api-client";
import { Button } from "@/components/ui/button";
import {
  buildIntegrationCapabilityGroupSections,
  capabilityDefinitionsForStepOffer,
  type CapabilityGroupSection,
} from "@/lib/capabilities";
import { useMemo } from "react";
import { Instructions } from "./Instructions";
import { CapabilitySelection } from "./CapabilitySelection";
import { ArrowLeft, MoveRight } from "lucide-react";

function capabilityByNameFromDefinitions(defs: IntegrationsCapabilityDefinition[]) {
  const map = new Map<string, IntegrationsCapabilityDefinition>();
  for (const capability of defs) {
    if (capability.name) {
      map.set(capability.name, capability);
    }
  }
  return map;
}

interface CapabilitySelectionStepProps {
  step: IntegrationSetupStepDefinition;
  integrationDefinition: IntegrationsIntegrationDefinition | undefined;
  integrationCapabilities: IntegrationsCapabilityDefinition[];
  selectedCapabilities: ReadonlySet<string>;
  onBack?: () => void;
  onToggleCapability: (capabilityName: string) => void;
  onToggleCapabilityGroup: (capabilityNames: string[]) => void;
  onSubmit: () => void;
  isReverting?: boolean;
  isSubmitting?: boolean;
}

export function CapabilitySelectionStep({
  step,
  integrationDefinition,
  integrationCapabilities,
  selectedCapabilities,
  onBack,
  onToggleCapability,
  onToggleCapabilityGroup,
  onSubmit,
  isReverting,
  isSubmitting,
}: CapabilitySelectionStepProps) {
  const offeredDefinitions = useMemo(
    () => capabilityDefinitionsForStepOffer(integrationCapabilities, step.capabilities),
    [integrationCapabilities, step.capabilities],
  );

  const capabilitySections: CapabilityGroupSection[] = useMemo(
    () => buildIntegrationCapabilityGroupSections(integrationDefinition, offeredDefinitions),
    [integrationDefinition, offeredDefinitions],
  );

  const capabilityByName = useMemo(() => capabilityByNameFromDefinitions(offeredDefinitions), [offeredDefinitions]);

  const selectionDisabled = Boolean(isSubmitting || isReverting);
  const hasInstructions = Boolean(step.instructions?.trim());
  const mustPickOne = offeredDefinitions.length > 0;

  return (
    <div className="space-y-4">
      <CapabilitySelection
        integrationCapabilities={offeredDefinitions}
        capabilitySections={capabilitySections}
        capabilityByName={capabilityByName}
        selectedCapabilities={selectedCapabilities}
        onToggleCapability={onToggleCapability}
        onToggleCapabilityGroup={onToggleCapabilityGroup}
        selectionDisabled={selectionDisabled}
      />

      <div className="flex w-fit max-w-full items-center gap-4 pt-2">
        {onBack ? (
          <Button
            type="button"
            variant="link"
            onClick={onBack}
            disabled={selectionDisabled}
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
          type="button"
          onClick={() => void onSubmit()}
          disabled={selectionDisabled || (mustPickOne && selectedCapabilities.size === 0)}
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

      <Instructions description={step.instructions} />
    </div>
  );
}
