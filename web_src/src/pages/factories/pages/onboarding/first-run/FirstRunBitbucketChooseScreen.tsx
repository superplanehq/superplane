import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { Loader2 } from "lucide-react";

import { RepositoryPicker } from "../onboardingSteps";
import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunHeading, FirstRunShell } from "./FirstRunShell";
import type { FirstRunSphereProps } from "./FirstRunSpherePane";
import type { FirstRunChrome } from "./firstRunTypes";

const copy = FIRST_RUN_COPY.bitbucket;
const chooseCopy = FIRST_RUN_COPY.choose;

export function FirstRunBitbucketChooseScreen({
  connected,
  repositories,
  selectedRepository,
  loading = false,
  loadError = false,
  saving = false,
  chrome,
  sphere,
  onConnect,
  onSelectRepository,
  onContinue,
}: {
  connected: boolean;
  repositories: string[];
  selectedRepository: string | null;
  loading?: boolean;
  loadError?: boolean;
  saving?: boolean;
  chrome?: FirstRunChrome;
  sphere?: FirstRunSphereProps;
  onConnect: () => void;
  onSelectRepository: (repository: string) => void;
  onContinue: () => void;
}) {
  if (!connected) {
    return (
      <FirstRunShell testId="first-run-bitbucket-connect" chrome={chrome} sphere={sphere} visual="preview">
        <FirstRunHeading headline={copy.connectHeadline}>
          <p className="text-[15px] leading-6 text-muted-foreground">{copy.connectBody}</p>
        </FirstRunHeading>
        <div className="mt-8">
          <Button type="button" onClick={onConnect} data-testid="first-run-connect-bitbucket">
            {copy.connectAction}
          </Button>
        </div>
      </FirstRunShell>
    );
  }

  const busy = loading || saving;
  return (
    <FirstRunShell
      testId="first-run-bitbucket-choose"
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
        <BitbucketRepositoryList
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

function BitbucketRepositoryList({
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
        data-testid="first-run-bitbucket-repositories-loading"
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
      host="bitbucket"
      repos={repositories}
      selectedRepo={selectedRepository}
      disabled={disabled}
      listClassName="max-h-48"
      onSelect={onSelectRepository}
    />
  );
}
