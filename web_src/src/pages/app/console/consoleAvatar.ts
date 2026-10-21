import { githubPersonAvatarUrl } from "@/lib/githubAvatar";

export interface ConsoleAvatarDisplay {
  src?: string;
  initials?: string;
  name: string;
}

/**
 * Resolve avatar image / initials for GitHub webhook author maps. Mirrors the
 * `githubAvatarOrInitial` CEL helper used in HTML panels. Strings that are
 * already image URLs (e.g. an `avatar_url` field or a `{{ cel }}` expression
 * that builds one) are passed through as the image source directly.
 */
export function resolveConsoleAvatar(author: unknown, committer?: unknown, host?: string | null): ConsoleAvatarDisplay {
  const fromString = avatarFromUsernameString(author, host);
  if (fromString) {
    return fromString;
  }

  const authorRecord = asRecord(author);
  const committerRecord = asRecord(committer);
  const username = coerceToString(authorRecord?.username).trim();
  const name = coerceToString(authorRecord?.name).trim() || coerceToString(committerRecord?.name).trim() || username;
  const src = githubPersonAvatarUrl(author, host);
  if (src) {
    return { src, name };
  }

  const initials = firstInitialFromValues(
    authorRecord?.name,
    committerRecord?.name,
    authorRecord?.username,
    committerRecord?.username,
  );
  return { initials: initials || undefined, name };
}

function avatarFromUsernameString(author: unknown, host?: string | null): ConsoleAvatarDisplay | undefined {
  if (typeof author !== "string") {
    return undefined;
  }
  const username = author.trim();
  const src = githubPersonAvatarUrl(username, host);
  if (!src) {
    return undefined;
  }
  if (username.startsWith("https://") || username.startsWith("http://")) {
    return { src, name: "" };
  }
  return { src, name: username };
}

function asRecord(value: unknown): Record<string, unknown> | null {
  if (!value || typeof value !== "object" || Array.isArray(value)) return null;
  return value as Record<string, unknown>;
}

function coerceToString(value: unknown): string {
  if (value === null || value === undefined) return "";
  if (typeof value === "string") return value;
  return String(value);
}

function initialLetter(value: unknown): string {
  const text = coerceToString(value).trim();
  if (text === "") return "";
  const match = text.match(/[A-Za-z0-9]/);
  return match ? match[0].toUpperCase() : text.charAt(0).toUpperCase();
}

function firstInitialFromValues(a: unknown, b?: unknown, c?: unknown, d?: unknown): string {
  for (const candidate of [a, b, c, d]) {
    if (candidate === undefined) continue;
    const letter = initialLetter(candidate);
    if (letter) return letter;
  }
  return "";
}
