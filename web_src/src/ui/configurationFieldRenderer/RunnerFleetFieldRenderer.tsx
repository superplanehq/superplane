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

type RunnerFleetOption = {
  id: string;
  spec?: RunnerFleetSpec;
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
  const fleetOptions = runnerFleetOptions(query);
  const selection = runnerFleetSelection(value, fleetOptions, query);

  if (selection.showOnlyStatus) {
    return <Text className="text-sm text-gray-500 dark:text-gray-400">{selection.status}</Text>;
  }

  return (
    <div className="space-y-2">
      <Select
        value={selection.selected}
        onValueChange={(nextValue) => onChange(nextValue || undefined)}
        disabled={readOnly || selection.disabled}
      >
        <SelectTrigger className="w-full" data-testid={runnerFleetTestId(field.name)}>
          <SelectValue placeholder="Select a runner fleet">{selection.displayValue}</SelectValue>
        </SelectTrigger>
        <SelectContent position="popper" className="max-h-60">
          <MissingFleetOption value={selection.missingFleet} label={selection.displayValue} />
          {fleetOptions.map((fleet) => (
            <SelectItem key={fleet.id} value={fleet.id}>
              {runnerFleetLabel(fleet.id, fleet.spec)}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <FleetStatus status={selection.status} />
    </div>
  );
};

function runnerFleetOptions(query: ReturnType<typeof useOrganizationRunnerFleets>): RunnerFleetOption[] {
  return (query.data?.fleets ?? []).flatMap((fleet) => (fleet.id ? [{ id: fleet.id, spec: fleet.spec }] : []));
}

function runnerFleetSelection(
  value: unknown,
  fleetOptions: RunnerFleetOption[],
  query: ReturnType<typeof useOrganizationRunnerFleets>,
) {
  const selected = typeof value === "string" ? value : "";
  const selectedIsMissing = Boolean(selected) && !fleetOptions.some((fleet) => fleet.id === selected);
  const selectedIsUnavailable = selectedIsMissing && !query.isLoading && !query.isError;
  const displayValue = selectedIsMissing ? `${selected}${selectedIsUnavailable ? " (Unavailable)" : ""}` : undefined;

  return {
    selected,
    displayValue,
    missingFleet: selectedIsMissing ? selected : "",
    status: fleetStatus(query, fleetOptions.length, selectedIsUnavailable),
    showOnlyStatus: !selected && fleetOptions.length === 0,
    disabled: query.isLoading || query.isError || fleetOptions.length === 0,
  };
}

function runnerFleetTestId(fieldName: string | undefined): string | undefined {
  return fieldName ? toTestId(`field-${fieldName}-runner-fleet`) : undefined;
}

function MissingFleetOption({ value, label }: { value: string; label?: string }) {
  if (!value) return null;
  return (
    <SelectItem value={value} disabled>
      {label}
    </SelectItem>
  );
}

function FleetStatus({ status }: { status: string }) {
  if (!status) return null;
  return <Text className="text-sm text-gray-500 dark:text-gray-400">{status}</Text>;
}

function fleetStatus(
  query: ReturnType<typeof useOrganizationRunnerFleets>,
  fleetCount: number,
  selectedIsUnavailable: boolean,
): string {
  if (query.isLoading) return "Loading runner fleets...";
  if (query.isError) return "SuperPlane could not load runner fleets. Try again.";
  if (selectedIsUnavailable) return "Select an available runner fleet before new work can run.";
  if (fleetCount === 0) return "No runner fleets are available.";
  return "";
}

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
