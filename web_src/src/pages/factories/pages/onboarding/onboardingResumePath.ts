import type { FactoriesFactory } from "@/api-client";
import { GITHUB_SETUP_ORG_PARAM, GITHUB_SETUP_REQUEST_PARAM } from "@/lib/integrationSetupReturn";

import { factorySetupPath } from "../../lib/factoryPagePaths";
import { isFactoryOnboardingComplete } from "./onboardingStatus";

const RESUME_SEARCH_PARAMS = ["step", "pick", GITHUB_SETUP_REQUEST_PARAM, GITHUB_SETUP_ORG_PARAM] as const;

/**
 * Maps the first-run `/onboarding?...` URL to the org-scoped setup path that
 * last-location can store and restore. Query params that identify the wizard
 * step and a pending GitHub install request are kept.
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
