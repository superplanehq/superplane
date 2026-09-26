import { buildSubtitle, buildExecutionSubtitle } from "../utils";
import type { OutputPayload } from "../types";

export const buildBitbucketSubtitle = buildSubtitle;
export const buildBitbucketExecutionSubtitle = buildExecutionSubtitle;

/** Shorten a commit hash for display, leaving expression placeholders intact. */
export function shortHash(hash?: string): string {
  if (!hash) {
    return "";
  }

  return hash.includes("{{") ? hash : hash.slice(0, 7);
}

/** Read the first payload emitted on the default output channel. */
export function defaultOutput<T>(outputs: unknown): { data: T; timestamp?: string } | undefined {
  const channels = outputs as { default?: OutputPayload[] } | undefined;
  const payload = channels?.default?.[0];

  if (!payload?.data) {
    return undefined;
  }

  return { data: payload.data as T, timestamp: payload.timestamp };
}

export function addDetailIfPresent(details: Record<string, string>, label: string, value?: string | number) {
  if (value !== undefined && value !== null && value !== "") {
    details[label] = String(value);
  }
}
