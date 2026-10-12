import type React from "react";

import { Text } from "@/components/Text/text";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useOrganizationRunnerFleets } from "@/hooks/useOrganizationRunnerFleets";
import { toTestId } from "@/lib/testID";
import type { FieldRendererProps } from "./types";

type RunnerFleetSpec = {
  operatingSystem?: string;
  cpuMillicores?: number;
  memoryMb?: number;
};

export const RunnerFleetFieldRenderer: React.FC<FieldRendererProps> = ({
  field,
  value,
  onChange,
  organizationId,
  readOnly = false,
}) => {
  const query = useOrganizationRunnerFleets(organizationId ?? "");

  if (!organizationId) {
    return <div className="text-sm text-red-500 dark:text-red-400">This field requires organization context.</div>;
  }
  if (query.isLoading) {
    return <Text className="text-sm text-gray-500 dark:text-gray-400">Loading machine types...</Text>;
  }
  if (query.isError) {
    return (
      <Text className="text-sm text-gray-500 dark:text-gray-400">
        SuperPlane could not load machine types. Try again.
      </Text>
    );
  }

  const fleetOptions = (query.data?.fleets ?? []).flatMap((fleet) =>
    fleet.id ? [{ id: fleet.id, spec: fleet.spec }] : [],
  );
  if (fleetOptions.length === 0) {
    return <Text className="text-sm text-gray-500 dark:text-gray-400">No machine types are available.</Text>;
  }

  const testId = field.name ? toTestId(`field-${field.name}-runner-fleet`) : undefined;
  return (
    <Select
      value={typeof value === "string" ? value : ""}
      onValueChange={(nextValue) => onChange(nextValue || undefined)}
      disabled={readOnly}
    >
      <SelectTrigger className="w-full" data-testid={testId}>
        <SelectValue placeholder="Select a machine type" />
      </SelectTrigger>
      <SelectContent position="popper" className="max-h-60">
        {fleetOptions.map((fleet) => (
          <SelectItem key={fleet.id} value={fleet.id}>
            {runnerFleetLabel(fleet.id, fleet.spec)}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
};

function runnerFleetLabel(id: string, spec: RunnerFleetSpec | undefined): string {
  if (!spec) {
    return id;
  }

  const details = [
    spec.operatingSystem,
    spec.cpuMillicores ? `${spec.cpuMillicores / 1000} vCPU` : "",
    spec.memoryMb ? `${spec.memoryMb / 1024}GB` : "",
  ].filter(Boolean);
  return details.length > 0 ? `${id} (${details.join(", ")})` : id;
}
