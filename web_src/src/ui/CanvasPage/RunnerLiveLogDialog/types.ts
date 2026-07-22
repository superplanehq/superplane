import type { ExecutionInfo } from "../../../pages/app/mappers/types";

export type RunnerLiveLogDialogProps = {
  title: string;
  canvasMode: "live" | "edit";
  execution: ExecutionInfo | null;
};

export function isExecutionInFlight(execution: ExecutionInfo): boolean {
  return (
    execution.state === "STATE_PENDING" || execution.state === "STATE_STARTED" || execution.state === "STATE_CANCELLING"
  );
}

export type LiveLogRecord =
  | { type: "line"; text: string }
  | { type: "error"; message: string }
  | { type: "cmd_start"; index: number; text: string; started_at?: number }
  | { type: "cmd_end"; index: number; status: "passed" | "failed"; duration_ms: number };

export type CommandSection = {
  index: number;
  text: string;
  lines: string[];
  status: "running" | "passed" | "failed";
  duration_ms: number | null;
  started_at: number | null;
  collapsed: boolean;
};

export type LogState = {
  sections: CommandSection[];
  orphanLines: string[];
  error: string | null;
  isStreaming: boolean;
};
