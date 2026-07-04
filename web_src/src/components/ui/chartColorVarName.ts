/** Map a chart config / data series key to a valid CSS custom-property suffix. */
export function toChartColorVarName(key: string): string {
  const trimmed = key.trim();
  if (!trimmed) return "empty";
  return trimmed
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
}
