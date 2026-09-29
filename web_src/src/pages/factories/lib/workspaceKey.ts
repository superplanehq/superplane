/**
 * Utilities for the Jira-style workspace key attached to every factory.
 *
 * A key is 2-5 uppercase Latin letters, unique per organization, and drives
 * the human-readable identifier for every task that belongs to the
 * workspace: `${key}-${number}` (for example, `SP-1`).
 *
 * Backend validation is authoritative — these helpers only shape the input
 * the user typed and mirror the backend regex so the form can offer inline
 * feedback before the request lands.
 */

export const WORKSPACE_KEY_MIN_LENGTH = 2;
export const WORKSPACE_KEY_MAX_LENGTH = 5;

const KEY_PATTERN = /^[A-Z]{2,5}$/;

/**
 * Turn free-form input (from typing or paste) into the canonical form
 * the backend expects: uppercase letters only, no spaces, capped at the
 * max length. Called from every onChange handler so users cannot enter a
 * value the server will reject.
 */
export function normalizeWorkspaceKey(input: string): string {
  return input
    .toUpperCase()
    .replace(/[^A-Z]/g, "")
    .slice(0, WORKSPACE_KEY_MAX_LENGTH);
}

export function isValidWorkspaceKey(value: string): boolean {
  return KEY_PATTERN.test(value);
}

/**
 * Derive a key candidate from a workspace name. We take the leading
 * letters after normalizing so the auto-generated key looks predictable
 * even for punctuated or accented names.
 */
export function suggestWorkspaceKeyFromName(name: string): string {
  const letters = name.toUpperCase().replace(/[^A-Z]/g, "");
  if (letters.length === 0) {
    return "";
  }
  const trimmed = letters.slice(0, WORKSPACE_KEY_MAX_LENGTH);
  if (trimmed.length < WORKSPACE_KEY_MIN_LENGTH) {
    return "";
  }
  return trimmed;
}

/**
 * Render the display identifier for a task (`SP-42`). Returns an
 * empty string when either the key or the number is missing so callers
 * can fall back to whatever short identifier they already show.
 *
 * Accepts `number` as either a number or the JSON-encoded int64 string
 * the generated API client returns.
 */
export function formatWorkOrderIdentifier(
  key: string | undefined | null,
  number: string | number | undefined | null,
): string {
  if (!key || number === undefined || number === null || number === "") {
    return "";
  }
  const parsed = typeof number === "number" ? number : Number(number);
  if (!Number.isFinite(parsed) || parsed <= 0) {
    return "";
  }
  return `${key}-${parsed}`;
}
