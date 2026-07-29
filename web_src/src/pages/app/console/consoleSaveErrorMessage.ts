/**
 * Formats an error thrown by `useUpdateCanvasConsole.mutate` into the
 * user-facing string shown by the error toast in `ConsoleOverlay`.
 *
 * The mutation wraps validation failures (delta cap exceeded, malformed
 * shape, unknown panel type, ...) as `new Error("invalid console yaml:
 * <detail>")`. Users don't need to see the wrapper prefix — the interesting
 * signal is the underlying reason (e.g., "Too many panels (max 20 per
 * page)."). Non-wrapped errors (network 5xx, aborted fetch, ...) are
 * surfaced verbatim so operators can still act on them.
 *
 * Extracted so the toast wiring in `ConsoleOverlay.handleChange` stays a
 * one-liner and this formatting stays covered by unit tests without
 * having to mount the whole overlay + a mocked React Query mutation.
 */
export function formatConsoleSaveErrorMessage(error: unknown): string {
  const raw = error instanceof Error ? error.message : String(error);
  const prefix = "invalid console yaml: ";
  const reason = (raw.startsWith(prefix) ? raw.slice(prefix.length) : raw).trim();
  return `Failed to save console: ${reason || "unknown error"}`;
}
