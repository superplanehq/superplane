/**
 * Utilities for the Jira-style workspace key attached to every factory.
 *
 * A key is 2-5 lowercase Latin letters, unique per organization, and drives
 * the human-readable identifier for every task that belongs to the
 * workspace: `${key}-${number}` (for example, `sp-1`).
 *
 * Existing workspaces created before this rule may still carry an
 * uppercase key (the backend does not rewrite them) — `isValidWorkspaceKey`
 * only governs new/edited keys. Use `isWorkspaceKeyShaped` for case-insensitive
 * checks like matching a `:factoryKey` route segment.
 *
 * Backend validation is authoritative — these helpers only shape the input
 * the user typed and mirror the backend regex so the form can offer inline
 * feedback before the request lands.
 */

export const WORKSPACE_KEY_MIN_LENGTH = 2;
export const WORKSPACE_KEY_MAX_LENGTH = 5;

/** Seed used when a name has no usable letters (mirrors the backend default). */
const DEFAULT_WORKSPACE_KEY_SEED = "ws";

const KEY_PATTERN = /^[a-z]{2,5}$/;
const KEY_SHAPE_PATTERN = /^[A-Za-z]{2,5}$/;

/**
 * Turn free-form input (from typing or paste) into the canonical form
 * the backend expects: lowercase letters only, no spaces, capped at the
 * max length. Called from every onChange handler so users cannot enter a
 * value the server will reject.
 */
export function normalizeWorkspaceKey(input: string): string {
  return input
    .toLowerCase()
    .replace(/[^a-z]/g, "")
    .slice(0, WORKSPACE_KEY_MAX_LENGTH);
}

/** Strict check for a new/edited key: 2-5 lowercase letters. */
export function isValidWorkspaceKey(value: string): boolean {
  return KEY_PATTERN.test(value);
}

/**
 * Loose, case-insensitive shape check for a key regardless of when it was
 * created. Route matching needs this because legacy workspaces can still
 * carry an uppercase key.
 */
export function isWorkspaceKeyShaped(value: string): boolean {
  return KEY_SHAPE_PATTERN.test(value);
}

/**
 * Derive a key candidate from a workspace name. We take the leading
 * letters after normalizing so the auto-generated key looks predictable
 * even for punctuated or accented names.
 */
export function suggestWorkspaceKeyFromName(name: string): string {
  const letters = name.toLowerCase().replace(/[^a-z]/g, "");
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
 * Derives a key from a workspace/repository name and walks to a variant
 * that is not already used by another workspace in the organization. Used
 * to regenerate the onboarding slug once the real name is known, so it no
 * longer stays the placeholder-derived value (`new`, `neww`, ...).
 *
 * Mirrors (a simplified, client-side version of) the backend's
 * `GenerateUniqueFactoryKey` collision walk: keep the seed's leading
 * letters and cycle the last character through the alphabet.
 */
export function uniqueWorkspaceKeyFromName(name: string, takenKeys: Iterable<string>): string {
  const taken = new Set([...takenKeys].map((key) => key.trim().toLowerCase()).filter(Boolean));
  const seed = suggestWorkspaceKeyFromName(name) || DEFAULT_WORKSPACE_KEY_SEED;
  if (!taken.has(seed)) {
    return seed;
  }

  const prefix = seed.slice(0, WORKSPACE_KEY_MAX_LENGTH - 1);
  for (let code = "a".charCodeAt(0); code <= "z".charCodeAt(0); code += 1) {
    const candidate = `${prefix}${String.fromCharCode(code)}`;
    if (!taken.has(candidate)) {
      return candidate;
    }
  }
  return `${prefix}z`;
}

/**
 * Render the display identifier for a task (`sp-42`). Returns an
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
