import type { IntegrationsCapabilityDefinition } from "@/api-client";
import { Info, MoveRight } from "lucide-react";
import type { CapabilityGroupSection } from "@/lib/capabilities";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { PreCreateCapabilitySelection } from "./PreCreateCapabilitySelection";

export interface PreCreateIntegrationSetupProps {
  instanceName: string;
  onInstanceNameChange: (value: string) => void;
  integrationName: string;
  integrationCapabilities: IntegrationsCapabilityDefinition[];
  capabilitySections: CapabilityGroupSection[];
  capabilityByName: Map<string, IntegrationsCapabilityDefinition>;
  selectedCapabilities: ReadonlySet<string>;
  onToggleCapability: (capabilityName: string) => void;
  onToggleCapabilityGroup: (capabilityNames: string[]) => void;
  isCreatePending: boolean;
  onCreate: () => void;
}

export function PreCreateIntegrationSetup({
  instanceName,
  onInstanceNameChange,
  integrationName,
  integrationCapabilities,
  capabilitySections,
  capabilityByName,
  selectedCapabilities,
  onToggleCapability,
  onToggleCapabilityGroup,
  isCreatePending,
  onCreate,
}: PreCreateIntegrationSetupProps) {
  return (
    <div className="space-y-6">
      <div className="space-y-3">
        <div className="flex flex-wrap items-center gap-x-3 gap-y-2">
          <Label htmlFor="integration-instance-name" className="mb-0 shrink-0">
            Name
          </Label>
          <Input
            id="integration-instance-name"
            value={instanceName}
            onChange={(event) => onInstanceNameChange(event.target.value)}
            placeholder={`${integrationName}-integration`}
            autoComplete="off"
            className="h-9 w-72 max-w-full"
          />
        </div>
        <div className="flex gap-3 rounded-md border border-gray-300 bg-gray-50 p-3 text-sm leading-relaxed text-gray-600 dark:border-gray-700 dark:bg-gray-900/60 dark:text-gray-400">
          <Info className="mt-0.5 size-4 shrink-0 text-gray-500 dark:text-gray-500" aria-hidden />
          <p className="min-w-0">
            You can connect the same integration type more than once—for different environments, namespaces, or
            organizations. Use a name that identifies this connection.
          </p>
        </div>
      </div>

      <PreCreateCapabilitySelection
        integrationCapabilities={integrationCapabilities}
        capabilitySections={capabilitySections}
        capabilityByName={capabilityByName}
        selectedCapabilities={selectedCapabilities}
        onToggleCapability={onToggleCapability}
        onToggleCapabilityGroup={onToggleCapabilityGroup}
        isCreatePending={isCreatePending}
      />

      <div className="flex w-fit max-w-full items-center gap-4 pt-2">
        <Button
          type="button"
          onClick={() => void onCreate()}
          disabled={
            isCreatePending ||
            !instanceName.trim() ||
            (integrationCapabilities.length > 0 && selectedCapabilities.size === 0)
          }
          className="group justify-center gap-2 text-sm !px-7 hover:!bg-primary"
        >
          {isCreatePending ? "Creating..." : "Next"}
          <MoveRight
            aria-hidden
            className="size-4 shrink-0 transition-transform duration-200 ease-out group-hover:translate-x-1 motion-reduce:transition-none motion-reduce:group-hover:translate-x-0"
          />
        </Button>
      </div>
    </div>
  );
}
