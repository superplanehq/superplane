export type RunStatus = "Running" | "Passed" | "Failed" | "Cancelled";

export type RunLogs =
  | { state: "ready"; text: string }
  | { state: "loading" | "waiting" | "empty" }
  | { state: "error"; message: string };

export interface RunActivity {
  id: string;
  name: string;
  status: RunStatus;
  time: string;
  logs: RunLogs;
  outputs?: Record<string, unknown>;
}

export interface AutomationRun {
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
