export type InstallationFleet = {
  id: string;
  enabled: boolean;
};

export type FleetCapacity = {
  queuedTasks: number;
  pendingRunners: number;
  idleRunners: number;
  busyRunners: number;
};

export type FleetRunner = {
  id: string;
  state: string;
  runnerVersion?: string;
  os?: string;
  arch?: string;
  hostname?: string;
  ip?: string;
  tags?: Record<string, string>;
  ephemeral?: boolean;
  createdAt?: string | null;
  lastSeenAt?: string | null;
};

export type FleetTask = {
  id: string;
  organizationId?: string;
  state: string;
  runnerId?: string | null;
  queuedAt?: string | null;
  startedAt?: string | null;
  finishedAt?: string | null;
};

export type RecordPage<T> = {
  rows: T[];
  totalCount: number;
  hasNextPage: boolean;
};

export type BrokerTask = {
  id: string;
  status: string;
  fleet_id: string;
  created_at: string;
  claimed_at?: string;
  lease_until?: string;
  runner_id?: string;
  execution_mode?: string;
  docker_image?: string;
  cancel_requested?: boolean;
};

export const REFRESH_INTERVAL_MS = 5000;
export const FLEET_REQUEST_TIMEOUT_MS = 30_000;
export const PAGE_SIZE = 50;
export const RUNNER_STATES = ["idle", "busy"];
export const TASK_STATES = ["queued", "reserved", "running"];

export const statusBadgeClass = (status: string, cancelRequested: boolean) => {
  if (cancelRequested) {
    return "bg-amber-100 text-amber-800 dark:bg-amber-950/40 dark:text-amber-300";
  }
  if (status === "queued") {
    return "bg-slate-100 text-slate-700 dark:bg-gray-800 dark:text-gray-300";
  }
  if (status === "idle" || status === "claimed" || status === "reserved") {
    return "bg-blue-100 text-blue-800 dark:bg-blue-950/40 dark:text-blue-300";
  }
  if (status === "busy" || status === "running") {
    return "bg-emerald-100 text-emerald-800 dark:bg-emerald-950/40 dark:text-emerald-300";
  }
  return "bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300";
};

export const formatBrokerStatus = (status: string, cancelRequested: boolean) => {
  if (cancelRequested) {
    return "cancel requested";
  }
  return status;
};

export const formatExecutionMode = (task: BrokerTask) => {
  const mode = task.execution_mode?.trim() || "host";
  if (mode === "docker" && task.docker_image) {
    return `docker (${task.docker_image})`;
  }
  return mode;
};

export const parseCount = (value: unknown) => {
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }
  if (typeof value !== "string" || value.trim() === "") {
    return 0;
  }
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : 0;
};

export const fleetOptionLabel = (fleet: InstallationFleet) => (fleet.enabled ? fleet.id : `${fleet.id} (disabled)`);

export const listQuery = (states: string[], afterId: string | null) => {
  const params = new URLSearchParams();
  params.set("limit", String(PAGE_SIZE));
  for (const state of states) {
    params.append("states", state);
  }
  if (afterId) {
    params.set("afterId", afterId);
  }
  return params.toString();
};

export const fleetURL = (fleetId: string, suffix: string) =>
  `/admin/api/installation/fleets/${encodeURIComponent(fleetId)}${suffix}`;

export const visibleRange = (pageIndex: number, rowCount: number) => {
  if (rowCount === 0) {
    return { start: 0, end: 0 };
  }
  const start = pageIndex * PAGE_SIZE + 1;
  return { start, end: start + rowCount - 1 };
};

export const readyRunnerCount = (capacity: FleetCapacity) => capacity.idleRunners + capacity.busyRunners;

export const hasRunnerShortage = (capacity: FleetCapacity) => capacity.queuedTasks > readyRunnerCount(capacity);

export const nextCursor = (current: string[], lastId: string | undefined, hasNextPage: boolean) => {
  if (!hasNextPage || !lastId || current.at(-1) === lastId) {
    return current;
  }
  return [...current, lastId];
};

export const previousCursor = (current: string[]) => current.slice(0, -1);

const adminFetch = (url: string, signal?: AbortSignal) => fetch(url, { credentials: "include", signal });

const readError = async (response: Response, fallback: string) => {
  const text = await response.text();
  return text.trim() || fallback;
};

export const fetchInstallationFleets = async (signal?: AbortSignal) => {
  const response = await adminFetch("/admin/api/installation/fleets", signal);
  if (!response.ok) {
    throw new Error(await readError(response, "Failed to load fleets"));
  }
  const data: { fleets?: InstallationFleet[] } = await response.json();
  return data.fleets ?? [];
};

export const fetchRunner = async (fleetId: string, runnerId: string, signal?: AbortSignal): Promise<FleetRunner> => {
  const response = await adminFetch(fleetURL(fleetId, `/runners/${encodeURIComponent(runnerId)}`), signal);
  if (response.status === 404) {
    throw new Error("Runner not found.");
  }
  if (!response.ok) {
    throw new Error("Failed to load runner.");
  }
  const body: { runner: FleetRunner } = await response.json();
  return body.runner;
};

export const fetchBrokerTasks = async () => {
  const response = await adminFetch("/admin/api/runner/tasks");
  if (!response.ok) {
    throw new Error(await readError(response, "Failed to load runner tasks"));
  }
  const data: { configured: boolean; tasks?: BrokerTask[] } = await response.json();
  return { configured: data.configured, tasks: data.tasks ?? [] };
};

type ListBody<T> = {
  rows: T[];
  totalCount: unknown;
  hasNextPage?: boolean;
};

const recordPage = <T>(body: ListBody<T>): RecordPage<T> => ({
  rows: body.rows,
  totalCount: parseCount(body.totalCount),
  hasNextPage: body.hasNextPage === true,
});

export type FleetDetails = {
  capacity: FleetCapacity;
  runners: RecordPage<FleetRunner>;
  tasks: RecordPage<FleetTask>;
};

export const fetchFleetDetails = async (
  fleetId: string,
  runnerCursor: string | null,
  taskCursor: string | null,
  signal?: AbortSignal,
): Promise<FleetDetails> => {
  const [capacityResponse, runnerResponse, taskResponse] = await Promise.all([
    adminFetch(fleetURL(fleetId, "/capacity"), signal),
    adminFetch(`${fleetURL(fleetId, "/runners")}?${listQuery(RUNNER_STATES, runnerCursor)}`, signal),
    adminFetch(`${fleetURL(fleetId, "/tasks")}?${listQuery(TASK_STATES, taskCursor)}`, signal),
  ]);
  if (!capacityResponse.ok || !runnerResponse.ok || !taskResponse.ok) {
    throw new Error("Failed to load fleet");
  }

  const capacityBody: Record<string, unknown> = await capacityResponse.json();
  const runnerBody: { runners?: FleetRunner[]; totalCount?: unknown; hasNextPage?: boolean } =
    await runnerResponse.json();
  const taskBody: { tasks?: FleetTask[]; totalCount?: unknown; hasNextPage?: boolean } = await taskResponse.json();
  return {
    capacity: {
      queuedTasks: parseCount(capacityBody.runnableTasks),
      pendingRunners: parseCount(capacityBody.pendingRunners),
      idleRunners: parseCount(capacityBody.idleRunners),
      busyRunners: parseCount(capacityBody.busyRunners),
    },
    runners: recordPage({
      rows: runnerBody.runners ?? [],
      totalCount: runnerBody.totalCount,
      hasNextPage: runnerBody.hasNextPage,
    }),
    tasks: recordPage({
      rows: taskBody.tasks ?? [],
      totalCount: taskBody.totalCount,
      hasNextPage: taskBody.hasNextPage,
    }),
  };
};

export const selectedFleetIdAfterRefresh = (current: string | null, fleets: InstallationFleet[]) => {
  if (current && fleets.some((fleet) => fleet.id === current)) {
    return current;
  }
  return fleets[0]?.id ?? null;
};
