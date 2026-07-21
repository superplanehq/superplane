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
export function resolveConsoleAvatar(author: unknown, committer?: unknown): ConsoleAvatarDisplay {
  if (typeof author === "string") {
    const fromString = resolveStringAuthor(author.trim());
    if (fromString) return fromString;
  }

  const authorRecord = asRecord(author);
  const committerRecord = asRecord(committer);
  const username = coerceToString(authorRecord?.username).trim();
  const name = coerceToString(authorRecord?.name).trim() || coerceToString(committerRecord?.name).trim() || username;

  // Always resolve initials so callers have a graceful fallback to render when
  // the avatar image URL fails to load (e.g. a bot account with no GitHub
  // avatar), not just when there is no image source to begin with.
  const initials =
    firstInitialFromValues(
      authorRecord?.name,
      committerRecord?.name,
      authorRecord?.username,
      committerRecord?.username,
    ) || undefined;

  if (username) {
    return { src: `https://github.com/${username}.png`, name, initials };
  }

  return { initials, name };
}

/**
 * Resolve a bare string author into an avatar display: an image URL is used as
 * the source verbatim, while a plain username maps to its GitHub avatar with an
 * initials fallback. Returns `null` for an empty string so the caller can fall
 * back to record-based resolution.
 */
function resolveStringAuthor(username: string): ConsoleAvatarDisplay | null {
  if (isImageUrl(username)) {
    return { src: username, name: "" };
  }
  if (username) {
    return {
      src: `https://github.com/${username}.png`,
      name: username,
      initials: initialLetter(username) || undefined,
    };
  }
  return null;
}

function isImageUrl(value: string): boolean {
  return value.startsWith("https://") || value.startsWith("http://");
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
