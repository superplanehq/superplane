const NODE_ICON_COLOR_NAMES = [
  "slate",
  "gray",
  "zinc",
  "neutral",
  "stone",
  "red",
  "orange",
  "amber",
  "yellow",
  "lime",
  "green",
  "emerald",
  "teal",
  "cyan",
  "sky",
  "blue",
  "indigo",
  "violet",
  "purple",
  "fuchsia",
  "pink",
  "rose",
] as const;

/** Literal dark-mode icon classes so Tailwind JIT includes them. */
const NODE_ICON_DARK_TEXT_CLASS: Record<string, string> = {
  "slate-300": "dark:text-slate-300",
  "slate-400": "dark:text-slate-400",
  "gray-300": "dark:text-gray-300",
  "gray-400": "dark:text-gray-400",
  "zinc-300": "dark:text-zinc-300",
  "zinc-400": "dark:text-zinc-400",
  "neutral-300": "dark:text-neutral-300",
  "neutral-400": "dark:text-neutral-400",
  "stone-300": "dark:text-stone-300",
  "stone-400": "dark:text-stone-400",
  "red-300": "dark:text-red-300",
  "red-400": "dark:text-red-400",
  "orange-300": "dark:text-orange-300",
  "orange-400": "dark:text-orange-400",
  "amber-300": "dark:text-amber-300",
  "amber-400": "dark:text-amber-400",
  "yellow-300": "dark:text-yellow-300",
  "yellow-400": "dark:text-yellow-400",
  "lime-300": "dark:text-lime-300",
  "lime-400": "dark:text-lime-400",
  "green-300": "dark:text-green-300",
  "green-400": "dark:text-green-400",
  "emerald-300": "dark:text-emerald-300",
  "emerald-400": "dark:text-emerald-400",
  "teal-300": "dark:text-teal-300",
  "teal-400": "dark:text-teal-400",
  "cyan-300": "dark:text-cyan-300",
  "cyan-400": "dark:text-cyan-400",
  "sky-300": "dark:text-sky-300",
  "sky-400": "dark:text-sky-400",
  "blue-300": "dark:text-blue-300",
  "blue-400": "dark:text-blue-400",
  "indigo-300": "dark:text-indigo-300",
  "indigo-400": "dark:text-indigo-400",
  "violet-300": "dark:text-violet-300",
  "violet-400": "dark:text-violet-400",
  "purple-300": "dark:text-purple-300",
  "purple-400": "dark:text-purple-400",
  "fuchsia-300": "dark:text-fuchsia-300",
  "fuchsia-400": "dark:text-fuchsia-400",
  "pink-300": "dark:text-pink-300",
  "pink-400": "dark:text-pink-400",
  "rose-300": "dark:text-rose-300",
  "rose-400": "dark:text-rose-400",
};

function getNodeIconDarkTextClass(color: string, shade: 300 | 400): string | undefined {
  if (!NODE_ICON_COLOR_NAMES.includes(color as (typeof NODE_ICON_COLOR_NAMES)[number])) {
    return undefined;
  }

  return NODE_ICON_DARK_TEXT_CLASS[`${color}-${shade}`];
}

const COLOR_TEXT_CLASSES: Readonly<Record<string, string>> = {
  slate: "text-slate-600 dark:text-slate-400",
  gray: "text-gray-500 dark:text-gray-400",
  zinc: "text-gray-500 dark:text-gray-400",
  neutral: "text-neutral-600 dark:text-neutral-400",
  stone: "text-stone-600 dark:text-stone-400",
  red: "text-red-600 dark:text-red-400",
  orange: "text-orange-600 dark:text-orange-400",
  amber: "text-amber-600 dark:text-amber-400",
  yellow: "text-yellow-600 dark:text-yellow-400",
  lime: "text-lime-600 dark:text-lime-400",
  green: "text-green-600 dark:text-green-400",
  emerald: "text-emerald-600 dark:text-emerald-400",
  teal: "text-teal-600 dark:text-teal-400",
  cyan: "text-cyan-600 dark:text-cyan-400",
  sky: "text-sky-600 dark:text-sky-400",
  blue: "text-blue-600 dark:text-blue-400",
  indigo: "text-indigo-600 dark:text-indigo-400",
  violet: "text-violet-600 dark:text-violet-400",
  purple: "text-purple-600 dark:text-purple-400",
  fuchsia: "text-fuchsia-600 dark:text-fuchsia-400",
  pink: "text-pink-600 dark:text-pink-400",
  rose: "text-rose-600 dark:text-rose-400",
  white: "text-white dark:text-white",
  black: "text-gray-900 dark:text-gray-300",
};

export const getColorClass = (color?: string): string => {
  if (!color || !Object.prototype.hasOwnProperty.call(COLOR_TEXT_CLASSES, color)) {
    return COLOR_TEXT_CLASSES.gray;
  }

  return COLOR_TEXT_CLASSES[color];
};

/** Ensures canvas node header icons remain visible in dark mode. */
export function resolveNodeIconColorClass(iconColor?: string): string {
  const trimmed = iconColor?.trim();
  if (!trimmed) {
    return getColorClass("gray");
  }

  if (/\bdark:/.test(trimmed)) {
    return trimmed;
  }

  const match = /^text-([a-z]+)-(\d+)$/.exec(trimmed);
  if (match) {
    const [, color, shade] = match;
    const shadeNum = Number(shade);
    const darkShade = shadeNum >= 600 ? 400 : 300;
    const darkClass = getNodeIconDarkTextClass(color, darkShade);
    if (darkClass) {
      return `${trimmed} ${darkClass}`;
    }
  }

  return `${trimmed} dark:text-gray-300`;
}

const COLOR_BACKGROUND_CLASSES: Readonly<Record<string, string>> = {
  slate: "bg-slate-100",
  gray: "bg-gray-100",
  zinc: "bg-gray-100",
  neutral: "bg-neutral-100",
  stone: "bg-stone-100",
  red: "bg-red-100",
  orange: "bg-orange-100",
  amber: "bg-amber-100",
  yellow: "bg-yellow-100",
  lime: "bg-lime-100",
  green: "bg-green-100",
  emerald: "bg-emerald-100",
  teal: "bg-teal-100",
  cyan: "bg-cyan-100",
  sky: "bg-sky-100",
  blue: "bg-blue-100",
  indigo: "bg-indigo-100",
  violet: "bg-violet-100",
  purple: "bg-purple-100",
  fuchsia: "bg-fuchsia-100",
  pink: "bg-pink-100",
  rose: "bg-rose-100",
  white: "bg-white",
  black: "bg-black",
};

export const getBackgroundColorClass = (color?: string): string => {
  if (!color || !Object.prototype.hasOwnProperty.call(COLOR_BACKGROUND_CLASSES, color)) {
    return COLOR_BACKGROUND_CLASSES.gray;
  }

  return COLOR_BACKGROUND_CLASSES[color];
};
