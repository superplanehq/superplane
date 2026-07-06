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

export const getColorClass = (color?: string): string => {
  switch (color) {
    case "slate":
      return "text-slate-600 dark:text-slate-400";
    case "gray":
      return "text-gray-500 dark:text-gray-400";
    case "zinc":
      return "text-gray-500 dark:text-gray-400";
    case "neutral":
      return "text-neutral-600 dark:text-neutral-400";
    case "stone":
      return "text-stone-600 dark:text-stone-400";
    case "red":
      return "text-red-600 dark:text-red-400";
    case "orange":
      return "text-orange-600 dark:text-orange-400";
    case "amber":
      return "text-amber-600 dark:text-amber-400";
    case "yellow":
      return "text-yellow-600 dark:text-yellow-400";
    case "lime":
      return "text-lime-600 dark:text-lime-400";
    case "green":
      return "text-green-600 dark:text-green-400";
    case "emerald":
      return "text-emerald-600 dark:text-emerald-400";
    case "teal":
      return "text-teal-600 dark:text-teal-400";
    case "cyan":
      return "text-cyan-600 dark:text-cyan-400";
    case "sky":
      return "text-sky-600 dark:text-sky-400";
    case "blue":
      return "text-blue-600 dark:text-blue-400";
    case "indigo":
      return "text-indigo-600 dark:text-indigo-400";
    case "violet":
      return "text-violet-600 dark:text-violet-400";
    case "purple":
      return "text-purple-600 dark:text-purple-400";
    case "fuchsia":
      return "text-fuchsia-600 dark:text-fuchsia-400";
    case "pink":
      return "text-pink-600 dark:text-pink-400";
    case "rose":
      return "text-rose-600 dark:text-rose-400";
    case "white":
      return "text-white dark:text-white";
    case "black":
      return "text-gray-900 dark:text-gray-300";
    default:
      return "text-gray-500 dark:text-gray-400";
  }
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

export const getBackgroundColorClass = (color?: string): string => {
  switch (color) {
    case "slate":
      return "bg-slate-100";
    case "gray":
      return "bg-gray-100";
    case "zinc":
      return "bg-gray-100";
    case "neutral":
      return "bg-neutral-100";
    case "stone":
      return "bg-stone-100";
    case "red":
      return "bg-red-100";
    case "orange":
      return "bg-orange-100";
    case "amber":
      return "bg-amber-100";
    case "yellow":
      return "bg-yellow-100";
    case "lime":
      return "bg-lime-100";
    case "green":
      return "bg-green-100";
    case "emerald":
      return "bg-emerald-100";
    case "teal":
      return "bg-teal-100";
    case "cyan":
      return "bg-cyan-100";
    case "sky":
      return "bg-sky-100";
    case "blue":
      return "bg-blue-100";
    case "indigo":
      return "bg-indigo-100";
    case "violet":
      return "bg-violet-100";
    case "purple":
      return "bg-purple-100";
    case "fuchsia":
      return "bg-fuchsia-100";
    case "pink":
      return "bg-pink-100";
    case "rose":
      return "bg-rose-100";
    case "white":
      return "bg-white";
    case "black":
      return "bg-black";
    default:
      return "bg-gray-100";
  }
};
