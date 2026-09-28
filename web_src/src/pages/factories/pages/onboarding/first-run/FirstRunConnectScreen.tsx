import { LoadingButton } from "@/components/ui/loading-button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Clock } from "lucide-react";

import { hostedGitHubInstallURL, type PendingGitHubInstallation } from "@/lib/hostedGitHubInstall";

import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunGithubStepper } from "./FirstRunGithubStepper";
import { FirstRunHeading, FirstRunShell } from "./FirstRunShell";
import type { FirstRunSphereProps } from "./FirstRunSpherePane";
import type { FirstRunChrome } from "./firstRunTypes";

const copy = FIRST_RUN_COPY.connect;

function SignedInAsLine({ login }: { login: string }) {
  const [before, after] = copy.signedInAs(login).split(login);
  return (
    <p className="text-[15px] leading-6 text-muted-foreground" data-testid="first-run-github-signed-in-as">
      {before}
      <span className="font-medium text-foreground">{login}</span>
      {after}
    </p>
  );
}

function FirstRunGitHubInstallLink({
  githubAppSlug,
  githubState,
  disabled = false,
  onInstallOther,
}: {
  githubAppSlug: string;
  githubState: string;
  disabled?: boolean;
  onInstallOther?: () => void;
}) {
  if (githubAppSlug === "") return null;

  return (
    <p className="text-[13px] text-muted-foreground">
      {copy.missingAccount}{" "}
      <a
        href={hostedGitHubInstallURL(githubAppSlug, githubState)}
        data-testid="first-run-github-install-other"
        aria-disabled={disabled || undefined}
        tabIndex={disabled ? -1 : undefined}
        onClick={(event) => {
          if (disabled || onInstallOther) event.preventDefault();
          if (!disabled) onInstallOther?.();
        }}
        className={`font-medium text-foreground underline underline-offset-2 hover:no-underline ${disabled ? "pointer-events-none opacity-50" : ""}`}
      >
        {copy.installThere}
      </a>
    </p>
  );
}

/**
 * Requested organizations as rows in the organization list, so a pending
 * approval reads as one more organization instead of a separate panel. The
 * connect screen rechecks the request every few seconds; an approved
 * organization replaces its waiting row with a Use button. The tooltip
 * carries the detail (who approves, no refresh needed) so the row itself
 * stays one line.
 */
function FirstRunPendingOrganizationRows({ githubOrganizations }: { githubOrganizations: string[] }) {
  const rows = githubOrganizations.length > 0 ? githubOrganizations : [""];
  return (
    <div className="space-y-2 text-left" data-testid="first-run-github-install-requested">
      {rows.map((account) => (
        <Tooltip key={account}>
          <TooltipTrigger asChild>
            <div
              className="flex items-center justify-between gap-3 rounded-md border border-border px-3 py-2.5 text-[13px]"
              data-testid="first-run-github-waiting-row"
            >
              <span className="min-w-0 truncate font-medium">{account || copy.installRequested}</span>
              <span className="flex shrink-0 items-center gap-1.5 text-muted-foreground">
                <Clock className="size-3.5" aria-hidden />
                {account ? copy.installRequested : null}
              </span>
            </div>
          </TooltipTrigger>
          <TooltipContent side="top" className="max-w-md space-y-1 text-left text-pretty">
            <p>{copy.installRequestedBody(account)}</p>
            <p>{copy.installRequestedNext}</p>
          </TooltipContent>
        </Tooltip>
      ))}
    </div>
  );
}

function FirstRunGitHubAccountPicker({
  installations,
  githubAppSlug,
  githubState,
  bindingInstallationId,
  disabled,
  discoveringAccounts,
  syncError,
  onUseInstallation,
  onInstallOther,
  onRetrySync,
}: {
  installations: PendingGitHubInstallation[];
  githubAppSlug: string;
  githubState: string;
  bindingInstallationId?: string;
  disabled?: boolean;
  discoveringAccounts?: boolean;
  syncError?: string;
  onUseInstallation: (installation: PendingGitHubInstallation) => void;
  onInstallOther?: () => void;
  onRetrySync?: () => void;
}) {
  const binding = bindingInstallationId !== undefined || disabled;
  return (
    <div className="space-y-3 text-left" data-testid="first-run-github-account-picker">
      {installations
        .filter((installation) => installation.repositories.length > 0)
        .map((installation) => (
          <LoadingButton
            key={installation.id}
            type="button"
            className="w-full justify-start"
            data-testid={`first-run-github-use-${installation.accountLogin}`}
            loading={bindingInstallationId === installation.id}
            loadingText={copy.connectingAccount(installation.accountLogin)}
            disabled={binding}
            onClick={() => onUseInstallation(installation)}
          >
            {copy.useAccount(installation.accountLogin)}
          </LoadingButton>
        ))}
      {discoveringAccounts ? (
        <p className="text-[13px] text-muted-foreground" role="status">
          {copy.checkingMoreAccounts}
        </p>
      ) : null}
      {syncError ? (
        <div className="space-y-2">
          <p className="text-[13px] text-destructive" role="alert">
            {syncError}
          </p>
          {onRetrySync ? (
            <LoadingButton type="button" size="sm" variant="outline" onClick={onRetrySync}>
              {copy.tryAgain}
            </LoadingButton>
          ) : null}
        </div>
      ) : null}
      <FirstRunGitHubInstallLink
        githubAppSlug={githubAppSlug}
        githubState={githubState}
        disabled={binding}
        onInstallOther={onInstallOther}
      />
      {disabled ? (
        <p className="text-[13px] text-muted-foreground" role="status">
          {copy.openingGitHub}
        </p>
      ) : null}
    </div>
  );
}

function connectScreenState({
  installRequested,
  githubOrganizations,
  pendingInstallations,
  githubState,
  discoveringAccounts,
  installAvailable,
  syncError,
}: {
  installRequested: boolean;
  githubOrganizations: string[];
  pendingInstallations: PendingGitHubInstallation[];
  githubState: string;
  discoveringAccounts: boolean;
  installAvailable: boolean;
  syncError?: string;
}) {
  const showAccountPicker =
    githubState !== "" &&
    (pendingInstallations.length >= 1 || (discoveringAccounts && (Boolean(syncError) || installAvailable)));
  // A requested organization is ready only after the server verifies at
  // least one repository that the member can use. An installation row with
  // no repositories can be stale and must not clear the waiting state.
  const requestApproved =
    githubOrganizations.length > 0 &&
    githubOrganizations.every((organization) =>
      pendingInstallations.some(
        (installation) =>
          installation.accountLogin.toLowerCase() === organization.toLowerCase() &&
          installation.repositories.length > 0,
      ),
    );
  return {
    showAccountPicker,
    waitingForApproval: installRequested && !(showAccountPicker && requestApproved),
  };
}

export function FirstRunConnectScreen({
  loading = false,
  pickerExpected = false,
  installRequested = false,
  discoveringAccounts = false,
  installAvailable = false,
  githubOrganization = "",
  githubOrganizations,
  pendingInstallations = [],
  githubState = "",
  githubAppSlug = "",
  githubLogin = "",
  bindingInstallationId,
  connecting = false,
  connectError,
  syncError,
  chrome,
  sphere,
  onConnectGitHub,
  onUseInstallation,
  onInstallOther,
  onRetrySync,
}: {
  /** True while the picker data still loads after a GitHub round trip. */
  loading?: boolean;
  /** True after GitHub returns, while the organization picker synchronizes. */
  pickerExpected?: boolean;
  installRequested?: boolean;
  discoveringAccounts?: boolean;
  installAvailable?: boolean;
  githubOrganization?: string;
  githubOrganizations?: string[];
  pendingInstallations?: PendingGitHubInstallation[];
  githubState?: string;
  githubAppSlug?: string;
  /** GitHub login that authorized this connect. Shown only with the account picker. */
  githubLogin?: string;
  bindingInstallationId?: string;
  connecting?: boolean;
  connectError?: string;
  syncError?: string;
  chrome?: FirstRunChrome;
  sphere?: FirstRunSphereProps;
  onConnectGitHub: () => void;
  onUseInstallation?: (installation: PendingGitHubInstallation) => void;
  onInstallOther?: () => void;
  onRetrySync?: () => void;
}) {
  const requestedOrganizations = requestedGitHubOrganizations(githubOrganizations, githubOrganization);
  const { showAccountPicker, waitingForApproval } = connectScreenState({
    installRequested,
    githubOrganizations: requestedOrganizations,
    pendingInstallations,
    githubState,
    discoveringAccounts,
    installAvailable,
    syncError,
  });

  return (
    <FirstRunShell
      testId="first-run-connect"
      chrome={chrome}
      busy={connecting || bindingInstallationId !== undefined}
      sphere={sphere}
    >
      <ConnectScreenHeading showAccountPicker={showAccountPicker || pickerExpected} githubLogin={githubLogin} />

      <div className="mt-8 space-y-6">
        {loading ? (
          <ConnectScreenLoading
            githubAppSlug={githubAppSlug}
            githubState={githubState}
            onInstallOther={onInstallOther}
          />
        ) : syncError && pickerExpected && !showAccountPicker ? (
          <ConnectScreenSyncError error={syncError} onRetry={onRetrySync} />
        ) : (
          <ConnectScreenBody
            githubOrganizations={requestedOrganizations}
            pendingInstallations={pendingInstallations}
            githubState={githubState}
            githubAppSlug={githubAppSlug}
            bindingInstallationId={bindingInstallationId}
            connecting={connecting}
            discoveringAccounts={discoveringAccounts}
            syncError={syncError}
            showAccountPicker={showAccountPicker}
            waitingForApproval={waitingForApproval}
            onConnectGitHub={onConnectGitHub}
            onUseInstallation={onUseInstallation}
            onInstallOther={onInstallOther}
            onRetrySync={onRetrySync}
          />
        )}
        <ConnectScreenError error={connectError} waitingForApproval={waitingForApproval} />
      </div>
    </FirstRunShell>
  );
}

function requestedGitHubOrganizations(organizations: string[] | undefined, legacyOrganization: string): string[] {
  if (organizations) return organizations;
  return legacyOrganization ? [legacyOrganization] : [];
}

function ConnectScreenHeading({ showAccountPicker, githubLogin }: { showAccountPicker: boolean; githubLogin: string }) {
  // On the stepper card the picker is the organization step, so the heading
  // asks the organization question instead of repeating the connect ask.
  if (showAccountPicker) {
    return (
      <FirstRunHeading headline={copy.selectAccount}>
        <p className="text-[15px] leading-6 text-muted-foreground">{copy.selectAccountBody}</p>
        {githubLogin ? <SignedInAsLine login={githubLogin} /> : null}
      </FirstRunHeading>
    );
  }
  return (
    <FirstRunHeading headline={copy.headline}>
      <p className="text-[15px] leading-6 text-muted-foreground">{copy.body}</p>
    </FirstRunHeading>
  );
}

function ConnectScreenError({ error, waitingForApproval }: { error?: string; waitingForApproval: boolean }) {
  if (!error || waitingForApproval) return null;
  return <p className="text-[13px] text-destructive">{error}</p>;
}

/**
 * Placeholder while the picker data loads after a GitHub round trip, so the
 * screen does not flash the connect button before the picker.
 */
function ConnectScreenLoading({
  githubAppSlug,
  githubState,
  onInstallOther,
}: {
  githubAppSlug: string;
  githubState: string;
  onInstallOther?: () => void;
}) {
  return (
    <div className="space-y-3" data-testid="first-run-connect-loading">
      <p className="text-[13px] text-muted-foreground" role="status">
        {copy.loadingAccounts}
      </p>
      <div aria-hidden>
        <div className="h-14 animate-pulse rounded-md bg-accent/40" />
        <div className="mt-3 h-9 w-40 animate-pulse rounded-md bg-accent/40" />
      </div>
      <FirstRunGitHubInstallLink
        githubAppSlug={githubAppSlug}
        githubState={githubState}
        onInstallOther={onInstallOther}
      />
    </div>
  );
}

function ConnectScreenSyncError({ error, onRetry }: { error: string; onRetry?: () => void }) {
  return (
    <FirstRunGithubStepper current="organization">
      <div className="space-y-3 text-left">
        <p className="text-[13px] text-destructive" role="alert">
          {error}
        </p>
        {onRetry ? (
          <LoadingButton type="button" size="sm" variant="outline" onClick={onRetry}>
            {copy.tryAgain}
          </LoadingButton>
        ) : null}
      </div>
    </FirstRunGithubStepper>
  );
}

function ConnectScreenBody({
  githubOrganizations,
  pendingInstallations,
  githubState,
  githubAppSlug,
  bindingInstallationId,
  connecting,
  discoveringAccounts,
  syncError,
  showAccountPicker,
  waitingForApproval,
  onConnectGitHub,
  onUseInstallation,
  onInstallOther,
  onRetrySync,
}: {
  githubOrganizations: string[];
  pendingInstallations: PendingGitHubInstallation[];
  githubState: string;
  githubAppSlug: string;
  bindingInstallationId?: string;
  connecting: boolean;
  discoveringAccounts: boolean;
  syncError?: string;
  showAccountPicker: boolean;
  waitingForApproval: boolean;
  onConnectGitHub: () => void;
  onUseInstallation?: (installation: PendingGitHubInstallation) => void;
  onInstallOther?: () => void;
  onRetrySync?: () => void;
}) {
  const organizationStatus = waitingForApproval ? (
    <FirstRunPendingOrganizationRows githubOrganizations={githubOrganizations} />
  ) : null;

  if (showAccountPicker && onUseInstallation) {
    return (
      <FirstRunGithubStepper current="organization" organizationStatus={organizationStatus}>
        <FirstRunGitHubAccountPicker
          installations={pendingInstallations}
          githubAppSlug={githubAppSlug}
          githubState={githubState}
          bindingInstallationId={bindingInstallationId}
          disabled={connecting}
          discoveringAccounts={discoveringAccounts}
          syncError={syncError}
          onUseInstallation={onUseInstallation}
          onInstallOther={onInstallOther}
          onRetrySync={onRetrySync}
        />
      </FirstRunGithubStepper>
    );
  }

  return (
    <FirstRunGithubStepper
      current="connect"
      action={
        <LoadingButton
          type="button"
          size="sm"
          onClick={onConnectGitHub}
          loading={connecting}
          loadingText={copy.openingGitHub}
          data-testid="first-run-connect-github"
        >
          {copy.connectAction}
        </LoadingButton>
      }
      organizationStatus={organizationStatus}
    />
  );
}
