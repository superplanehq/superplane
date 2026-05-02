import type { IntegrationsCapabilityDefinition } from "@/api-client";
import type { CapabilityGroupSection } from "@/lib/capabilities";
import { PreCreateCapabilitySection } from "./PreCreateCapabilitySection";

export interface PreCreateCapabilitySelectionProps {
  integrationCapabilities: IntegrationsCapabilityDefinition[];
  capabilitySections: CapabilityGroupSection[];
  capabilityByName: Map<string, IntegrationsCapabilityDefinition>;
  selectedCapabilities: ReadonlySet<string>;
  onToggleCapability: (capabilityName: string) => void;
  onToggleCapabilityGroup: (capabilityNames: string[]) => void;
  isCreatePending: boolean;
}

export function PreCreateCapabilitySelection({
  integrationCapabilities,
  capabilitySections,
  capabilityByName,
  selectedCapabilities,
  onToggleCapability,
  onToggleCapabilityGroup,
  isCreatePending,
}: PreCreateCapabilitySelectionProps) {
  if (integrationCapabilities.length === 0) {
    return null;
  }

  return (
    <div className="space-y-3">
      <hr className="border-gray-200 dark:border-gray-800" />
      <p className="text-sm text-gray-600 dark:text-gray-400">
        Choose which capabilities to enable for this integration. You need at least one. Use a group row to select or
        clear every capability in that group at once.
      </p>
      <div className="space-y-4">
        {capabilitySections.map((section) => (
          <PreCreateCapabilitySection
            key={section.key}
            section={section}
            capabilityByName={capabilityByName}
            selectedCapabilities={selectedCapabilities}
            onToggleCapability={onToggleCapability}
            onToggleCapabilityGroup={onToggleCapabilityGroup}
            isCreatePending={isCreatePending}
          />
        ))}
      </div>
    </div>
  );
}
