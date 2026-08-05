/**
 * Guard for user-supplied URLs that will end up in an `href`. Returns the
 * trimmed URL when it parses as an absolute `http(s)` URL with a host, and
 * `null` otherwise, so callers can render a non-link fallback instead of
 * exposing a `javascript:` / `data:` / protocol-relative URL to teammates who
 * click it.
 *
 * Server-side validation is the primary defense (see
 * `models.isSafeArtifactURL`); this helper is defense in depth for anything
 * that reached the client before that validation existed, or that the client
 * synthesizes locally (e.g. optimistic UI).
 */
export function safeExternalUrl(raw: string | null | undefined): string | null {
  if (!raw) return null;
  const trimmed = raw.trim();
  if (!trimmed) return null;

  // Reject protocol-relative URLs (`//evil.example/x`) explicitly; the URL
  // constructor treats them as valid when a base is supplied but they lose
  // their scheme when rendered in an `href`.
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
