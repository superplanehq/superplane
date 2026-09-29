import type { FactoriesFactory } from "@/api-client";

import { factorySetupPath } from "../../lib/factoryPagePaths";
import { isFactoryOnboardingComplete } from "./onboardingStatus";

const RESUME_SEARCH_PARAMS = ["step"] as const;

/**
 * Maps the first-run `/onboarding?...` URL to the org-scoped setup path that
 * last-location can store and restore. Only the current wizard step is kept.
 */
export function onboardingResumePath(organizationSlug: string, factoryKey: string, search: string): string {
  const setupPath = factorySetupPath(organizationSlug, factoryKey);
  const incoming = new URLSearchParams(search.startsWith("?") ? search.slice(1) : search);
  const next = new URLSearchParams();

  for (const key of RESUME_SEARCH_PARAMS) {
    const value = incoming.get(key);
    if (value) {
      next.set(key, value);
    }
  }

  const query = next.toString();
  return query ? `${setupPath}?${query}` : setupPath;
}

/** First incomplete workspace setup path, or null when every workspace is complete. */
export function incompleteWorkspaceSetupPath(
  organizationSlug: string,
  factories: FactoriesFactory[] | undefined,
): string | null {
  const factory = factories?.find((item) => item.key && !isFactoryOnboardingComplete(item));
  if (!factory?.key) {
    return null;
  }

  return factorySetupPath(organizationSlug, factory.key);
}
