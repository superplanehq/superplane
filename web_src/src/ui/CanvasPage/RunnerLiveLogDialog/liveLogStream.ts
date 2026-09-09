import {
  applyPromptUsageRecord,
  emptyPromptUsageState,
  parseAgentTurnLiveLogText,
  promptUsageSeries,
  startPromptUsageSeries,
  type AgentPromptUsageSeries,
} from "@/lib/agentRunTelemetry";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

type LiveLogRecordEnvelope = {
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
};

type LiveLogSessionResponse = {
  stream_url?: string;
  token?: string;
  expires_at?: string;
};

export type LiveLogStreamHandlers = {
  onOpen?: () => void;
  onLogLine: (text: string, commandIndex?: number) => void;
  onStreamError: (message: string) => void;
  onCmdStart?: (index: number, text: string, startedAtMs: number | null, kind?: string, preview?: string) => void;
  onCmdEnd?: (index: number, status: "passed" | "failed", durationMs: number) => void;
  onToolStart?: (kind: string, text: string, id?: string, turn?: number) => void;
  onToolEnd?: (status: "passed" | "failed", durationMs: number, id?: string, turn?: number) => void;
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

async function fetchRunnerLiveLogResponse(url: string, token: string, signal: AbortSignal): Promise<Response> {
  const res = await fetch(url, {
    method: "GET",
    credentials: "omit",
    signal,
    headers: {
      Accept: "application/x-ndjson",
      Authorization: `Bearer ${token}`,
      "Accept-Encoding": "identity",
    },
  });

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

function requireLiveLogSession(session: LiveLogSessionResponse): { streamUrl: string; token: string } {
  const streamUrl = session.stream_url?.trim();
  const token = session.token?.trim();
  if (!streamUrl || !token) {
    throw new Error("Live log session response is incomplete");
  }
  return { streamUrl, token };
}

/**
 * Fetches a short-lived task-broker stream session from SuperPlane, then consumes NDJSON live logs
 * directly from the task broker until the stream ends or aborts.
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
    const { streamUrl, token } = requireLiveLogSession(session);
    const res = await fetchRunnerLiveLogResponse(streamUrl, token, this.abortController.signal);
    const reader = requireBodyReader(res);
    handlers.onOpen?.();
    await pumpReaderNdjson(reader, handlers);
  }
}
