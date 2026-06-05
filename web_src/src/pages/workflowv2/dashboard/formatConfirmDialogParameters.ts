/** Pretty-print trigger parameters for read-only JSON preview blocks. */
export function formatParameters(parameters: Record<string, unknown> | undefined): string {
  if (!parameters || Object.keys(parameters).length === 0) return "(empty)";
  try {
    return JSON.stringify(parameters, null, 2);
  } catch {
    return String(parameters);
  }
}
