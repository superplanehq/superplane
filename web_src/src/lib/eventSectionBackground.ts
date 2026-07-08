/**
 * Literal class strings so Tailwind detects dark: utilities (dynamic template strings are not scanned).
 * Light classes are preserved; dark: classes are additive only.
 */
const EVENT_SECTION_BACKGROUND_CLASSES: Record<string, string> = {
  "bg-amber-100": "bg-amber-100 dark:bg-amber-900/50",
  "bg-blue-100": "bg-blue-100 dark:bg-blue-900/50",
  "bg-gray-50": "bg-gray-50 dark:bg-gray-900/50",
  "bg-gray-100": "bg-gray-100 dark:bg-gray-900/50",
  "bg-green-100": "bg-green-100 dark:bg-green-900/50",
  "bg-indigo-100": "bg-indigo-100 dark:bg-indigo-900/50",
  "bg-orange-100": "bg-orange-100 dark:bg-orange-900/50",
  "bg-red-100": "bg-red-100 dark:bg-red-900/50",
  "bg-sky-100": "bg-sky-100 dark:bg-sky-900/50",
  "bg-violet-100": "bg-violet-100 dark:bg-violet-900/50",
  "bg-yellow-100": "bg-yellow-100 dark:bg-yellow-900/50",
};

export function withEventSectionDarkBackground(backgroundColor: string): string {
  const trimmed = backgroundColor.trim();
  if (/\bdark:bg-/.test(trimmed)) {
    return trimmed;
  }

  return trimmed
    .split(/\s+/)
    .map((cls) => EVENT_SECTION_BACKGROUND_CLASSES[cls] ?? cls)
    .join(" ");
}
