/**
 * Guard for user-supplied URLs bound for an `href`: returns the URL if it
 * parses as an absolute http(s) URL with a host, else null. Defense in depth
 * alongside server-side validation (see `models.isSafeArtifactURL`).
 */
export function safeExternalUrl(raw: string | null | undefined): string | null {
  if (!raw) return null;
  const trimmed = raw.trim();
  if (!trimmed) return null;

  // `URL` accepts `//evil.example/x` with a base and drops the scheme.
  if (trimmed.startsWith("//") || trimmed.startsWith("\\\\")) {
    return null;
  }

  let parsed: URL;
  try {
    parsed = new URL(trimmed);
  } catch {
    return null;
  }

  const scheme = parsed.protocol.toLowerCase();
  if (scheme !== "http:" && scheme !== "https:") {
    return null;
  }

  if (!parsed.host) {
    return null;
  }

  return parsed.toString();
}
