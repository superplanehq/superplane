import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";
import { Puzzle, type LucideIcon } from "lucide-react";
import * as LucideIcons from "lucide-react";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export const resolveIcon = (slug?: string): LucideIcon => {
  if (!slug) {
    return Puzzle;
  }

  const normalized = slug.toLowerCase();
  const aliases: Record<string, string> = {
    close: "X",
    "x-mark": "X",
    xmark: "X",
  };
  const alias = aliases[normalized];
  if (alias && (LucideIcons as Record<string, unknown>)[alias]) {
    return (LucideIcons as Record<string, unknown>)[alias] as LucideIcon;
  }

  const pascalCase = slug
    .split("-")
    .map((segment) => segment.charAt(0).toUpperCase() + segment.slice(1))
    .join("");

  const candidate = (LucideIcons as Record<string, unknown>)[pascalCase];

  if (candidate && (typeof candidate === "function" || (typeof candidate === "object" && "render" in candidate))) {
    return candidate as LucideIcon;
  }

  return Puzzle;
};

export const calcRelativeTimeFromDiff = (diff: number) => {
  const seconds = Math.floor(diff / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);
  if (days > 0) {
    return `${days}d`;
  } else if (hours > 0) {
    return `${hours}h`;
  } else if (minutes > 0) {
    return `${minutes}m`;
  } else {
    return `${seconds}s`;
  }
};

export const formatDuration = (value: number, unit: string): string => {
  const unitLabels: Record<string, string> = {
    seconds: value === 1 ? "second" : "seconds",
    minutes: value === 1 ? "minute" : "minutes",
    hours: value === 1 ? "hour" : "hours",
  };
  return `${value} ${unitLabels[unit] || unit}`;
};

export const formatTimestamp = (date: Date): string => {
  return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
};

export function splitBySpaces(input: string): string[] {
  const regex = /(?:[^\s"']+|"(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*')+/g;
  const matches = input.match(regex);
  return matches || [];
}

/**
 * Flattens a nested object structure by extracting values from arrays and nested objects.
 * This is particularly useful for extracting data from complex API responses.
 *
 * @param obj - The object to flatten
 * @param maxDepth - Maximum recursion depth to prevent infinite loops (default: 5)
 * @returns A flattened object with primitive values
 */
export function flattenObject(obj: any, maxDepth: number = 5): Record<string, any> {
  if (maxDepth <= 0 || obj === null || obj === undefined) {
    return {};
  }

  function flatten(current: any, depth: number): Record<string, any> {
    if (depth <= 0 || current === null || current === undefined) {
      return {};
    }

    const flatResult: Record<string, any> = {};

    if (Array.isArray(current)) {
      // For arrays, flatten each element and merge results
      current.forEach((item, index) => {
        if (typeof item === "object" && item !== null) {
          const flattened = flatten(item, depth - 1);
          Object.assign(flatResult, flattened);
        } else if (item !== null && item !== undefined) {
          flatResult[`item_${index}`] = item;
        }
      });
    } else if (typeof current === "object") {
      // For objects, recursively flatten
      for (const [key, value] of Object.entries(current)) {
        if (value === null || value === undefined) {
          continue;
        }

        if (typeof value === "object") {
          if (Array.isArray(value)) {
            // Handle arrays
            value.forEach((item, index) => {
              if (typeof item === "object" && item !== null) {
                const flattened = flatten(item, depth - 1);
                Object.assign(flatResult, flattened);
              } else if (item !== null && item !== undefined) {
                flatResult[`${key}_${index}`] = item;
              }
            });
          } else {
            // Handle nested objects
            const flattened = flatten(value, depth - 1);
            Object.assign(flatResult, flattened);
          }
        } else {
          // Handle primitive values
          flatResult[key] = value;
        }
      }
    } else {
      // Handle primitive values at root level
      return { value: current };
    }

    return flatResult;
  }

  return flatten(obj, maxDepth);
}

/**
 * Checks if a string is a valid HTTP or HTTPS URL
 * @param value - The string to check
 * @returns true if the string is a valid URL, false otherwise
 */
export function isUrl(value: string): boolean {
  try {
    const url = new URL(value);
    return url.protocol === "http:" || url.protocol === "https:";
  } catch {
    return false;
  }
}

export const isUUID = (value: string): boolean => {
  return /^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$/.test(value);
};
