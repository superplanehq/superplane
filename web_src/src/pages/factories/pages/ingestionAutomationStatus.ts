const ACTIVE_RUN_STATES = new Set(["STATE_PENDING", "STATE_STARTED", "STATE_CANCELLING"]);

type ScheduleNode = {
  component?: string;
  configuration?: {
    type?: string;
    minutesInterval?: number;
  };
  metadata?: {
    nextTrigger?: string;
  };
};

type Run = {
  id?: string;
  state?: string;
  createdAt?: string;
};

export function findActiveRun(runs: Run[]): Run | undefined {
  return runs.find((run) => ACTIVE_RUN_STATES.has(run.state ?? ""));
}

export function nextScheduledCycle(
  nodes: ScheduleNode[],
  runs: Run[],
  canvasCreatedAt: string | undefined,
  now: Date,
): Date | null {
  const schedule = nodes.find((node) => node.component === "schedule");
  if (!schedule) return null;

  const backendNextTrigger = parseDate(schedule.metadata?.nextTrigger);
  const interval = schedule.configuration?.type === "minutes" ? schedule.configuration.minutesInterval : undefined;
  if (backendNextTrigger && backendNextTrigger > now) return backendNextTrigger;
  if (!interval || interval < 1) return backendNextTrigger;

  const lastRunAt = runs.map((run) => parseDate(run.createdAt)).find((date) => date !== null);
  const reference = backendNextTrigger ?? lastRunAt ?? parseDate(canvasCreatedAt);
  if (!reference) return null;

  const intervalMs = interval * 60_000;
  const elapsedIntervals = Math.max(0, Math.floor((now.getTime() - reference.getTime()) / intervalMs) + 1);
  return new Date(reference.getTime() + elapsedIntervals * intervalMs);
}

export function formatNextCycle(nextCycle: Date, now: Date): string {
  const minutes = Math.max(1, Math.ceil((nextCycle.getTime() - now.getTime()) / 60_000));
  if (minutes < 60) return `Next scan in ${minutes} min`;

  return `Next scan at ${nextCycle.toLocaleTimeString([], { hour: "numeric", minute: "2-digit" })}`;
}

function parseDate(value: string | undefined): Date | null {
  if (!value) return null;
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? null : date;
}
