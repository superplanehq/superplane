/**
 * Deep link to the organization Members page with a row highlight for the given user id.
 */
export function membersHighlightHref(organizationId: string, userId: string): string {
  const params = new URLSearchParams({ highlightUserId: userId });
  return `/${organizationId}/settings/members?${params.toString()}`;
}
