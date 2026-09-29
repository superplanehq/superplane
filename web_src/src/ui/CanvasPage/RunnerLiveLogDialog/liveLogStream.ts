import { parseAgentActivityRecordText, type AgentActivityRecord } from "@/lib/agentActivity";
import {
  applyPromptUsageRecord,
  emptyPromptUsageState,
  parseAgentTurnLiveLogText,
  promptUsageSeries,
  startPromptUsageSeries,
  type AgentPromptUsageSeries,
} from "@/lib/agentRunTelemetry";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

export type LiveLogRecordEnvelope = {
  type?: string;
  text?: string;
  kind?: string;
  preview?: string;
  id?: string;
  message?: string;
  index?: number;
  turn?: number;
  usage?: Record<string, number>;
  status?: "passed" | "failed";
  duration_ms?: number;
  started_at?: number;
  schema_version?: number;
  event_id?: string;
  activity_id?: string;
  sequence?: number;
  timestamp?: string;
  provider?: string;
  channel?: string;
  content_id?: string;
  tool_id?: string;
  name?: string;
  input?: string;
  partial_json?: string;
  complete?: boolean;
  output_stream?: string;
  exit_code?: number;
  signal?: string;
  truncated?: boolean;
};

type LiveLogSessionResponse = {
  backend?: "legacy" | "integrated";
  stream_url?: string;
  token?: string;
  expires_at?: string;
};

export type LiveLogStreamHandlers = {
  onOpen?: () => void;
  onRecord?: (record: AgentActivityRecord) => void;
  onLogLine: (text: string, commandIndex?: number) => void;
  onStreamError: (message: string) => void;
  onCmdStart?: (index: number, text: string, startedAtMs: number | null, kind?: string, preview?: string) => void;
  onCmdEnd?: (index: number, status: "passed" | "failed", durationMs: number) => void;
  onToolStart?: (kind: string, text: string, id?: string, turn?: number, commandIndex?: number) => void;
  onToolEnd?: (
    status: "passed" | "failed",
    durationMs: number,
    id?: string,
    turn?: number,
    commandIndex?: number,
  ) => void;
  onTurn?: (turn: number, usage: Record<string, number>, message?: string) => void;
};

async function fetchRunnerLiveLogSession(
  sessionUrl: string,
  organizationId: string,
  signal: AbortSignal,
): Promise<LiveLogSessionResponse> {
  const res = await fetch(
    sessionUrl,
    withOrganizationHeader({
      organizationId,
      signal,
      credentials: "include",
      headers: { Accept: "application/json" },
    }),
  );

  if (!res.ok) {
    const body = await res.text();
    throw new Error(body.trim() || res.statusText || `Request failed (${res.status})`);
  }

  return (await res.json()) as LiveLogSessionResponse;
}

async function fetchRunnerLiveLogResponse(
  session: RequiredLiveLogSession,
  organizationId: string,
  signal: AbortSignal,
): Promise<Response> {
  const init =
    session.backend === "integrated"
      ? withOrganizationHeader({
          organizationId,
          method: "GET",
          credentials: "include" as const,
          signal,
          headers: { Accept: "application/x-ndjson" },
        })
      : {
          method: "GET",
          credentials: "omit" as const,
          signal,
          headers: {
            Accept: "application/x-ndjson",
            Authorization: `Bearer ${session.token}`,
            "Accept-Encoding": "identity",
          },
        };
  const res = await fetch(session.streamUrl, init);

  if (!res.ok) {
    const body = await res.text();
    throw new Error(body.trim() || res.statusText || `Request failed (${res.status})`);
  }

  return res;
}

function requireBodyReader(res: Response): ReadableStreamDefaultReader<Uint8Array> {
  const reader = res.body?.getReader();
  if (!reader) {
    throw new Error("No response body");
  }
  return reader;
}

function tryParseLiveLogRecord(line: string): LiveLogRecordEnvelope | null {
  try {
    return JSON.parse(line) as LiveLogRecordEnvelope;
  } catch {
    return null;
  }
}

function parseStartedAtMs(value: number | undefined): number | null {
  return typeof value === "number" && value >= 0 ? value : null;
}

function dispatchLineRecord(rec: LiveLogRecordEnvelope, handlers: LiveLogStreamHandlers): boolean {
  if (rec.type !== "line" || typeof rec.text !== "string") {
    return false;
  }
  const nestedTurn = parseAgentTurnLiveLogText(rec.text);
  if (nestedTurn) {
    handlers.onTurn?.(nestedTurn.turn, nestedTurn.usage, nestedTurn.message);
    return true;
  }
  const nestedActivity = parseAgentActivityRecordText(rec.text);
  if (nestedActivity) {
    handlers.onRecord?.(nestedActivity);
    return true;
  }
  if (typeof rec.index === "number") {
    handlers.onLogLine(rec.text, rec.index);
  } else {
    handlers.onLogLine(rec.text);
  }
  return true;
}

function dispatchErrorRecord(rec: LiveLogRecordEnvelope, handlers: LiveLogStreamHandlers): boolean {
  if (rec.type !== "error" || typeof rec.message !== "string") {
    return false;
  }
  handlers.onStreamError(rec.message);
  return true;
}

function dispatchCmdStartRecord(rec: LiveLogRecordEnvelope, handlers: LiveLogStreamHandlers): boolean {
  if (rec.type !== "cmd_start" || typeof rec.index !== "number" || typeof rec.text !== "string") {
    return false;
  }
  handlers.onCmdStart?.(
    rec.index,
    rec.text,
    parseStartedAtMs(rec.started_at),
    typeof rec.kind === "string" ? rec.kind : undefined,
    typeof rec.preview === "string" ? rec.preview : undefined,
  );
  return true;
}

function dispatchToolStartRecord(rec: LiveLogRecordEnvelope, handlers: LiveLogStreamHandlers): boolean {
  if (rec.type !== "tool_start") {
    return false;
  }
  handlers.onToolStart?.(
    typeof rec.kind === "string" ? rec.kind : "tool",
    typeof rec.text === "string" ? rec.text : "",
    typeof rec.id === "string" ? rec.id : undefined,
    typeof rec.turn === "number" ? rec.turn : undefined,
    typeof rec.index === "number" ? rec.index : undefined,
  );
  return true;
}

function dispatchToolEndRecord(rec: LiveLogRecordEnvelope, handlers: LiveLogStreamHandlers): boolean {
  if (
    rec.type !== "tool_end" ||
    (rec.status !== "passed" && rec.status !== "failed") ||
    typeof rec.duration_ms !== "number"
  ) {
    return false;
  }
  handlers.onToolEnd?.(
    rec.status,
    rec.duration_ms,
    typeof rec.id === "string" ? rec.id : undefined,
    typeof rec.turn === "number" ? rec.turn : undefined,
    typeof rec.index === "number" ? rec.index : undefined,
  );
  return true;
}

function dispatchTurnRecord(rec: LiveLogRecordEnvelope, handlers: LiveLogStreamHandlers): boolean {
  if (rec.type !== "turn" || typeof rec.turn !== "number") {
    return false;
  }
  handlers.onTurn?.(
    rec.turn,
    rec.usage && typeof rec.usage === "object" ? rec.usage : {},
    typeof rec.message === "string" ? rec.message : undefined,
  );
  return true;
}

function dispatchCmdEndRecord(rec: LiveLogRecordEnvelope, handlers: LiveLogStreamHandlers): boolean {
  if (
    rec.type !== "cmd_end" ||
    typeof rec.index !== "number" ||
    (rec.status !== "passed" && rec.status !== "failed") ||
    typeof rec.duration_ms !== "number"
  ) {
    return false;
  }
  handlers.onCmdEnd?.(rec.index, rec.status, rec.duration_ms);
  return true;
}

function dispatchLiveLogRecord(rec: LiveLogRecordEnvelope, handlers: LiveLogStreamHandlers): void {
  if (rec.schema_version === 2) {
    handlers.onRecord?.(rec);
    return;
  }
  if (dispatchLineRecord(rec, handlers)) {
    return;
  }
  if (dispatchErrorRecord(rec, handlers)) {
    return;
  }
  if (dispatchCmdStartRecord(rec, handlers)) {
    return;
  }
  if (dispatchCmdEndRecord(rec, handlers)) {
    return;
  }
  if (dispatchToolStartRecord(rec, handlers)) {
    return;
  }
  if (dispatchToolEndRecord(rec, handlers)) {
    return;
  }
  dispatchTurnRecord(rec, handlers);
}

export function consumeLiveLogNdjsonLine(line: string, handlers: LiveLogStreamHandlers): void {
  const rec = tryParseLiveLogRecord(line.trim());
  if (rec) {
    dispatchLiveLogRecord(rec, handlers);
  }
}

export function reducePromptUsageFromLiveLogLines(lines: string[]): AgentPromptUsageSeries[] {
  let state = emptyPromptUsageState();
  const handlers: LiveLogStreamHandlers = {
    onLogLine: () => undefined,
    onStreamError: () => undefined,
    onCmdStart: (index, text, _startedAtMs, kind) => {
      if (kind === "prompt") {
        state = startPromptUsageSeries(state, text, index);
      }
    },
    onToolStart: (kind, text, id, turn) => {
      state = applyPromptUsageRecord(state, { type: "tool_start", kind, text, id, turn });
    },
    onToolEnd: (status, durationMs, id, turn) => {
      state = applyPromptUsageRecord(state, { type: "tool_end", status, duration_ms: durationMs, id, turn });
    },
    onTurn: (turn, usage, message) => {
      state = applyPromptUsageRecord(state, { type: "turn", turn, usage, message });
    },
  };
  for (const line of lines) {
    consumeLiveLogNdjsonLine(line, handlers);
  }
  return promptUsageSeries(state);
}

/** Consumes complete NDJSON lines from buffer; returns the trailing incomplete fragment. */
function processCompleteLines(buffer: string, handlers: LiveLogStreamHandlers): string {
  let remainder = buffer;
  let newlineIndex: number;
  while ((newlineIndex = remainder.indexOf("\n")) >= 0) {
    const line = remainder.slice(0, newlineIndex).trim();
    remainder = remainder.slice(newlineIndex + 1);
    if (!line) {
      continue;
    }
    consumeLiveLogNdjsonLine(line, handlers);
  }
  return remainder;
}

async function pumpReaderNdjson(
  reader: ReadableStreamDefaultReader<Uint8Array>,
  handlers: LiveLogStreamHandlers,
): Promise<void> {
  const decoder = new TextDecoder();
  let buffer = "";

  for (;;) {
    const { done, value } = await reader.read();
    if (done) {
      break;
    }
    buffer += decoder.decode(value, { stream: true });
    buffer = processCompleteLines(buffer, handlers);
  }
}

type RequiredLiveLogSession = {
  backend: "legacy" | "integrated";
  streamUrl: string;
  token?: string;
};

function requireLiveLogSession(session: LiveLogSessionResponse): RequiredLiveLogSession {
  const streamUrl = session.stream_url?.trim();
  const token = session.token?.trim();
  const backend = session.backend === "integrated" ? "integrated" : "legacy";
  if (!streamUrl || (backend === "legacy" && !token)) {
    throw new Error("Live log session response is incomplete");
  }
  return { backend, streamUrl, token };
}

/**
 * Resolves the execution's log backend through SuperPlane, then consumes its
 * NDJSON stream until the stream ends or aborts.
 */
export class LiveLogStream {
  private readonly organizationId: string;
  private readonly sessionUrl: string;
  private readonly abortController: AbortController;

  constructor(organizationId: string, canvasId: string, executionId: string) {
    this.organizationId = organizationId;
    this.sessionUrl = `/api/v1/canvases/${encodeURIComponent(canvasId)}/node-executions/${encodeURIComponent(executionId)}/runner-live-logs/session`;
    this.abortController = new AbortController();
  }

  stop() {
    this.abortController.abort();
  }

  async pump(handlers: LiveLogStreamHandlers): Promise<void> {
    const session = await fetchRunnerLiveLogSession(this.sessionUrl, this.organizationId, this.abortController.signal);
    const required = requireLiveLogSession(session);
    const res = await fetchRunnerLiveLogResponse(required, this.organizationId, this.abortController.signal);
    const reader = requireBodyReader(res);
    handlers.onOpen?.();
    await pumpReaderNdjson(reader, handlers);
  }
}
