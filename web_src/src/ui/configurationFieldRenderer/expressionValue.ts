/**
 * Detect if value looks like a wrapped expression (e.g. {{ $["node-name"].value }}).
 * Requires both {{ and }} so fixed IDs are not misclassified.
 */
export function isExpressionValue(value: unknown): boolean {
  if (value == null) return false;
  const str = Array.isArray(value) ? value[0] : value;
  if (typeof str !== "string") return false;
  const trimmed = str.trim();
  if (!trimmed.length) return false;
  return /\{\{[\s\S]*?\}\}/.test(trimmed);
}
