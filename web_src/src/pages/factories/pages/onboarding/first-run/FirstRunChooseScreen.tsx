import type { MeVcsProviderIdentity } from "@/api-client";
import { LoadingButton } from "@/components/ui/loading-button";
import { Loader2 } from "lucide-react";
import { useMemo, useState } from "react";

import { RepositoryPicker } from "../onboardingSteps";
import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunGithubStepper } from "./FirstRunGithubStepper";
import { FirstRunMissingAccessLine } from "./FirstRunMissingAccessLine";
import { FirstRunOrganizationStep } from "./FirstRunOrganizationStep";
import {
  findOrganization,
  ownerOfRepository,
  repositoriesInOrganization,
  repositoryBelongsTo,
  repositoryOrganizations,
} from "./githubOrganizations";
import { FirstRunHeading, FirstRunShell } from "./FirstRunShell";
import { FirstRunSignedInAsLine, type FirstRunLinkedIdentity } from "./FirstRunSignedInAsLine";
import { repositorySphereFor, sphereFor } from "./firstRunSphereFor";
import type { FirstRunSphereProps } from "./FirstRunSpherePane";
import type { FirstRunChrome } from "./firstRunTypes";

const copy = FIRST_RUN_COPY.choose;
const githubSignedInTestIds = {
  signedInAs: "first-run-github-signed-in-as",
  switchAccount: "first-run-switch-github-account",
};

function githubLinkedIdentities(identities: MeVcsProviderIdentity[]): FirstRunLinkedIdentity[] {
  return identities.map((identity) => ({ key: identity.userId ?? "", login: identity.login ?? "" }));
}

function chooseSphere(
  organization: string | null,
  selectedRepository: string | null,
  sphere: FirstRunSphereProps | undefined,
): FirstRunSphereProps | undefined {
  if (organization) return repositorySphereFor(selectedRepository ?? organization);
  return sphere ?? sphereFor("choose", selectedRepository);
}

export function FirstRunChooseScreen({
  repositories,
  selectedRepository,
  githubLogin = "",
  githubUserId = "",
  githubIdentities = [],
  loading,
  saving = false,
  grantingAccess = false,
  switchingGitHubAccount = false,
  synchronizing = false,
  pendingOrganizations = [],
  appConfigured = true,
  chrome,
  sphere,
  onSelectRepository,
  onClearRepository,
  onSelectGitHubIdentity,
  onConnectAnotherGitHubAccount,
  onGrantAccess,
  onContinue,
}: {
  repositories: string[];
  selectedRepository: string | null;
  githubLogin?: string;
  githubUserId?: string;
  githubIdentities?: MeVcsProviderIdentity[];
  /** True before the initial repository list loads. */
  loading?: boolean;
  saving?: boolean;
  grantingAccess?: boolean;
  switchingGitHubAccount?: boolean;
  synchronizing?: boolean;
  pendingOrganizations?: string[];
  appConfigured?: boolean;
  chrome?: FirstRunChrome;
  sphere?: FirstRunSphereProps;
  onSelectRepository: (repository: string) => void;
  onClearRepository?: () => void;
  onSelectGitHubIdentity?: (userId: string) => void;
  onConnectAnotherGitHubAccount?: () => void;
  onGrantAccess: () => void;
  onContinue: () => void;
}) {
  const busy = Boolean(loading || saving || grantingAccess || switchingGitHubAccount);
  const organizationChoice = useOrganizationChoice(repositories, selectedRepository, onClearRepository);
  const organization = loading ? null : organizationChoice.organization;
  // In the repository step, Back returns to the organization step of this screen.
  const screenChrome = organization && chrome ? { ...chrome, onBack: organizationChoice.clear } : chrome;
  const syncText = organization ? copy.synchronizing : copy.synchronizingOrganizations;
  const syncStatus = synchronizing ? <RepositorySyncStatus text={syncText} /> : undefined;
  const grantAccessDisabled = busy || !appConfigured;

  const displaySphere = chooseSphere(organization, selectedRepository, sphere);

  return (
    <FirstRunShell
      testId="first-run-choose"
      chrome={screenChrome}
      busy={busy}
      contentSpacing="compact"
      sphere={displaySphere}
      visual="preview"
    >
      <FirstRunHeading headline={organization ? copy.headline : copy.organizationHeadline}>
        <p className="text-[13px] text-muted-foreground">
          {organization ? copy.repositoryHelper : copy.organizationHelper}
        </p>
        {githubLogin ? (
          <FirstRunSignedInAsLine
            login={githubLogin}
            activeKey={githubUserId}
            identities={githubLinkedIdentities(githubIdentities)}
            disabled={busy}
            signedInText={FIRST_RUN_COPY.connect.signedInAs(githubLogin)}
            connectAnotherLabel={copy.connectAnotherAccount}
            testIds={githubSignedInTestIds}
            onSelectIdentity={onSelectGitHubIdentity}
            onConnectAnotherAccount={onConnectAnotherGitHubAccount}
          />
        ) : null}
      </FirstRunHeading>

      <div className="mt-8 space-y-4">
        {organization ? (
          <FirstRunGithubStepper current="repository" organizationName={organization} action={syncStatus}>
            <RepositoryStepBody
              repositories={repositoriesInOrganization(repositories, organization)}
              selectedRepository={selectedRepository}
              busy={busy}
              grantAccessDisabled={grantAccessDisabled}
              grantingAccess={grantingAccess}
              onSelectRepository={onSelectRepository}
              onGrantAccess={onGrantAccess}
            />
            <ChooseContinue
              selectedRepository={selectedRepository}
              saving={saving}
              busy={busy}
              onContinue={onContinue}
            />
          </FirstRunGithubStepper>
        ) : (
          <FirstRunGithubStepper current="organization" action={syncStatus}>
            {loading ? (
              <RepositoryListLoading />
            ) : (
              <FirstRunOrganizationStep
                organizations={organizationChoice.organizations}
                pendingOrganizations={pendingOrganizations}
                synchronizing={synchronizing}
                disabled={busy}
                grantAccessDisabled={grantAccessDisabled}
                grantingAccess={grantingAccess}
                onSelectOrganization={organizationChoice.choose}
                onGrantAccess={onGrantAccess}
              />
            )}
          </FirstRunGithubStepper>
        )}
      </div>
    </FirstRunShell>
  );
}

/**
 * The organization is only a filter for the repository list. A saved
 * repository opens its owner. An organization that leaves the list, for
 * example after an account switch, returns the user to the organization step.
 */
function useOrganizationChoice(
  repositories: string[],
  selectedRepository: string | null,
  onClearRepository?: () => void,
) {
  const organizations = useMemo(() => repositoryOrganizations(repositories), [repositories]);
  const [chosen, setChosen] = useState<string | null>(() => ownerOfRepository(selectedRepository));
  return {
    organizations,
    organization: findOrganization(organizations, chosen),
    choose: (next: string) => {
      if (selectedRepository && !repositoryBelongsTo(selectedRepository, next)) onClearRepository?.();
      setChosen(next);
    },
    clear: () => setChosen(null),
  };
}

function RepositoryStepBody({
  repositories,
  selectedRepository,
  busy,
  grantAccessDisabled,
  grantingAccess,
  onSelectRepository,
  onGrantAccess,
}: {
  repositories: string[];
  selectedRepository: string | null;
  busy: boolean;
  grantAccessDisabled: boolean;
  grantingAccess: boolean;
  onSelectRepository: (repository: string) => void;
  onGrantAccess: () => void;
}) {
  return (
    <>
      <RepositoryPicker
        host="github"
        repos={repositories}
        selectedRepo={selectedRepository}
        disabled={busy}
        listClassName="max-h-48"
        onSelect={onSelectRepository}
      />
      <FirstRunMissingAccessLine
        hint={copy.writeAccessHint}
        question={copy.missingRepository}
        action={copy.grantAccess}
        disabled={grantAccessDisabled}
        loading={grantingAccess}
        onClick={onGrantAccess}
      />
    </>
  );
}

function RepositorySyncStatus({ text }: { text: string }) {
  return (
    <span className="flex items-center gap-2 text-[12px] font-normal text-muted-foreground">
      <span
        className="sp-ai-thinking inline-block leading-5"
        data-text={text}
        data-testid="first-run-repositories-synchronizing"
        role="status"
      >
        {text}
      </span>
    </span>
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
