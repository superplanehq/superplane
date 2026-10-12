export const bitbucketOnboardingPollIntervalMs = 3_000;
export const bitbucketInstallationAttemptMs = 5 * 60_000;

export type BitbucketOnboardingPollState = {
  status?: string;
  data?: {
    providerConfigured?: boolean;
    identity?: unknown;
    repositories?: unknown[];
  };
};

export function bitbucketOnboardingPollInterval(
  poll: boolean,
  state: BitbucketOnboardingPollState,
  attemptActive = false,
): number | false {
  if (!poll) return false;
  // An installation attempt keeps polling every three seconds for up to five
  // minutes, even when other repositories already exist.
  if (attemptActive) return bitbucketOnboardingPollIntervalMs;
  if (state.status === "error") return bitbucketOnboardingPollIntervalMs;
  const data = state.data;
  if (!data?.providerConfigured || !data.identity) return false;
  if ((data.repositories ?? []).length > 0) return false;
  return bitbucketOnboardingPollIntervalMs;
}
