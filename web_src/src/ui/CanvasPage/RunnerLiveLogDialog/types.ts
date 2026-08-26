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
  | { type: "cmd_start"; index: number; text: string; kind?: string; preview?: string; started_at?: number }
  | { type: "cmd_end"; index: number; status: "passed" | "failed"; duration_ms: number }
  | { type: "tool_start"; id?: string; kind?: string; text?: string; started_at?: number }
  | { type: "tool_end"; id?: string; kind?: string; status: "passed" | "failed"; duration_ms: number };

export type CommandTool = {
  id: string;
  sourceId?: string;
  kind: string;
  text: string;
  lines: string[];
  status: "running" | "passed" | "failed";
  duration_ms: number | null;
};

export type CommandSectionEvent = { kind: "note"; text: string } | { kind: "tools"; id: string; tools: CommandTool[] };

export type CommandSection = {
  index: number;
  text: string;
  kind?: string;
  preview?: string;
  lines: string[];
  events: CommandSectionEvent[];
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
