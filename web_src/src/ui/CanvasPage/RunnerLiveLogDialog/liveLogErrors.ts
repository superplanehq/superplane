export const LIVE_LOG_ERROR_CODE_HEADER = "X-Superplane-Error-Code";
export const LIVE_LOG_SESSION_NOT_READY_CODE = "live_log_session_not_ready";

const BENIGN_LOG_STREAM_NOT_FOUND_PATTERN = /ResourceNotFoundException.*log stream .*(does not exist|not found)/i;

const BROWSER_TRANSPORT_FAILURE_MESSAGES = new Set([
  "network error",
  "Failed to fetch",
  "NetworkError when attempting to fetch resource",
  "NetworkError when attempting to fetch resource.",
  "Load failed",
]);

export class LiveLogRequestError extends Error {
  readonly code: string | undefined;

  constructor(message: string, code?: string) {
    super(message);
    this.name = "LiveLogRequestError";
    this.code = code;
  }
}

export function liveLogRequestErrorFromResponse(res: Response, body: string): LiveLogRequestError {
  const code = res.headers.get(LIVE_LOG_ERROR_CODE_HEADER)?.trim();
  return new LiveLogRequestError(body.trim() || res.statusText || `Request failed (${res.status})`, code || undefined);
}

export function isLiveLogSessionNotReady(error: unknown): boolean {
  return error instanceof LiveLogRequestError && error.code === LIVE_LOG_SESSION_NOT_READY_CODE;
}

// Opening live logs can race the runner. The session 404s until a task id
// exists, and CloudWatch GetLogEvents raises ResourceNotFoundException until
// the log stream exists. Both are expected waits. Do not report them as failures.
export function isBenignLiveLogWait(error: unknown): boolean {
  if (isLiveLogSessionNotReady(error)) {
    return true;
  }
  const message = error instanceof Error ? error.message : String(error);
  return BENIGN_LOG_STREAM_NOT_FOUND_PATTERN.test(message);
}

export function isBrowserTransportFailure(error: unknown): boolean {
  if (error instanceof LiveLogRequestError) {
    return false;
  }
  const message = error instanceof Error ? error.message : typeof error === "string" ? error : "";
  return BROWSER_TRANSPORT_FAILURE_MESSAGES.has(message);
}
