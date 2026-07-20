/**
 * Console grid panel surfaces. The canvas stays `dark:bg-gray-900`; panels use a
 * 5% white tint so cards lift slightly without a custom color.
 */
export const CONSOLE_PANEL_SHELL_SURFACE = "dark:bg-white/5";

/** Inner body regions inherit the shell tint — avoid stacking opaque gray-900. */
export const CONSOLE_PANEL_BODY_SURFACE = "dark:bg-transparent";
