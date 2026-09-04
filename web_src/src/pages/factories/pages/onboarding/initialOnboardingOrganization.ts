import type { FactoriesFactory, OrganizationsIntegration } from "@/api-client";
import { getApiErrorMessage } from "@/lib/errors";

const ORGANIZATION_SUFFIX_ALPHABET = "abcdefghijklmnopqrstuvwxyz0123456789";

export function githubIntegrationOwner(integration: OrganizationsIntegration): string | undefined {
  const owner = integration.status?.metadata?.owner;
  return typeof owner === "string" && owner.trim() ? owner.trim() : undefined;
}

export function shouldNameOrganizationFromGitHub(factory: FactoriesFactory | null, selectNewest: boolean): boolean {
  return selectNewest && factory?.onboarding?.initial === true;
}

export function randomOrganizationSuffix(): string {
  const bytes = new Uint8Array(6);
  crypto.getRandomValues(bytes);
  return Array.from(bytes, (byte) => ORGANIZATION_SUFFIX_ALPHABET[byte % ORGANIZATION_SUFFIX_ALPHABET.length]).join("");
}

export function organizationIdentityFromOwner(owner: string, suffix?: string): { name: string; slug: string } {
  const base = owner.trim();
  const slugBase = slugifyOrganizationOwner(base);
  if (!suffix) {
    return { name: base, slug: slugBase };
  }
  return { name: `${base}-${suffix}`, slug: `${slugBase}-${suffix}` };
}

export function isOrganizationIdentityTaken(message: string): boolean {
  const normalized = message.toLowerCase();
  return normalized.includes("already in use") || normalized.includes("already used");
}

export async function nameOrganizationFromGitHubOwner(args: {
  owner: string;
  currentSlug: string;
  update: (identity: { name: string; slug: string }) => Promise<string | undefined>;
  randomSuffix?: () => string;
}): Promise<string | undefined> {
  const nextSuffix = args.randomSuffix ?? randomOrganizationSuffix;
  let identity = organizationIdentityFromOwner(args.owner);
  if (identity.slug === args.currentSlug) {
    return args.currentSlug;
  }

  for (let attempt = 0; attempt < 5; attempt++) {
    try {
      return await args.update(identity);
    } catch (error) {
      if (!isOrganizationIdentityTaken(getApiErrorMessage(error, "")) || attempt === 4) {
        throw error;
      }
      identity = organizationIdentityFromOwner(args.owner, nextSuffix());
    }
  }

  return undefined;
}

function slugifyOrganizationOwner(owner: string): string {
  const slug = owner
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
  return slug || "org";
}
