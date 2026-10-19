export type RunStatus = "Running" | "Passed" | "Failed" | "Cancelled" | "Pending" | "Cancelling" | "Unknown";

export type RunLogs =
  | { state: "ready"; text: string }
  | { state: "loading" | "waiting" | "empty" }
  | { state: "error"; message: string };

export interface RunActivity {
  id: string;
  name: string;
  status: RunStatus;
  time: string;
  recordedAt?: string;
  error?: string;
  logs: RunLogs;
  outputs?: Record<string, unknown>;
}

export interface AutomationRun {
  title?: string;
  id: string;
  task?: { id: string; title: string };
  status: RunStatus;
  started: string;
  startedAt: string;
  finishedAt?: string;
  duration: string;
  error?: string;
  activities: RunActivity[];
}
