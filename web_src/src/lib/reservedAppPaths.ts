// Keep this list in sync with reservedOrganizationSlugs in
// pkg/models/organization_slug.go. It combines the frontend's own top-level
// routes with infrastructure roots the server handles outside the SPA, so an
// organization slug can never shadow either kind of route.
const RESERVED_APP_PATH_SEGMENTS = new Set([
  // Frontend application routes.
  "admin",
  "login",
  "signup",
  "welcome",
  "create",
  "setup",
  "invite",
  "install",
  // Infrastructure roots served outside the SPA.
  "api",
  "health",
  "assets",
  "logout",
]);

/** True when a `/:organizationId` segment is actually a top-level app route. */
export function isReservedAppPathSegment(segment: string | undefined): boolean {
  if (!segment) {
    return false;
  }
  return RESERVED_APP_PATH_SEGMENTS.has(segment);
}
