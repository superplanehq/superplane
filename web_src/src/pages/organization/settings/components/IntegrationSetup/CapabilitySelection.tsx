import type { IntegrationsCapabilityDefinition } from "@/api-client";
import type { CapabilityGroupSection } from "@/lib/capabilities";
import { CapabilitySection } from "./CapabilitySection";

export interface CapabilitySelectionProps {
  integrationCapabilities: IntegrationsCapabilityDefinition[];
  capabilitySections: CapabilityGroupSection[];
  capabilityByName: Map<string, IntegrationsCapabilityDefinition>;
  selectedCapabilities: ReadonlySet<string>;
  onToggleCapability: (capabilityName: string) => void;
  onToggleCapabilityGroup: (capabilityNames: string[]) => void;
  selectionDisabled: boolean;
}

export function CapabilitySelection({
  integrationCapabilities,
  capabilitySections,
  capabilityByName,
  selectedCapabilities,
  onToggleCapability,
  onToggleCapabilityGroup,
  selectionDisabled,
}: CapabilitySelectionProps) {
  if (integrationCapabilities.length === 0) {
    return null;
  }

  return (
    <div className="space-y-3">
      <hr className="border-edge-subtle" />
      <p className="text-sm text-content-secondary">
        Choose which capabilities to enable for this integration. You need at least one. Use a group row to select or
        clear every capability in that group at once.
      </p>
      <div className="space-y-4">
        {capabilitySections.map((section) => (
          <CapabilitySection
            key={section.key}
            section={section}
            capabilityByName={capabilityByName}
            selectedCapabilities={selectedCapabilities}
            onToggleCapability={onToggleCapability}
            onToggleCapabilityGroup={onToggleCapabilityGroup}
            selectionDisabled={selectionDisabled}
          />
        ))}
      </div>
    </div>
  );
}
