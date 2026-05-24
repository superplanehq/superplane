import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
type LiveLogRecordEnvelope = {
  type?: string;
  text?: string;
  message?: string;
  index?: number;
  status?: "passed" | "failed";
  duration_ms?: number;
  started_at?: number;
};

export type LiveLogStreamHandlers = {
  onLogLine: (text: string) => void;
  onStreamError: (message: string) => void;
  onCmdStart?: (index: number, text: string, startedAtMs: number | null) => void;
  onCmdEnd?: (index: number, status: "passed" | "failed", durationMs: number) => void;
};

async function fetchRunnerLiveLogResponse(url: string, organizationId: string, signal: AbortSignal): Promise<Response> {
  const res = await fetch(url, {
    method: "GET",
    credentials: "include",
    signal,
    ...withOrganizationHeader({
      organizationId,
      headers: { Accept: "application/x-ndjson" },
    }),
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
  handlers.onLogLine(rec.text);
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
  handlers.onCmdStart?.(rec.index, rec.text, parseStartedAtMs(rec.started_at));
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
  dispatchCmdEndRecord(rec, handlers);
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
    const rec = tryParseLiveLogRecord(line);
    if (rec) {
      dispatchLiveLogRecord(rec, handlers);
    }
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

/**
 * Fetches runner NDJSON live logs and dispatches parsed records to handlers until the stream ends or aborts.
 */
export class LiveLogStream {
  private readonly organizationId: string;
  private readonly url: string;
  private readonly abortController: AbortController;

  constructor(organizationId: string, canvasId: string, executionId: string) {
    this.organizationId = organizationId;
    this.url = `/api/v1/canvases/${encodeURIComponent(canvasId)}/node-executions/${encodeURIComponent(executionId)}/runner-live-logs`;
    this.abortController = new AbortController();
  }

  stop() {
    this.abortController.abort();
  }

  async pump(handlers: LiveLogStreamHandlers): Promise<void> {
    const res = await fetchRunnerLiveLogResponse(this.url, this.organizationId, this.abortController.signal);
    const reader = requireBodyReader(res);
    await pumpReaderNdjson(reader, handlers);
  }
}
