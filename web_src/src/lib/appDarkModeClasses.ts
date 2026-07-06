/**
 * Shared dark-mode class additions for the App experience.
 * Light classes are preserved; append these via cn() where patterns repeat.
 */
export const appDarkModeClasses = {
  surface: "dark:bg-gray-900",
  surfaceRaised: "dark:bg-gray-800",
  border: "dark:border-gray-800/70",
  /** Matches PageHeader bottom border — sidebar vertical edges. */
  sidebarEdge: "border-slate-950/15 dark:border-gray-700/70",
  /** Horizontal rules inside sidebars — same color as sidebarEdge. */
  sidebarDivider: "border-slate-950/15 dark:border-gray-700/70",
  /** Floating panel edge — modals and dialogs (full border). */
  modalEdge: "border border-slate-950/15 dark:border-gray-700/70",
  textPrimary: "dark:text-gray-100",
  textSecondary: "dark:text-gray-400",
  textMuted: "dark:text-gray-500",
  hoverSurface: "dark:hover:bg-gray-800",
  activeTab: "dark:bg-gray-800 dark:text-gray-100 dark:shadow-none",
  /** Light: black fill. Dark: uses --primary tokens (indigo). */
  primaryAction:
    "bg-black text-white hover:bg-black/80 dark:bg-primary dark:text-primary-foreground dark:hover:bg-primary/90",
  /** Soft destructive: light red fill with dark red text (matches Commit/orange-style buttons). */
  destructiveSoftAction:
    "bg-red-400 text-red-950 hover:bg-red-500 hover:opacity-95 focus-visible:ring-red-500/40 dark:bg-red-400 dark:text-red-950 dark:hover:bg-red-500/90 dark:focus-visible:ring-red-500/40",
} as const;
