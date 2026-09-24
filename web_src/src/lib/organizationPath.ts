/**
 * Options for {@link resolveOrganizationUidRedirect}.
 */
interface ResolveOrganizationUidRedirectOptions {
  /** The current `location.pathname`, e.g. `/{segment}` or `/{segment}/settings`. */
  pathname: string;
  /** The current `location.search`, preserved as-is (e.g. `?run=123`). */
  search?: string;
  /** The current `location.hash`, preserved as-is. */
  hash?: string;
  /** The first path segment as it currently appears in the URL. */
  segment: string;
  /** The organization's UID, as resolved from the backend. */
  organizationId: string;
  /** The organization's slug, as resolved from the backend. May be empty. */
  organizationSlug: string;
}

/**
 * Computes the path to redirect to when an organization-scoped URL uses the
 * organization's UID as its first segment instead of its slug.
 *
 * Returns `null` when no redirect is needed: the segment already matches the
 * slug, the organization has no slug yet, or the segment isn't the org's UID
 * at all (e.g. it's already a valid, unrelated slug).
 *
 * The rest of the path, the query string, and the hash are preserved so deep
 * links (e.g. `?run=123`) keep working after the swap.
 */
export function resolveOrganizationUidRedirect({
  pathname,
  search = "",
  hash = "",
  segment,
  organizationId,
  organizationSlug,
}: ResolveOrganizationUidRedirectOptions): string | null {
  if (!segment || !organizationId || !organizationSlug) {
    return null;
  }

  if (segment !== organizationId || segment === organizationSlug) {
    return null;
  }

  if (!pathname.startsWith(`/${segment}`)) {
    return null;
  }

  const rest = pathname.slice(segment.length + 1);
  return `/${organizationSlug}${rest}${search}${hash}`;
}
