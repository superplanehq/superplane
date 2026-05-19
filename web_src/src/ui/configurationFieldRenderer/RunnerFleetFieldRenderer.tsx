import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useRunnerFleets } from "@/hooks/useRunnerFleets";
import type { ConfigurationField } from "../../api-client";

interface RunnerFleetFieldRendererProps {
  field: ConfigurationField;
  value: string;
  onChange: (value: string | undefined) => void;
  organizationId?: string;
}

export function RunnerFleetFieldRenderer({ value, onChange, organizationId }: RunnerFleetFieldRendererProps) {
  const { data: fleets, isLoading, error } = useRunnerFleets(organizationId);

  if (!organizationId?.trim()) {
    return <div className="text-sm text-red-500 dark:text-red-400">Machine type requires an organization context</div>;
  }

  if (error) {
    return (
      <div className="text-sm text-red-500 dark:text-red-400">
        Failed to load machine types: {error instanceof Error ? error.message : "Unknown error"}
      </div>
    );
  }

  if (isLoading) {
    return <div className="text-sm text-gray-500 dark:text-gray-400">Loading machine types...</div>;
  }

  if (!fleets?.length) {
    return (
      <div className="space-y-2">
        <Select value="" disabled>
          <SelectTrigger className="w-full">
            <SelectValue placeholder="No machine types available" />
          </SelectTrigger>
        </Select>
        <p className="text-xs text-gray-500 dark:text-gray-400">
          Ask your installation administrator to register a runner fleet.
        </p>
      </div>
    );
  }

  return (
    <Select value={value ?? ""} onValueChange={(val) => onChange(val || undefined)}>
      <SelectTrigger className="w-full">
        <SelectValue placeholder="Select machine type" />
      </SelectTrigger>
      <SelectContent>
        {fleets.map((fleet) => (
          <SelectItem key={fleet.id} value={fleet.id}>
            {fleet.name}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
