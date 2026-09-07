// A "safe" redirect path is relative to the SuperPlane origin, so following
// it can never send a user to another site. Two checks guard that:
// a leading "/" (so it is not an absolute URL like "https://evil.com"), and
// no second leading "/" or "\" (so it is not a protocol-relative URL like
// "//evil.com", which browsers treat as an absolute URL to another host).
export function isSafeRedirectPath(path: string | null | undefined): path is string {
  if (!path || path[0] !== "/") {
    return false;
  }

  if (path.length > 1 && (path[1] === "/" || path[1] === "\\")) {
    return false;
  }

  return true;
}

/** Decodes and validates a redirect path pulled from a URL query parameter. */
export function getSafeRedirectPath(rawRedirect: string | null | undefined): string | null {
  if (!rawRedirect) {
    return null;
  }

  try {
    const decoded = decodeURIComponent(rawRedirect);
    return isSafeRedirectPath(decoded) ? decoded : null;
  } catch {
    return null;
  }
}
