import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { useIntakeCatalogAvailability } from "@/hooks/useIntakeCatalogAvailability";
import { FEATURE_FACTORY_BITBUCKET } from "@/lib/experimentalFeatures";
import { isIntakeSelectable, type IntakeSurfaceEntry, type IntakeSurfaceState } from "@/lib/intakeCatalog";

export type FirstRunIntakeAvailability = {
  organizationReady: boolean;
  intakeState: (key: string) => IntakeSurfaceState | undefined;
  intakesLoading: boolean;
  ticketIntakes: IntakeSurfaceEntry[] | null;
  vcsAvailable: boolean;
  jiraAvailable: boolean;
  jiraFeatureLoading: boolean;
  linearAvailable: boolean;
  linearFeatureLoading: boolean;
  bitbucketAvailable: boolean;
  bitbucketFeatureLoading: boolean;
  bitbucket: { available: boolean; loading: boolean };
};

export function useFirstRunIntakeAvailability(organizationId: string): FirstRunIntakeAvailability {
  const intakeCatalog = useIntakeCatalogAvailability(organizationId);
  const intakeFeatures = useExperimentalFeature(organizationId);

  const vcsAvailable = !intakeCatalog.loaded || isIntakeSelectable(intakeCatalog.stateOf("github-issues"));
  const jiraAvailable = isIntakeSelectable(intakeCatalog.stateOf("jira-issues"));
  const linearAvailable = isIntakeSelectable(intakeCatalog.stateOf("linear-issues"));

  const bitbucketFeatureLoading = intakeFeatures.isLoading;
  const bitbucketAvailable = !bitbucketFeatureLoading && intakeFeatures.has(FEATURE_FACTORY_BITBUCKET);

  return {
    organizationReady: intakeCatalog.loaded,
    intakeState: intakeCatalog.stateOf,
    intakesLoading: intakeCatalog.loading,
    ticketIntakes: intakeCatalog.loaded ? intakeCatalog.entriesFor("onboardingTickets") : null,
    vcsAvailable,
    jiraAvailable,
    jiraFeatureLoading: intakeCatalog.loading,
    linearAvailable,
    linearFeatureLoading: intakeCatalog.loading,
    bitbucketAvailable,
    bitbucketFeatureLoading,
    bitbucket: { available: bitbucketAvailable, loading: bitbucketFeatureLoading },
  };
}
