import { isReservedAppPathSegment } from "./reservedAppPaths";

const MAX_ORGANIZATION_SLUG_LENGTH = 63;
const NON_SLUG_CHARS = /[^a-z0-9]+/g;

/**
 * Converts a name into a lowercase, URL-friendly slug: lowercases, replaces
 * runs of characters outside [a-z0-9] with a single dash, and trims leading
 * and trailing dashes. Mirrors Slugify in pkg/models/organization_slug.go so
 * the settings form can preview the slug the backend would generate.
 */
export function slugifyOrganizationName(name: string): string {
  const slug = name
    .trim()
    .toLowerCase()
    .replace(NON_SLUG_CHARS, "-")
    .replace(/^-+|-+$/g, "");

  return slug.slice(0, MAX_ORGANIZATION_SLUG_LENGTH).replace(/-+$/, "");
}

/** True when value is already a valid organization slug (see slugifyOrganizationName). */
export function isValidOrganizationSlugFormat(value: string): boolean {
  return value.length > 0 && slugifyOrganizationName(value) === value;
}

export type OrganizationSlugValidationError = "empty" | "invalid-format" | "reserved";

/**
 * Validates a user-entered organization slug client-side, before it is sent
 * to UpdateOrganization. Returns null when the slug is acceptable, or an
 * error code describing why it was rejected. This does not check
 * uniqueness; that is only known by the backend.
 */
export function validateOrganizationSlug(value: string): OrganizationSlugValidationError | null {
  if (!value) {
    return "empty";
  }
  if (!isValidOrganizationSlugFormat(value)) {
    return "invalid-format";
  }
  if (isReservedAppPathSegment(value)) {
    return "reserved";
  }
  return null;
}

export function organizationSlugValidationMessage(error: OrganizationSlugValidationError): string {
  switch (error) {
    case "empty":
      return "Enter an organization slug.";
    case "reserved":
      return "This slug is reserved. Choose a different one.";
    case "invalid-format":
    default:
      return "Use lowercase letters, numbers, and dashes only.";
  }
}
