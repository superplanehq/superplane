import type { RunnersGetFleetCapacityResponse } from "@/api-client";
import { TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

export function FleetStatus({ enabled }: { enabled: boolean }) {
  const label = `This fleet is ${enabled ? "enabled" : "disabled"}.`;
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          type="button"
          aria-label={label}
          className="inline-flex size-5 shrink-0 items-center justify-center rounded-sm focus-visible:outline-ring"
        >
          <span
            aria-hidden="true"
            className={`size-2 rounded-full ${enabled ? "bg-emerald-500" : "bg-gray-400 dark:bg-gray-500"}`}
          />
        </button>
      </TooltipTrigger>
      <TooltipContent>{label}</TooltipContent>
    </Tooltip>
  );
}

const runnerCounts = [
  {
    state: "pending",
    label: "Pending",
    key: "pendingRunners",
    color: "bg-amber-500",
    description: "Waiting to register.",
  },
  {
    state: "idle",
    label: "Idle",
    key: "idleRunners",
    color: "bg-blue-500",
    description: "Ready to accept work.",
  },
  {
    state: "busy",
    label: "Busy",
    key: "busyRunners",
    color: "bg-emerald-500",
    description: "Executing work.",
  },
  {
    state: "terminated",
    label: "Terminated",
    key: "terminatedRunners",
    color: "bg-gray-400 dark:bg-gray-500",
    description: "No longer accepting work.",
  },
] as const;

export function FleetCapacity({ capacity }: { capacity: RunnersGetFleetCapacityResponse }) {
  return (
    <div className="flex items-center gap-2 whitespace-nowrap text-[13px] tabular-nums">
      {runnerCounts.slice(0, 3).map(({ label, key, color, description }, index) => (
        <div key={label} className="flex items-center gap-2">
          {index > 0 && (
            <span aria-hidden="true" className="text-muted-foreground/50">
              /
            </span>
          )}
          <Tooltip>
            <TooltipTrigger asChild>
              <button
                type="button"
                aria-label={`${label} runners: ${Number(capacity[key] ?? 0)}`}
                className="inline-flex items-center gap-1.5 rounded-sm py-1 focus-visible:outline-ring"
              >
                <span aria-hidden="true" className={`size-2 shrink-0 rounded-full ${color}`} />
                {Number(capacity[key] ?? 0)}
              </button>
            </TooltipTrigger>
            <TooltipContent>
              {label} runners: {Number(capacity[key] ?? 0)}. {description}
            </TooltipContent>
          </Tooltip>
        </div>
      ))}
    </div>
  );
}

const runnerTabClassName =
  "h-8 rounded-md px-3 text-muted-foreground data-[state=active]:border-border data-[state=active]:bg-foreground/5 data-[state=active]:font-semibold data-[state=active]:text-foreground data-[state=active]:shadow-none dark:data-[state=active]:bg-foreground/10 dark:data-[state=active]:text-foreground";

export function FleetRunnerTabs({ capacity }: { capacity?: RunnersGetFleetCapacityResponse }) {
  return (
    <TabsList
      aria-label="Runner state"
      className="h-auto max-w-full flex-wrap justify-start gap-1.5 rounded-none bg-transparent p-0 dark:bg-transparent"
    >
      {runnerCounts.map(({ state, label, key, color, description }) => (
        <Tooltip key={state}>
          <TooltipTrigger asChild>
            <span className="inline-flex">
              <TabsTrigger
                value={state}
                className={runnerTabClassName}
                aria-label={`${label} runners: ${capacity ? Number(capacity[key] ?? 0) : "Unavailable"}`}
              >
                <span aria-hidden="true" className={`size-2 shrink-0 rounded-full ${color}`} />
                {label}
                <span className="tabular-nums opacity-70">{capacity ? Number(capacity[key] ?? 0) : "—"}</span>
              </TabsTrigger>
            </span>
          </TooltipTrigger>
          <TooltipContent>{description}</TooltipContent>
        </Tooltip>
      ))}
    </TabsList>
  );
}
