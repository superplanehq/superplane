import { githubWebHostFromUrl, githubWebOrigin } from "./githubHost";
import { safeExternalUrl } from "./safeExternalUrl";

export function githubAvatarImageUrl(login: string, host?: string | null): string {
  return `${githubWebOrigin(host)}/${login.trim()}.png`;
}

/** Image URL for a GitHub user map, or a username / image URL string. */
export function githubPersonAvatarUrl(person: unknown, host?: string | null): string | undefined {
  if (typeof person === "string") {
    const value = person.trim();
    if (!value) {
      return undefined;
    }
    return safeExternalUrl(value) ?? githubAvatarImageUrl(value, host);
  }
  const record = asRecord(person);
  if (!record) {
    return undefined;
  }
  const avatarUrl = safeExternalUrl(coerceToString(record.avatar_url));
  if (avatarUrl) {
    return avatarUrl;
  }
  const username = coerceToString(record.username) || coerceToString(record.login);
  if (!username) {
    return undefined;
  }
  return githubAvatarImageUrl(username, githubWebHostFromUrl(coerceToString(record.html_url)) || host);
}

function asRecord(value: unknown): Record<string, unknown> | null {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return null;
  }
  return value as Record<string, unknown>;
}

function coerceToString(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}
