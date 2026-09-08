import type { FactoriesFactory } from "@/api-client";

/**
 * The previous GitHub connection from initial org setup, when no other
 * workspace still uses it. Workspace onboarding never returns an id.
 */
export function unusedOnboardingVcsIntegrationId(args: {
  isInitial: boolean;
  previousId: string | undefined;
  nextId: string;
  factories: FactoriesFactory[];
  currentFactoryId: string;
}): string | undefined {
  if (!args.isInitial || !args.previousId || args.previousId === args.nextId) {
    return undefined;
  }

  const usedElsewhere = args.factories.some(
    (factory) => factory.id !== args.currentFactoryId && factory.onboarding?.vcsIntegrationId === args.previousId,
  );
  if (usedElsewhere) {
    return undefined;
  }

  return args.previousId;
}
