import React from "react";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useRunnerFleets, type RunnerFleet } from "@/hooks/useRunnerFleets";
import type { FieldRendererProps } from "./types";
import { toTestId } from "@/lib/testID";

const UNAVAILABLE_PREFIX = "__unavailable__:";

const RUNNER_TYPE_LABELS: Record<string, string> = {
  small: "Small",
  medium: "Standard",
  standard: "Standard",
  large: "Large",
  xlarge: "Extra large",
  "2xlarge": "2xlarge",
};

const RUNNER_TYPE_ORDER: Record<string, number> = {
  Small: 1,
  Standard: 2,
  Large: 3,
  "Extra large": 4,
  "2xlarge": 5,
};

export function formatRunnerFleetLabel(fleet: RunnerFleet): string {
  const runnerType = runnerTypeFromFleet(fleet);
  const metadata = [`${runnerType} runner`, fleet.arch].filter(
    (part): part is string => typeof part === "string" && part.trim() !== "",
  );

  return metadata.join(" · ");
}

function runnerTypeFromFleet(fleet: RunnerFleet): string {
  const tokens = `${fleet.id} ${fleet.size ?? ""}`
    .toLowerCase()
    .split(/[^a-z0-9]+/)
    .filter(Boolean);

  const orderedTokens = ["2xlarge", "xlarge", "large", "standard", "medium", "small"];
  const match = orderedTokens.find((token) => tokens.includes(token));
  if (match) {
    return RUNNER_TYPE_LABELS[match];
  }

  return "Standard";
}

export const RunnerFleetFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange, organizationId }) => {
  const selectedFleetID = typeof value === "string" ? value : "";
  const { data, isLoading, error } = useRunnerFleets(organizationId);
  const fleets = data?.fleets ?? [];
  const selectedFleet = fleets.find((fleet) => fleet.id === selectedFleetID);
  const selectedFleetUnavailable = selectedFleetID !== "" && !selectedFleet;
  const disabled = isLoading || !!error || data?.configured === false || fleets.length === 0;
  const architectures = React.useMemo(() => architectureOptions(fleets), [fleets]);
  const [selectedArch, setSelectedArch] = React.useState<string>("");

  React.useEffect(() => {
    if (selectedFleet?.arch) {
      setSelectedArch(selectedFleet.arch);
    }
  }, [selectedFleet?.arch]);

  const fleetsForSelectedArch = React.useMemo(
    () => fleets.filter((fleet) => normalizeArch(fleet.arch) === normalizeArch(selectedArch)).sort(compareRunnerFleets),
    [fleets, selectedArch],
  );

  if (error) {
    return (
      <div className="text-sm text-red-500 dark:text-red-400">
        {error instanceof Error ? error.message : "Failed to load runner fleets"}
      </div>
    );
  }

  return (
    <div className="space-y-2">
      <Select
        value={selectedArch}
        onValueChange={(nextArch) => {
          setSelectedArch(nextArch);
          if (selectedFleetID) {
            onChange(undefined);
          }
        }}
        disabled={disabled}
      >
        <SelectTrigger
          className="w-full"
          data-testid={field.name ? toTestId(`field-${field.name}-architecture-select`) : undefined}
        >
          <SelectValue placeholder={runnerArchitecturePlaceholder(isLoading, data)} />
        </SelectTrigger>
        <SelectContent className="max-h-60">
          {architectures.map((arch) => (
            <SelectItem key={arch} value={arch}>
              {formatArchitectureLabel(arch)}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      <Select
        value={selectedFleetUnavailable ? `${UNAVAILABLE_PREFIX}${selectedFleetID}` : selectedFleetID}
        onValueChange={(nextValue) => {
          if (nextValue.startsWith(UNAVAILABLE_PREFIX)) {
            return;
          }
          onChange(nextValue || undefined);
        }}
        disabled={disabled || !selectedArch || fleetsForSelectedArch.length === 0}
      >
        <SelectTrigger
          className="w-full"
          data-testid={field.name ? toTestId(`field-${field.name}-size-select`) : undefined}
        >
          <SelectValue placeholder={runnerSizePlaceholder(selectedArch, fleetsForSelectedArch)} />
        </SelectTrigger>
        <SelectContent className="max-h-60">
          {selectedFleetUnavailable ? (
            <SelectItem value={`${UNAVAILABLE_PREFIX}${selectedFleetID}`} disabled>
              Unavailable runner
            </SelectItem>
          ) : null}
          {fleetsForSelectedArch.map((fleet) => (
            <SelectItem key={fleet.id} value={fleet.id} title={fleet.id}>
              {runnerTypeFromFleet(fleet)} runner
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      {selectedFleetUnavailable ? (
        <p className="text-xs text-amber-600 dark:text-amber-400">
          Selected fleet is no longer registered in the task broker.
        </p>
      ) : null}
      {data?.configured === false ? (
        <p className="text-xs text-gray-500 dark:text-gray-400">Runner task broker is not configured.</p>
      ) : null}
      {data?.configured === true && fleets.length === 0 ? (
        <p className="text-xs text-gray-500 dark:text-gray-400">No runner fleets are registered yet.</p>
      ) : null}
    </div>
  );
};

function architectureOptions(fleets: RunnerFleet[]): string[] {
  return Array.from(new Set(fleets.map((fleet) => fleet.arch).filter((arch): arch is string => !!arch))).sort(
    compareArchitectures,
  );
}

function compareArchitectures(a: string, b: string): number {
  const order = ["amd64", "arm64"];
  const aIndex = order.indexOf(normalizeArch(a));
  const bIndex = order.indexOf(normalizeArch(b));
  if (aIndex !== -1 || bIndex !== -1) {
    return (aIndex === -1 ? Number.MAX_SAFE_INTEGER : aIndex) - (bIndex === -1 ? Number.MAX_SAFE_INTEGER : bIndex);
  }

  return a.localeCompare(b);
}

function compareRunnerFleets(a: RunnerFleet, b: RunnerFleet): number {
  const aType = runnerTypeFromFleet(a);
  const bType = runnerTypeFromFleet(b);
  const byType =
    (RUNNER_TYPE_ORDER[aType] ?? Number.MAX_SAFE_INTEGER) - (RUNNER_TYPE_ORDER[bType] ?? Number.MAX_SAFE_INTEGER);
  if (byType !== 0) {
    return byType;
  }

  return a.id.localeCompare(b.id);
}

function formatArchitectureLabel(arch: string): string {
  switch (normalizeArch(arch)) {
    case "amd64":
      return "AMD64";
    case "arm64":
      return "ARM64";
    default:
      return arch;
  }
}

function normalizeArch(arch: string | undefined): string {
  return arch?.trim().toLowerCase() ?? "";
}

function runnerArchitecturePlaceholder(isLoading: boolean, data: ReturnType<typeof useRunnerFleets>["data"]): string {
  if (isLoading) {
    return "Loading fleets...";
  }
  if (data?.configured === false) {
    return "Runner broker not configured";
  }
  if (data?.configured === true && data.fleets.length === 0) {
    return "No fleets registered";
  }
  return "Select architecture";
}

function runnerSizePlaceholder(selectedArch: string, fleets: RunnerFleet[]): string {
  if (!selectedArch) {
    return "Select architecture first";
  }
  if (fleets.length === 0) {
    return "No sizes for this architecture";
  }
  return "Select runner size";
}
