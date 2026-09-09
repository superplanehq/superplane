import { LoadingButton } from "@/components/ui/loading-button";
import { Loader2 } from "lucide-react";

import { RepositoryPicker } from "../onboardingSteps";
import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunGithubStepper } from "./FirstRunGithubStepper";
import { FirstRunHeading, FirstRunPanel, FirstRunShell } from "./FirstRunShell";
import type { FirstRunSphereProps } from "./FirstRunSpherePane";
import type { FirstRunChrome } from "./firstRunTypes";

const copy = FIRST_RUN_COPY.choose;

export function FirstRunChooseScreen({
  repositories,
  selectedRepository,
  loading,
  saving = false,
  chrome,
  sphere,
  stepper,
  onSelectRepository,
  onEditConnection,
  onContinue,
}: {
  repositories: string[];
  selectedRepository: string | null;
  /** True while the repository list loads or refreshes; hides stale entries. */
  loading?: boolean;
  saving?: boolean;
  chrome?: FirstRunChrome;
  sphere?: FirstRunSphereProps;
  /** Renders the GitHub steps on one card (the initial-organization look). */
  stepper?: { organizationName?: string };
  onSelectRepository: (repository: string) => void;
  onEditConnection: () => void;
  onContinue: () => void;
}) {
  const busy = Boolean(loading || saving);

  const repositoryStep = (
    <RepositoryStepBody
      repositories={repositories}
      selectedRepository={selectedRepository}
      loading={loading}
      busy={busy}
      onSelectRepository={onSelectRepository}
      onEditConnection={onEditConnection}
    />
  );
  const continueBlock = (
    <ChooseContinue selectedRepository={selectedRepository} saving={saving} busy={busy} onContinue={onContinue} />
  );

  return (
    <FirstRunShell testId="first-run-choose" chrome={chrome} busy={busy} sphere={sphere}>
      <FirstRunHeading headline={copy.headline}>
        <p className="text-[13px] text-muted-foreground">{copy.repositoryHelper}</p>
      </FirstRunHeading>

      <div className="mt-8 space-y-4">
        {stepper ? (
          <FirstRunGithubStepper current="repository" organizationName={stepper.organizationName}>
            {repositoryStep}
            {continueBlock}
          </FirstRunGithubStepper>
        ) : (
          <>
            <FirstRunPanel>{repositoryStep}</FirstRunPanel>
            {continueBlock}
          </>
        )}
      </div>
    </FirstRunShell>
  );
}

function RepositoryStepBody({
  repositories,
  selectedRepository,
  loading,
  busy,
  onSelectRepository,
  onEditConnection,
}: {
  repositories: string[];
  selectedRepository: string | null;
  loading?: boolean;
  busy: boolean;
  onSelectRepository: (repository: string) => void;
  onEditConnection: () => void;
}) {
  return (
    <>
      {loading ? (
        <RepositoryListLoading />
      ) : (
        <RepositoryPicker
          host="github"
          repos={repositories}
          selectedRepo={selectedRepository}
          disabled={busy}
          onSelect={onSelectRepository}
        />
      )}
      <p className="mt-3 text-[13px] text-muted-foreground">
        {copy.missingRepository}{" "}
        <button
          type="button"
          onClick={onEditConnection}
          disabled={busy}
          className="font-medium text-foreground underline underline-offset-2 hover:no-underline"
        >
          {copy.editConnection}
        </button>
      </p>
    </>
  );
}

function ChooseContinue({
  selectedRepository,
  saving,
  busy,
  onContinue,
}: {
  selectedRepository: string | null;
  saving: boolean;
  busy: boolean;
  onContinue: () => void;
}) {
  return (
    <div className="space-y-2">
      <LoadingButton
        type="button"
        className="w-full"
        disabled={!selectedRepository || busy}
        loading={saving}
        loadingText={copy.saving}
        onClick={onContinue}
        data-testid="first-run-continue-to-tickets"
      >
        {selectedRepository ? copy.continueReady : copy.continue}
      </LoadingButton>
      <p className="text-[12px] text-muted-foreground">{copy.moreLater}</p>
    </div>
  );
}

function RepositoryListLoading() {
  return (
    <div
      className="flex min-h-32 items-center justify-center gap-2 text-[13px] text-muted-foreground"
      data-testid="first-run-repositories-loading"
      role="status"
    >
      <Loader2 className="size-4 animate-spin" data-testid="first-run-repositories-spinner" aria-hidden />
      <span>{FIRST_RUN_COPY.choose.loading}</span>
    </div>
  );
}
