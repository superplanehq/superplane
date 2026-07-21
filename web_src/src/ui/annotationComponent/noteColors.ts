export type AnnotationColor = "yellow" | "blue" | "green" | "purple";

export const NOTE_COLORS: Record<
  AnnotationColor,
  { label: string; container: string; background: string; dot: string }
> = {
  yellow: {
    label: "Yellow",
    container: "bg-yellow-100 dark:bg-yellow-900 dark:outline dark:outline-1 dark:outline-yellow-950/80",
    background: "bg-yellow-100 dark:bg-yellow-900",
    dot: "bg-yellow-200 border-yellow-500 dark:bg-yellow-800 dark:border-yellow-500",
  },
  blue: {
    label: "Sky",
    container: "bg-sky-100 dark:bg-sky-900 dark:outline dark:outline-1 dark:outline-sky-950/80",
    background: "bg-sky-100 dark:bg-sky-900",
    dot: "bg-sky-200 border-sky-500 dark:bg-sky-800 dark:border-sky-500",
  },
  green: {
    label: "Green",
    container: "bg-green-100 dark:bg-green-900 dark:outline dark:outline-1 dark:outline-green-950/80",
    background: "bg-green-100 dark:bg-green-900",
    dot: "bg-green-200 border-green-500 dark:bg-green-800 dark:border-green-500",
  },
  purple: {
    label: "Purple",
    container: "bg-purple-100 dark:bg-purple-900 dark:outline dark:outline-1 dark:outline-purple-950/80",
    background: "bg-purple-100 dark:bg-purple-900",
    dot: "bg-purple-200 border-purple-500 dark:bg-purple-800 dark:border-purple-500",
  },
};
