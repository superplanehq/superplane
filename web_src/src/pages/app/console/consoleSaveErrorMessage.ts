/**
 * Extracts the user-facing "reason" out of an error thrown by
 * `useUpdateCanvasConsole.mutate`.
 *
 * The mutation wraps validation failures (delta cap exceeded, malformed
 * shape, unknown panel type, ...) as `new Error("invalid console yaml:
 * <detail>")`. Users don't need to see the wrapper prefix — the
 * interesting signal is the underlying reason (e.g., "Too many panels
 * (max 20 per page)."). Non-wrapped errors (network 5xx, aborted fetch,
 * ...) are surfaced verbatim so operators can still act on them.
 *
 * Extracted so the toast wiring in `ConsoleOverlay.handleChange` and
 * `useSpecFileAutosave.persistConsoleSpec` stays a one-liner and both
 * call sites share the same normalization. Each caller is free to add
 * whatever prefix fits its surface ("Failed to save console: …" for
 * the overlay, "Could not save console.yaml: …" for the Files tab
 * autosave, which routes through `onSpecParseError`).
 */
export function consoleSaveErrorReason(error: unknown): string {
  const raw = error instanceof Error ? error.message : String(error);
  const prefix = "invalid console yaml: ";
  const reason = (raw.startsWith(prefix) ? raw.slice(prefix.length) : raw).trim();
  return reason || "unknown error";
}

/**
 * Convenience wrapper for the overlay: `Failed to save console: <reason>`.
 * Kept for symmetry with the earlier API and to isolate the toast
 * copy for the overlay from the Files-tab path.
 */
export function formatConsoleSaveErrorMessage(error: unknown): string {
  return `Failed to save console: ${consoleSaveErrorReason(error)}`;
}
