const RESERVED_APP_PATH_SEGMENTS = new Set([
  "admin",
  "login",
  "signup",
  "welcome",
  "create",
  "setup",
  "invite",
  "install",
]);

/** True when a `/:organizationId` segment is actually a top-level app route. */
export function isReservedAppPathSegment(segment: string | undefined): boolean {
  if (!segment) {
    return false;
  }
  return RESERVED_APP_PATH_SEGMENTS.has(segment);
}
