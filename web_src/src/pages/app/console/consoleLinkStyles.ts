/**
 * Default inline link appearance used across console panels. Sky text in light
 * mode; indigo in dark. Uses `hover:!underline` because unlayered
 * `a { text-decoration: inherit }` in `index.css` beats layered Tailwind utilities.
 */
export const CONSOLE_LINK_CLASSES =
  "text-sky-600 no-underline underline-offset-2 hover:!underline hover:text-sky-700 dark:text-indigo-300 dark:hover:text-indigo-200";

/**
 * Tailwind selector utilities applied to all `<a>` tags inside console panel
 * bodies and shared markdown (`MarkdownContent`). For HTML panels, mount on a
 * wrapper outside `dark-mode-disabled` so `dark:` link colors still apply when
 * the app is in dark mode.
 */
export const CONSOLE_LINK_ANCHOR_SELECTOR_CLASSES =
  "[&_a]:text-sky-600 [&_a]:no-underline [&_a]:underline-offset-2 [&_a]:hover:!underline [&_a]:hover:text-sky-700 dark:[&_a]:text-indigo-300 dark:[&_a]:hover:!underline dark:[&_a]:hover:text-indigo-200";
