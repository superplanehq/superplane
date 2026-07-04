const AGENTS_DISABLED_CODE = 14;
const AGENTS_DISABLED_MESSAGE = "agents are not enabled on this installation";

export type AgentSetupState = "failed" | "loading" | "unavailable";

export function getAgentSetupState({
  chatId,
  error,
  isError,
  isFetching,
  isLoading,
}: {
  chatId: string | null;
  error: unknown;
  isError: boolean;
  isFetching: boolean;
  isLoading: boolean;
}): AgentSetupState | null {
  const chatFailed = isError && !isFetching && !chatId;
  if (chatFailed && isAgentsDisabledError(error)) return "unavailable";
  if (chatFailed) return "failed";
  if (isLoading || !chatId) return "loading";
  return null;
}

const SESSION_BUSY_CODE = 9; // gRPC FAILED_PRECONDITION — e.g. reset while the turn is still streaming.

// True when the server rejected a request because the agent session is busy, so
// callers can show the intentional "wait" notice instead of a raw error.
export function isSessionBusyError(error: unknown): boolean {
  return getErrorStatusCandidates(error).some((status) => {
    if (!status || typeof status !== "object") return false;
    return (status as { code?: unknown }).code === SESSION_BUSY_CODE;
  });
}

function isAgentsDisabledError(error: unknown): boolean {
  return getErrorStatusCandidates(error).some(isAgentsDisabledStatus);
}

function isAgentsDisabledStatus(status: unknown): boolean {
  if (!status || typeof status !== "object") return false;

  const maybeStatus = status as { code?: unknown; message?: unknown };
  return maybeStatus.code === AGENTS_DISABLED_CODE && maybeStatus.message === AGENTS_DISABLED_MESSAGE;
}

function getErrorStatusCandidates(error: unknown, seen = new Set<object>()): unknown[] {
  if (!error || typeof error !== "object" || seen.has(error)) return [];
  seen.add(error);

  const record = error as Record<string, unknown>;
  return [
    error,
    ...getNestedErrorStatusCandidates(record.response, seen),
    ...getNestedErrorStatusCandidates(record.error, seen),
  ];
}

function getNestedErrorStatusCandidates(error: unknown, seen: Set<object>): unknown[] {
  if (!error || typeof error !== "object") return [];

  const record = error as Record<string, unknown>;
  return [error, ...getErrorStatusCandidates(record.data, seen), ...getErrorStatusCandidates(record.error, seen)];
}
