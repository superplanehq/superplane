import type { MeVcsProviderIdentity } from "@/api-client";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/ui/dropdownMenu";
import { Loader2 } from "lucide-react";
import { useMemo, useState } from "react";

import { RepositoryPicker } from "../onboardingSteps";
import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunGithubStepper } from "./FirstRunGithubStepper";
import { FirstRunMissingAccessLine } from "./FirstRunMissingAccessLine";
import { FirstRunOrganizationStep } from "./FirstRunOrganizationStep";
import { FirstRunSkeletonRows } from "./FirstRunSkeletonRows";
import {
  findOrganization,
  ownerOfRepository,
  repositoriesInOrganization,
  repositoryBelongsTo,
  repositoryOrganizations,
} from "./githubOrganizations";
import { FirstRunHeading, FirstRunShell } from "./FirstRunShell";
import type { FirstRunSphereProps } from "./FirstRunSpherePane";
import type { FirstRunChrome } from "./firstRunTypes";

const copy = FIRST_RUN_COPY.choose;

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

  return (
    <FirstRunShell testId="first-run-choose" chrome={screenChrome} busy={busy} contentSpacing="compact" sphere={sphere}>
      <FirstRunHeading headline={organization ? copy.headline : copy.organizationHeadline}>
        <p className="text-[13px] text-muted-foreground">
          {organization ? copy.repositoryHelper : copy.organizationHelper}
        </p>
        {githubLogin ? (
          <SignedInAsLine
            login={githubLogin}
            userId={githubUserId}
            identities={githubIdentities}
            disabled={busy}
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
              synchronizing={synchronizing}
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

function SignedInAsLine({
  login,
  userId,
  identities,
  disabled,
  onSelectIdentity,
  onConnectAnotherAccount,
}: {
  login: string;
  userId: string;
  identities: MeVcsProviderIdentity[];
  disabled: boolean;
  onSelectIdentity?: (userId: string) => void;
  onConnectAnotherAccount?: () => void;
}) {
  const [before, after] = FIRST_RUN_COPY.connect.signedInAs(login).split(login);
  return (
    <div className="flex items-center gap-2 text-[13px] leading-5">
      <p className="text-muted-foreground" data-testid="first-run-github-signed-in-as">
        {before}
        <span className="font-medium text-foreground">{login}</span>
        {after}
      </p>
      {onSelectIdentity && onConnectAnotherAccount ? (
        <GitHubAccountMenu
          userId={userId}
          identities={identities}
          disabled={disabled}
          onSelectIdentity={onSelectIdentity}
          onConnectAnotherAccount={onConnectAnotherAccount}
        />
      ) : null}
    </div>
  );
}

function GitHubAccountMenu({
  userId,
  identities,
  disabled,
  onSelectIdentity,
  onConnectAnotherAccount,
}: {
  userId: string;
  identities: MeVcsProviderIdentity[];
  disabled: boolean;
  onSelectIdentity: (userId: string) => void;
  onConnectAnotherAccount: () => void;
}) {
  const linkedIdentities = identities.filter(
    (identity): identity is MeVcsProviderIdentity & { userId: string; login: string } =>
      Boolean(identity.userId && identity.login),
  );

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          variant="link"
          size="sm"
          className="h-auto p-0 text-[13px] font-normal text-muted-foreground hover:text-foreground"
          disabled={disabled}
          data-testid="first-run-switch-github-account"
        >
          {copy.switchAccount}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="w-60">
        <DropdownMenuRadioGroup
          value={userId}
          onValueChange={(nextUserId) => {
            if (nextUserId !== userId) onSelectIdentity(nextUserId);
          }}
        >
          {linkedIdentities.map((identity) => (
            <DropdownMenuRadioItem key={identity.userId} value={identity.userId} className="text-[13px]">
              {identity.login}
            </DropdownMenuRadioItem>
          ))}
        </DropdownMenuRadioGroup>
        {linkedIdentities.length > 0 ? <DropdownMenuSeparator /> : null}
        <DropdownMenuItem className="text-[13px]" onSelect={onConnectAnotherAccount}>
          {copy.connectAnotherAccount}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function RepositoryStepBody({
  repositories,
  selectedRepository,
  busy,
  synchronizing,
  grantAccessDisabled,
  grantingAccess,
  onSelectRepository,
  onGrantAccess,
}: {
  repositories: string[];
  selectedRepository: string | null;
  busy: boolean;
  synchronizing: boolean;
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
      {synchronizing ? (
        <FirstRunSkeletonRows count={1} label={copy.loadingRepositories} testId="first-run-repositories-loading-more" />
      ) : null}
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
