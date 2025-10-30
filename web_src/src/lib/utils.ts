import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"
import { BookMarked, type LucideIcon } from "lucide-react"
import * as LucideIcons from "lucide-react"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export const resolveIcon = (slug?: string): LucideIcon => {
  if (!slug) {
    return BookMarked
  }

  const pascalCase = slug
    .split("-")
    .map((segment) => segment.charAt(0).toUpperCase() + segment.slice(1))
    .join("")

  const candidate = (LucideIcons as Record<string, unknown>)[pascalCase]

  if (
    candidate &&
    (typeof candidate === "function" ||
      (typeof candidate === "object" && "render" in candidate))
  ) {
    return candidate as LucideIcon
  }

  return BookMarked
}

export const calcRelativeTimeFromDiff = (diff: number) => {
  const seconds = Math.floor(diff / 1000)
  const minutes = Math.floor(seconds / 60)
  const hours = Math.floor(minutes / 60)
  const days = Math.floor(hours / 24)
  if (days > 0) {
    return `${days}d`
  } else if (hours > 0) {
    return `${hours}h`
  } else if (minutes > 0) {
    return `${minutes}m`
  } else {
    return `${seconds}s`
  }
}

export function splitBySpaces(input: string): string[] {
  const regex = /(?:[^\s"']+|"(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*')+/g;
  const matches = input.match(regex);
  return matches || [];
}