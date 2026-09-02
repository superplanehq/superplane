/**
 * Copy helper for telling a non-owner who they must contact to purchase
 * hosted credit. Organization admins and viewers can see the credit metrics
 * on Organization Spending, but only an `org_owner` can start checkout, so
 * the page names an owner instead of showing pack buttons.
 */

export interface HostedCreditOwnerContactOwner {
  name?: string;
  email?: string;
}

export interface HostedCreditOwnerContactArgs {
  organizationName?: string;
  owners: HostedCreditOwnerContactOwner[];
}

const GENERIC_CONTACT_MESSAGE = "Contact an organization owner to purchase hosted credit.";

/**
 * Builds the sentence shown to admins and viewers in place of the hosted
 * credit checkout packs. Each owner label prefers a display name and falls
 * back to an email address; owners with neither are skipped. When no owner
 * has a usable label, the copy falls back to a generic sentence rather than
 * naming the organization with an empty owner list.
 */
export function hostedCreditOwnerContactCopy({ organizationName, owners }: HostedCreditOwnerContactArgs): string {
  const ownerLabels = owners
    .map((owner) => owner.name?.trim() || owner.email?.trim())
    .filter((label): label is string => Boolean(label));

  if (ownerLabels.length === 0) {
    return GENERIC_CONTACT_MESSAGE;
  }

  const trimmedOrganizationName = organizationName?.trim();
  const ownerNoun = trimmedOrganizationName ? `${trimmedOrganizationName} owner` : "organization owner";
  const article = ownerLabels.length === 1 ? "the" : "an";

  return `Contact ${article} ${ownerNoun} (${ownerLabels.join(", ")}) to purchase hosted credit.`;
}
