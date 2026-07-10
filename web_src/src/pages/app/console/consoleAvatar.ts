export interface ConsoleAvatarDisplay {
  src?: string;
  initials?: string;
  name: string;
}

/**
 * Resolve avatar image / initials for GitHub webhook author maps. Mirrors the
 * `githubAvatarOrInitial` CEL helper used in HTML panels.
 */
export function resolveConsoleAvatar(author: unknown, committer?: unknown): ConsoleAvatarDisplay {
  if (typeof author === "string") {
    const username = author.trim();
    if (username) {
      return { src: `https://github.com/${username}.png`, name: username };
    }
  }

  const authorRecord = asRecord(author);
  const committerRecord = asRecord(committer);
  const username = coerceToString(authorRecord?.username).trim();
  const name = coerceToString(authorRecord?.name).trim() || coerceToString(committerRecord?.name).trim() || username;

  if (username) {
    return { src: `https://github.com/${username}.png`, name };
  }

  const initials = firstInitialFromValues(
    authorRecord?.name,
    committerRecord?.name,
    authorRecord?.username,
    committerRecord?.username,
  );

  return { initials: initials || undefined, name };
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
