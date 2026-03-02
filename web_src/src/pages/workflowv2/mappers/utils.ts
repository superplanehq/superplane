import { DEFAULT_EVENT_STATE_MAP } from "@/ui/componentBase";
import { formatTimeAgo } from "@/utils/date";
import { EventStateRegistry } from "./types";
import { defaultStateFunction } from "./stateRegistry";

/*
 *
 * Formats a number of bytes to MB.
 *
 * @param value - The number of bytes to format.
 * @returns The formatted number of MB.
 */
export function formatBytes(value?: number): string {
  if (value === undefined || value === null) {
    return "-";
  }

  // convert to MB
  const mb = value / 1000 / 1000;
  return `${mb.toFixed(2)} MB`;
}

/*
 *
 * Returns a number or 0 if the value is undefined or null.
 *
 * @param value - The number to return.
 * @returns The number or 0.
 */
export function numberOrZero(value?: number): number {
  if (value === undefined || value === null) {
    return 0;
  }

  return value;
}

/*
 *
 * Returns a string or "-" if the value is undefined, null, or an empty string.
 *
 * @param value - The string to return.
 * @returns The string or "-".
 */
export function stringOrDash(value?: unknown): string {
  if (value === undefined || value === null || value === "") {
    return "-";
  }

  return String(value);
}

/*
 *
 * Builds an action state registry.
 *
 * @param successState - The state to return when the action is successful.
 * @returns The action state registry.
 */
export function buildActionStateRegistry(successState: string): EventStateRegistry {
  return {
    stateMap: {
      ...DEFAULT_EVENT_STATE_MAP,
      [successState]: DEFAULT_EVENT_STATE_MAP.success,
    },
    getState: (execution) => {
      const state = defaultStateFunction(execution);
      return state === "success" ? successState : state;
    },
  };
}

/*
 * Predicate type and format function.
 * See: AnyPredicateListFieldRenderer.
 *
 * @param predicate - The predicate to format.
 * @returns The formatted predicate.
 */
export type PredicateType = "equals" | "notEquals" | "matches";

export interface Predicate {
  type: PredicateType;
  value: string;
}

export function formatPredicate(predicate: Predicate): string {
  switch (predicate.type) {
    case "equals":
      return `=${predicate.value}`;
    case "notEquals":
      return `!=${predicate.value}`;
    case "matches":
      return `~${predicate.value}`;
    default:
      return predicate.value;
  }
}

export function buildSubtitle(content: string | undefined, createdAt?: string): string {
  const trimmed = (content || "").trim();
  const timeAgo = createdAt ? formatTimeAgo(new Date(createdAt)) : "";

  if (trimmed && timeAgo) {
    return `${trimmed} · ${timeAgo}`;
  }
  return trimmed || timeAgo;
}

export interface ExecutionLike {
  createdAt: string;
  updatedAt?: string;
}

export function buildExecutionSubtitle(execution: ExecutionLike, content?: string): string {
  const timestamp = execution.updatedAt || execution.createdAt;
  return buildSubtitle(content || "", timestamp);
}

export function formatTimestamp(value?: string, fallback?: string): string {
  const timestamp = value || fallback;
  if (!timestamp) {
    return "-";
  }
  const date = new Date(timestamp);
  if (Number.isNaN(date.getTime())) {
    return "-";
  }
  return date.toLocaleString();
}
