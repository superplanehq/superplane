import { DEFAULT_EVENT_STATE_MAP } from "@/ui/componentBase";
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
