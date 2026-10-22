import type { IntakeRunOutcome } from "./intakeRunOutcome";

export type RunStatus = "Running" | "Passed" | "Failed" | "Cancelled" | "Pending" | "Cancelling" | "Unknown";

export type RunLogs =
  | { state: "ready"; text: string }
  | { state: "loading" | "waiting" | "empty" }
  | { state: "error"; message: string };

export interface RunActivity {
  id: string;
  name: string;
  status: RunStatus;
  recordedAt?: string;
  error?: string;
  logs: RunLogs;
  outputs?: Record<string, unknown>;
  additionalInstructions?: string;
  defaultTab?: "logs" | "outputs";
  outputsLabel?: string;
}

export interface AutomationRun {
  title?: string;
  source?: { reference: string; title: string; url?: string };
  intake?: { outcome: IntakeRunOutcome; summary: string; event: string };
  id: string;
  task?: { id: string; title: string; url?: string };
  status: RunStatus;
  startedAt: string;
  finishedAt?: string;
  duration: string;
  error?: string;
  activities: RunActivity[];
}
