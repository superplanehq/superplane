import { Text } from "@/components/Text/text";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

import {
  fleetOptionLabel,
  hasRunnerShortage,
  readyRunnerCount,
  type FleetCapacity,
  type InstallationFleet,
} from "./fleetAdmin";

export const FleetMachineType = ({
  fleets,
  selectedId,
  onChange,
}: {
  fleets: InstallationFleet[];
  selectedId?: string;
  onChange: (fleetId: string) => void;
}) => (
  <div>
    <Label htmlFor="admin-fleet-machine-type" className="mb-1.5 block text-xs text-gray-500 dark:text-gray-400">
      Machine type
    </Label>
    <Select value={selectedId} onValueChange={onChange}>
      <SelectTrigger
        id="admin-fleet-machine-type"
        aria-label="Machine type"
        data-testid="admin-fleet-machine-type"
        className="h-9 w-72"
      >
        <SelectValue placeholder="Machine type" />
      </SelectTrigger>
      <SelectContent>
        {fleets.map((fleet) => (
          <SelectItem key={fleet.id} value={fleet.id}>
            {fleetOptionLabel(fleet)}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  </div>
);

const SummaryCount = ({ label, value }: { label: string; value: number }) => (
  <div className="rounded-md bg-white px-4 py-3 shadow-sm outline outline-slate-950/10 dark:bg-gray-900 dark:outline-gray-700/70">
    <Text className="text-xs text-gray-500 dark:text-gray-400">{label}</Text>
    <p className="mt-1 text-lg font-semibold text-gray-900 dark:text-gray-100">{value}</p>
  </div>
);

export const FleetCapacitySummary = ({ capacity }: { capacity: FleetCapacity }) => (
  <div className="space-y-3" data-testid="fleet-capacity-summary">
    <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-5">
      <SummaryCount label="Queued tasks" value={capacity.queuedTasks} />
      <SummaryCount label="Ready runners" value={readyRunnerCount(capacity)} />
      <SummaryCount label="Pending runners" value={capacity.pendingRunners} />
      <SummaryCount label="Idle runners" value={capacity.idleRunners} />
      <SummaryCount label="Busy runners" value={capacity.busyRunners} />
    </div>
    {hasRunnerShortage(capacity) ? (
      <div
        className="rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-900 dark:border-amber-900/60 dark:bg-amber-950/40 dark:text-amber-200"
        data-testid="fleet-capacity-shortage"
      >
        More queued tasks than ready runners.
      </div>
    ) : null}
  </div>
);
