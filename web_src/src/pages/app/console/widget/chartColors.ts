/** Tailwind default hex literals for Recharts fill/stroke props. */
export const CHART_COLOR = {
  blue600: "#2563eb",
  blue500: "#3b82f6",
  blue400: "#60a5fa",
  emerald600: "#059669",
  emerald500: "#10b981",
  emerald400: "#34d399",
  red500: "#ef4444",
} as const;

/** Softer palette for dark mode — slightly lighter blues/greens for contrast on gray-900 panels. */
export const DARK_CHART_COLOR = {
  blue500: "#60a5fa",
  blue400: "#93c5fd",
  blue300: "#bfdbfe",
  emerald500: "#34d399",
  emerald400: "#6ee7b7",
  emerald300: "#86efac",
  red400: "#f87171",
} as const;

export const DEFAULT_CHART_PALETTE = [
  CHART_COLOR.blue600,
  CHART_COLOR.blue500,
  CHART_COLOR.blue400,
  CHART_COLOR.emerald600,
  CHART_COLOR.emerald500,
  CHART_COLOR.emerald400,
];

export const DEFAULT_DARK_CHART_PALETTE = [
  DARK_CHART_COLOR.blue500,
  DARK_CHART_COLOR.blue400,
  DARK_CHART_COLOR.blue300,
  DARK_CHART_COLOR.emerald500,
  DARK_CHART_COLOR.emerald400,
  DARK_CHART_COLOR.emerald300,
];

const SEMANTIC_CHART_COLORS: Record<string, string> = {
  passed: CHART_COLOR.emerald500,
  failed: CHART_COLOR.red500,
};

const SEMANTIC_DARK_CHART_COLORS: Record<string, string> = {
  passed: DARK_CHART_COLOR.emerald400,
  failed: DARK_CHART_COLOR.red400,
};

function normalizeChartColorKey(name: string): string {
  return name.trim().toLowerCase();
}

/**
 * Resolve a series or slice color. Known status names (`passed`, `failed`)
 * map to semantic colors; everything else uses the default palette by index.
 *
 * Stored `series[].color` values in dashboard YAML are ignored until the panel
 * editor exposes an intentional color picker.
 */
export function resolveChartColor(name: string, paletteIndex: number, isDark = false): string {
  const semanticMap = isDark ? SEMANTIC_DARK_CHART_COLORS : SEMANTIC_CHART_COLORS;
  const semantic = semanticMap[normalizeChartColorKey(name)];
  if (semantic) return semantic;
  const palette = isDark ? DEFAULT_DARK_CHART_PALETTE : DEFAULT_CHART_PALETTE;
  return palette[paletteIndex % palette.length]!;
}
