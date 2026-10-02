import type { MeVcsProviderIdentity } from "@/api-client";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/ui/dropdownMenu";
import { Clock, Loader2 } from "lucide-react";

import { RepositoryPicker } from "../onboardingSteps";
import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunGithubStepper } from "./FirstRunGithubStepper";
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
  onSelectGitHubIdentity?: (userId: string) => void;
  onConnectAnotherGitHubAccount?: () => void;
  onGrantAccess: () => void;
  onContinue: () => void;
}) {
  const busy = Boolean(loading || saving || grantingAccess || switchingGitHubAccount);

  return (
    <FirstRunShell testId="first-run-choose" chrome={chrome} busy={busy} contentSpacing="compact" sphere={sphere}>
      <FirstRunHeading headline={copy.headline}>
        <p className="text-[13px] text-muted-foreground">{copy.repositoryHelper}</p>
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
        <FirstRunGithubStepper current="repository" action={synchronizing ? <RepositorySyncStatus /> : undefined}>
          <RepositoryStepBody
            repositories={repositories}
            selectedRepository={selectedRepository}
            loading={loading}
            busy={busy}
            grantingAccess={grantingAccess}
            pendingOrganizations={pendingOrganizations}
            appConfigured={appConfigured}
            onSelectRepository={onSelectRepository}
            onGrantAccess={onGrantAccess}
          />
          <ChooseContinue selectedRepository={selectedRepository} saving={saving} busy={busy} onContinue={onContinue} />
        </FirstRunGithubStepper>
      </div>
    </FirstRunShell>
  );
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
  loading,
  busy,
  grantingAccess,
  pendingOrganizations,
  appConfigured,
  onSelectRepository,
  onGrantAccess,
}: {
  repositories: string[];
  selectedRepository: string | null;
  loading?: boolean;
  busy: boolean;
  grantingAccess: boolean;
  pendingOrganizations: string[];
  appConfigured: boolean;
  onSelectRepository: (repository: string) => void;
  onGrantAccess: () => void;
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
          listClassName="max-h-48"
          onSelect={onSelectRepository}
        />
      )}
      {pendingOrganizations.length > 0 ? <PendingApprovalRows organizations={pendingOrganizations} /> : null}
      <p className="mt-3 text-[13px] text-muted-foreground">
        {copy.missingRepository}{" "}
        <button
          type="button"
          onClick={onGrantAccess}
          disabled={busy || !appConfigured}
          className="font-medium text-foreground underline underline-offset-2 hover:no-underline disabled:cursor-not-allowed disabled:opacity-50"
        >
          {grantingAccess ? copy.openingGitHub : copy.grantAccess}
        </button>
      </p>
    </>
  );
}

function RepositorySyncStatus() {
  return (
    <span className="flex items-center gap-2 text-[12px] font-normal text-muted-foreground">
      <span
        className="sp-ai-thinking inline-block leading-5"
        data-text={copy.synchronizing}
        data-testid="first-run-repositories-synchronizing"
        role="status"
      >
        {copy.synchronizing}
      </span>
    </span>
  );
}

function PendingApprovalRows({ organizations }: { organizations: string[] }) {
  return (
    <div className="space-y-2 text-left" data-testid="first-run-github-install-requested">
      {organizations.map((organization) => (
        <Tooltip key={organization}>
          <TooltipTrigger asChild>
            <div
              className="flex items-center justify-between gap-3 rounded-md border border-border px-3 py-2.5 text-[13px]"
              data-testid="first-run-github-waiting-row"
            >
              <span className="min-w-0 truncate font-medium">{copy.installRequested(organization)}</span>
              <Clock className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />
            </div>
          </TooltipTrigger>
          <TooltipContent side="top" className="max-w-md space-y-1 text-left text-pretty">
            <p>{copy.installRequestedBody(organization)}</p>
            <p>{copy.installRequestedNext}</p>
          </TooltipContent>
        </Tooltip>
      ))}
    </div>
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
