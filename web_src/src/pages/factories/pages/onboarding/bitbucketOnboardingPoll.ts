export const bitbucketOnboardingPollIntervalMs = 3_000;

export type BitbucketOnboardingPollState = {
  status?: string;
  data?: {
    providerConfigured?: boolean;
    identity?: unknown;
    repositories?: unknown[];
  };
};

export function bitbucketOnboardingPollInterval(poll: boolean, state: BitbucketOnboardingPollState): number | false {
  if (!poll) return false;
  if (state.status === "error") return bitbucketOnboardingPollIntervalMs;
  const data = state.data;
  if (!data?.providerConfigured || !data.identity) return false;
  if ((data.repositories ?? []).length > 0) return false;
  return bitbucketOnboardingPollIntervalMs;
}
