import type React from "react";

import { Text } from "@/components/Text/text";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectLabel,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
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
  const fleetOptions = (query.data?.fleets ?? []).flatMap((fleet) =>
    fleet.id ? [{ id: fleet.id, spec: fleet.spec, scope: fleet.scope }] : [],
  );
  const selected = typeof value === "string" ? value : "";
  const { absent, unavailable, status, selectionDisabled } = fleetSelectionState(query, selected);

  const testId = field.name ? toTestId(`field-${field.name}-runner-fleet`) : undefined;
  return (
    <div className="space-y-2">
      <Select
        value={selected}
        onValueChange={(nextValue) => onChange(nextValue || undefined)}
        disabled={readOnly || selectionDisabled}
      >
        <SelectTrigger className="w-full" data-testid={testId}>
          <SelectValue placeholder="Select a runner fleet" />
        </SelectTrigger>
        <SelectContent position="popper" className="max-h-60">
          {absent ? (
            <SelectItem value={selected} disabled>
              {selected}
              {unavailable ? " (Unavailable)" : ""}
            </SelectItem>
          ) : null}
          {(["organization", "installation"] as const).map((scope) => {
            const fleets = fleetOptions.filter((fleet) => fleet.scope === scope);
            return fleets.length ? (
              <SelectGroup key={scope}>
                <SelectLabel>{scope === "organization" ? "Organization fleets" : "Installation fleets"}</SelectLabel>
                {fleets.map((fleet) => (
                  <SelectItem key={fleet.id} value={fleet.id}>
                    {runnerFleetLabel(fleet.id, fleet.spec)}
                  </SelectItem>
                ))}
              </SelectGroup>
            ) : null;
          })}
        </SelectContent>
      </Select>
      {status ? <Text className="text-sm text-muted-foreground">{status}</Text> : null}
    </div>
  );
};

function fleetSelectionState(query: ReturnType<typeof useOrganizationRunnerFleets>, selected: string) {
  const fleets = query.data?.fleets ?? [];
  const absent = Boolean(selected && !fleets.some((fleet) => fleet.id === selected));
  const unavailable = absent && !query.isLoading && !query.isError && !selected.includes("{{");
  const selectionDisabled = query.isLoading || query.isError || fleets.length === 0;
  let status: string | undefined;
  if (query.isLoading) status = "Loading runner fleets...";
  else if (query.isError) status = "SuperPlane could not load runner fleets. Try again.";
  else if (unavailable) status = "Select an available runner fleet before new work can execute.";
  else if (fleets.length === 0) status = "No runner fleets are available.";
  return { absent, unavailable, status, selectionDisabled };
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
