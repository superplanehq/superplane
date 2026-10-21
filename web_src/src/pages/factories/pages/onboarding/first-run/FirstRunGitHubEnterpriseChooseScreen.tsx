import { LoadingButton } from "@/components/ui/loading-button";
import { useIntegrationResources } from "@/hooks/useIntegrations";
import { Loader2 } from "lucide-react";

import type { FirstRunSetupFlow, OnboardingPageModel } from "../useFirstRunSetupFlow";
import { RepositoryPicker } from "../onboardingSteps";
import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunHeading, FirstRunShell } from "./FirstRunShell";
import type { FirstRunSphereProps } from "./FirstRunSpherePane";
import { repositorySphereFor } from "./firstRunSphereFor";
import type { FirstRunChrome } from "./firstRunTypes";

const copy = FIRST_RUN_COPY.githubEnterprise;
const chooseCopy = FIRST_RUN_COPY.choose;

export function GitHubEnterpriseChooseHost({
  organizationId,
  flow,
  model,
  chrome,
}: {
  organizationId: string;
  flow: FirstRunSetupFlow;
  model: OnboardingPageModel;
  chrome?: FirstRunChrome;
}) {
  const integrationId = flow.githubEnterpriseIntegrationId || model.githubIntegrationId;
  const resources = useIntegrationResources(organizationId, integrationId, "repository", undefined, {
    enabled: Boolean(integrationId),
  });
  const repositories = (resources.data ?? []).map((resource) => resource.name ?? "").filter(Boolean);
  return (
    <FirstRunGitHubEnterpriseChooseScreen
      repositories={repositories}
      selectedRepository={model.setup.selectedRepo}
      loading={resources.isLoading}
      loadError={resources.isError}
      saving={flow.blockingAction === "saving-repository"}
      chrome={chrome}
      sphere={repositorySphereFor(model.setup.selectedRepo)}
      onSelectRepository={model.setup.selectRepo}
      onContinue={() => void flow.continueFromRepository()}
    />
  );
}

export function FirstRunGitHubEnterpriseChooseScreen({
  repositories,
  selectedRepository,
  loading = false,
  loadError = false,
  saving = false,
  chrome,
  sphere,
  onSelectRepository,
  onContinue,
}: {
  repositories: string[];
  selectedRepository: string | null;
  loading?: boolean;
  loadError?: boolean;
  saving?: boolean;
  chrome?: FirstRunChrome;
  sphere?: FirstRunSphereProps;
  onSelectRepository: (repository: string) => void;
  onContinue: () => void;
}) {
  const busy = loading || saving;
  return (
    <FirstRunShell
      testId="first-run-github-enterprise-choose"
      chrome={chrome}
      busy={busy}
      contentSpacing="compact"
      sphere={sphere}
      visual="preview"
    >
      <FirstRunHeading headline={chooseCopy.headline}>
        <p className="text-[13px] text-muted-foreground">{copy.repositoryHelper}</p>
      </FirstRunHeading>

      <div className="mt-8 space-y-4">
        <GitHubEnterpriseRepositoryList
          repositories={repositories}
          selectedRepository={selectedRepository}
          loading={loading}
          loadError={loadError}
          disabled={busy}
          onSelectRepository={onSelectRepository}
        />
        <div className="space-y-2">
          <LoadingButton
            type="button"
            className="w-full"
            disabled={!selectedRepository || busy}
            loading={saving}
            loadingText={chooseCopy.saving}
            onClick={onContinue}
            data-testid="first-run-continue-to-tickets"
          >
            {selectedRepository ? chooseCopy.continueReady : chooseCopy.continue}
          </LoadingButton>
          <p className="text-[12px] text-muted-foreground">{chooseCopy.moreLater}</p>
        </div>
      </div>
    </FirstRunShell>
  );
}

function GitHubEnterpriseRepositoryList({
  repositories,
  selectedRepository,
  loading,
  loadError,
  disabled,
  onSelectRepository,
}: {
  repositories: string[];
  selectedRepository: string | null;
  loading: boolean;
  loadError: boolean;
  disabled: boolean;
  onSelectRepository: (repository: string) => void;
}) {
  if (loading) {
    return (
      <div
        className="flex min-h-32 items-center justify-center gap-2 text-[13px] text-muted-foreground"
        data-testid="first-run-github-enterprise-repositories-loading"
        role="status"
      >
        <Loader2 className="size-4 animate-spin" aria-hidden />
        <span>{copy.loading}</span>
      </div>
    );
  }
  if (loadError) return <p className="text-[13px] text-destructive">{copy.loadError}</p>;
  if (repositories.length === 0) return <p className="text-[13px] text-muted-foreground">{copy.empty}</p>;
  return (
    <RepositoryPicker
      host="github"
      repos={repositories}
      selectedRepo={selectedRepository}
      disabled={disabled}
      listClassName="max-h-48"
      onSelect={onSelectRepository}
    />
  );
}
