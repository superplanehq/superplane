/**
 * Color presets for the public badge.
 *
 * A theme is only a set of colors. The settings page loads a preset into
 * editable fields, and the badge URL carries every color the user changed.
 *
 * These palettes must stay equal to `themes` in `pkg/badges/theme.go`. A Go
 * test reads this file and compares the two tables, so update both together.
 */

/** URL parameter names for the colors the badge accepts. */
export const BADGE_COLOR_SLOTS = ["accent", "manual", "text", "muted", "bg", "border"] as const;

export type BadgeColorSlot = (typeof BADGE_COLOR_SLOTS)[number];

export type BadgeColors = Record<BadgeColorSlot, string>;

export const BADGE_COLOR_LABELS: Record<BadgeColorSlot, string> = {
  accent: "Accent",
  manual: "Manual work",
  text: "Text",
  muted: "Muted text",
  bg: "Background",
  border: "Border",
};

export const BADGE_THEMES = [
  {
    value: "superplane",
    label: "SuperPlane",
    colors: {
      accent: "#10b981",
      manual: "#64748b",
      text: "#f0f6fc",
      muted: "#8b949e",
      bg: "#0d1117",
      border: "#30363d",
    },
  },
  {
    value: "light",
    label: "Light",
    colors: {
      accent: "#059669",
      manual: "#8c959f",
      text: "#1f2328",
      muted: "#59636e",
      bg: "#ffffff",
      border: "#d1d9e0",
    },
  },
  {
    value: "github_dark",
    label: "GitHub dark",
    colors: {
      accent: "#58a6ff",
      manual: "#484f58",
      text: "#c9d1d9",
      muted: "#8b949e",
      bg: "#0d1117",
      border: "#30363d",
    },
  },
  {
    value: "github_light",
    label: "GitHub light",
    colors: {
      accent: "#0969da",
      manual: "#8c959f",
      text: "#1f2328",
      muted: "#59636e",
      bg: "#ffffff",
      border: "#d1d9e0",
    },
  },
  {
    value: "dracula",
    label: "Dracula",
    colors: {
      accent: "#50fa7b",
      manual: "#6272a4",
      text: "#f8f8f2",
      muted: "#6272a4",
      bg: "#282a36",
      border: "#44475a",
    },
  },
  {
    value: "tokyonight",
    label: "Tokyo Night",
    colors: {
      accent: "#7aa2f7",
      manual: "#414868",
      text: "#c0caf5",
      muted: "#565f89",
      bg: "#1a1b27",
      border: "#292e42",
    },
  },
  {
    value: "nord",
    label: "Nord",
    colors: {
      accent: "#88c0d0",
      manual: "#4c566a",
      text: "#d8dee9",
      muted: "#7b88a1",
      bg: "#2e3440",
      border: "#3b4252",
    },
  },
  {
    value: "gruvbox",
    label: "Gruvbox",
    colors: {
      accent: "#b8bb26",
      manual: "#665c54",
      text: "#ebdbb2",
      muted: "#a89984",
      bg: "#282828",
      border: "#3c3836",
    },
  },
  {
    value: "catppuccin_mocha",
    label: "Catppuccin Mocha",
    colors: {
      accent: "#a6e3a1",
      manual: "#45475a",
      text: "#cdd6f4",
      muted: "#6c7086",
      bg: "#1e1e2e",
      border: "#313244",
    },
  },
] as const satisfies ReadonlyArray<{ value: string; label: string; colors: BadgeColors }>;

export type BadgeTheme = (typeof BADGE_THEMES)[number]["value"];

export const DEFAULT_BADGE_THEME: BadgeTheme = BADGE_THEMES[0].value;

export function badgeThemeColors(theme: BadgeTheme): BadgeColors {
  const preset = BADGE_THEMES.find((option) => option.value === theme) ?? BADGE_THEMES[0];
  return { ...preset.colors };
}

export function badgeThemeLabel(theme: BadgeTheme): string {
  return BADGE_THEMES.find((option) => option.value === theme)?.label ?? theme;
}

/** The renderer ignores anything that is not a six-digit hex color. */
export function isBadgeColor(value: string): boolean {
  return /^#[0-9a-fA-F]{6}$/.test(value);
}

/**
 * Colors the user changed away from the preset. The badge URL carries only
 * these, so a snippet that uses a preset stays short.
 */
export function changedBadgeColors(theme: BadgeTheme, colors: BadgeColors): Array<[BadgeColorSlot, string]> {
  const preset = badgeThemeColors(theme);
  return BADGE_COLOR_SLOTS.filter(
    (slot) => isBadgeColor(colors[slot]) && colors[slot].toLowerCase() !== preset[slot].toLowerCase(),
  ).map((slot) => [slot, colors[slot].toLowerCase()]);
}
