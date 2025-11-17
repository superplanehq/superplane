export const getColorClass = (color?: string): string => {
  switch (color) {
    case "slate":
      return "text-slate-600 dark:text-slate-400";
    case "gray":
      return "text-gray-600 dark:text-gray-400";
    case "zinc":
      return "text-zinc-600 dark:text-zinc-400";
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
      return "text-black dark:text-black";
    default:
      return "text-gray-600 dark:text-gray-400";
  }
};

export const getBackgroundColorClass = (color?: string): string => {
  switch (color) {
    case "slate":
      return "bg-slate-100";
    case "gray":
      return "bg-gray-100";
    case "zinc":
      return "bg-zinc-100";
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
