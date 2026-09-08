import { Avatar } from "@/components/Avatar/avatar";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { ChevronRight, Clock, Loader2 } from "lucide-react";

import {
  hostedGitHubInstallURL,
  hostedGitHubProfileURL,
  type PendingGitHubInstallation,
} from "@/lib/hostedGitHubInstall";
import { cn } from "@/lib/utils";

import { IntegrationChoiceIcon } from "../onboardingSteps";
import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunHeading, FirstRunPanel, FirstRunShell } from "./FirstRunShell";
import type { FirstRunChrome } from "./firstRunTypes";

const copy = FIRST_RUN_COPY.connect;

function SignedInAsLine({ login }: { login: string }) {
  const handle = `@${login}`;
  const [before, after] = copy.signedInAs(login).split(handle);
  return (
    <p className="text-[15px] leading-6 text-muted-foreground" data-testid="first-run-github-signed-in-as">
      {before}
      <a
        href={hostedGitHubProfileURL(login)}
        className="font-medium text-foreground underline underline-offset-2 hover:no-underline"
      >
        {handle}
      </a>
      {after}
    </p>
  );
}

function FirstRunInstallRequested({ githubOrganization }: { githubOrganization: string }) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <div data-testid="first-run-github-install-requested">
          <FirstRunPanel>
            <div className="flex items-start gap-3">
              <IntegrationChoiceIcon name="github" />
              <div className="min-w-0 flex-1 text-left">
                <p className="text-[13px] font-medium">{copy.installRequested}</p>
                {githubOrganization ? (
                  <p className="mt-0.5 text-[13px] text-muted-foreground" data-testid="first-run-github-install-org">
                    {githubOrganization}
                  </p>
                ) : null}
              </div>
              <Clock className="mt-0.5 size-3.5 shrink-0 text-muted-foreground" aria-hidden />
            </div>
          </FirstRunPanel>
        </div>
      </TooltipTrigger>
      <TooltipContent
        side="top"
        className="w-[var(--radix-tooltip-trigger-width)] max-w-md space-y-1 text-left text-pretty"
      >
        <p>{copy.installRequestedBody(githubOrganization)}</p>
        <p>{copy.installRequestedNext}</p>
      </TooltipContent>
    </Tooltip>
  );
}

type PickerRow =
  | { kind: "waiting"; login: string }
  | { kind: "ready"; installation: PendingGitHubInstallation };

function githubAccountPickerRows(
  installations: PendingGitHubInstallation[],
  waitingForApproval: boolean,
  githubOrganization: string,
): PickerRow[] {
  const ready: PickerRow[] = installations.map((installation) => ({ kind: "ready", installation }));
  if (!waitingForApproval) {
    return ready;
  }
  const requested = githubOrganization.trim();
  const hasRequested = installations.some(
    (installation) => installation.accountLogin.toLowerCase() === requested.toLowerCase(),
  );
  if (hasRequested) {
    return ready;
  }
  return [{ kind: "waiting", login: requested }, ...ready];
}

function accountInitials(login: string): string {
  return login.slice(0, 1).toUpperCase() || "G";
}

function AccountGlyph({ login }: { login: string }) {
  return (
    <Avatar
      square
      initials={accountInitials(login)}
      alt=""
      className="size-8 bg-muted text-muted-foreground"
    />
  );
}

function AccountPickerWaitingRow({ login }: { login: string }) {
  const name = login !== "" ? login : copy.waitingAccount;
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <div
          tabIndex={0}
          className="flex cursor-default items-center gap-3 px-4 py-3"
          data-testid="first-run-github-install-requested"
        >
          <AccountGlyph login={name} />
          <span
            className="min-w-0 flex-1 truncate text-left text-[14px] font-medium"
            data-testid={login !== "" ? "first-run-github-install-org" : undefined}
          >
            {name}
          </span>
          <span className="shrink-0 text-[13px] text-muted-foreground">{copy.installRequested}</span>
        </div>
      </TooltipTrigger>
      <TooltipContent side="top" className="max-w-md text-left text-pretty">
        <p>{copy.installRequestedBody(login)}</p>
      </TooltipContent>
    </Tooltip>
  );
}

function AccountPickerReadyRow({
  installation,
  binding,
  bindingThis,
  onUseInstallation,
}: {
  installation: PendingGitHubInstallation;
  binding: boolean;
  bindingThis: boolean;
  onUseInstallation: (installation: PendingGitHubInstallation) => void;
}) {
  return (
    <button
      type="button"
      className={cn(
        "flex w-full items-center gap-3 px-4 py-3 text-left hover:bg-accent",
        binding && "opacity-60",
      )}
      data-testid={`first-run-github-use-${installation.accountLogin}`}
      aria-label={copy.useAccount(installation.accountLogin)}
      disabled={binding}
      onClick={() => onUseInstallation(installation)}
    >
      <AccountGlyph login={installation.accountLogin} />
      <span className="min-w-0 flex-1 truncate text-[14px] font-medium">{installation.accountLogin}</span>
      {bindingThis ? (
        <Loader2 className="size-4 shrink-0 animate-spin text-muted-foreground" aria-hidden />
      ) : (
        <ChevronRight className="size-4 shrink-0 text-muted-foreground" aria-hidden />
      )}
    </button>
  );
}

function FirstRunGitHubAccountPicker({
  installations,
  githubAppSlug,
  githubState,
  githubOrganization,
  waitingForApproval,
  bindingInstallationId,
  onUseInstallation,
}: {
  installations: PendingGitHubInstallation[];
  githubAppSlug: string;
  githubState: string;
  githubOrganization: string;
  waitingForApproval: boolean;
  bindingInstallationId?: string;
  onUseInstallation: (installation: PendingGitHubInstallation) => void;
}) {
  const binding = bindingInstallationId !== undefined;
  const rows = githubAccountPickerRows(installations, waitingForApproval, githubOrganization);
  return (
    <div className="space-y-3 text-left" data-testid="first-run-github-account-picker">
      <div className="divide-y divide-border overflow-hidden rounded-lg border border-border bg-card">
        {rows.map((row) =>
          row.kind === "waiting" ? (
            <AccountPickerWaitingRow key={`waiting-${row.login}`} login={row.login} />
          ) : (
            <AccountPickerReadyRow
              key={row.installation.id}
              installation={row.installation}
              binding={binding}
              bindingThis={bindingInstallationId === row.installation.id}
              onUseInstallation={onUseInstallation}
            />
          ),
        )}
      </div>
      {githubAppSlug !== "" ? (
        <p className="text-[13px] text-muted-foreground">
          {copy.missingAccount}{" "}
          <a
            href={hostedGitHubInstallURL(githubAppSlug, githubState)}
            className="font-medium text-foreground underline underline-offset-2 hover:no-underline"
            data-testid="first-run-github-install-other"
          >
            {copy.installThere}
          </a>
        </p>
      ) : null}
    </div>
  );
}

function connectScreenState({
  installRequested,
  githubOrganization,
  pendingInstallations,
  githubState,
}: {
  installRequested: boolean;
  githubOrganization: string;
  pendingInstallations: PendingGitHubInstallation[];
  githubState: string;
}) {
  const hasPickerData = pendingInstallations.length >= 1 && githubState !== "";
  // A picker that offers the requested organization means the request is
  // approved, so the waiting state must not show next to it.
  const requestApproved = pendingInstallations.some(
    (installation) => installation.accountLogin.toLowerCase() === githubOrganization.toLowerCase(),
  );
  const waitingForApproval = installRequested && !(hasPickerData && requestApproved);
  // After GitHub returns, the account list is the page. A pending request is
  // a row on that list, even when no account is ready to use yet.
  const showAccountPicker = hasPickerData || (waitingForApproval && githubState !== "");
  return {
    showAccountPicker,
    waitingForApproval,
  };
}

export function FirstRunConnectScreen({
  loading = false,
  installRequested = false,
  githubOrganization = "",
  pendingInstallations = [],
  githubState = "",
  githubAppSlug = "",
  githubLogin = "",
  bindingInstallationId,
  connectError,
  chrome,
  onConnectGitHub,
  onUseInstallation,
}: {
  /** True while the picker data still loads after a GitHub round trip. */
  loading?: boolean;
  installRequested?: boolean;
  githubOrganization?: string;
  pendingInstallations?: PendingGitHubInstallation[];
  githubState?: string;
  githubAppSlug?: string;
  /** GitHub login that authorized this connect. Shown only with the account picker. */
  githubLogin?: string;
  bindingInstallationId?: string;
  connectError?: string;
  chrome?: FirstRunChrome;
  onConnectGitHub: () => void;
  onUseInstallation?: (installation: PendingGitHubInstallation) => void;
}) {
  const { showAccountPicker, waitingForApproval } = connectScreenState({
    installRequested,
    githubOrganization,
    pendingInstallations,
    githubState,
  });
  return (
    <FirstRunShell testId="first-run-connect" chrome={chrome}>
      <FirstRunHeading headline={showAccountPicker ? copy.selectAccount : copy.headline}>
        <p className="text-[15px] leading-6 text-muted-foreground">
          {showAccountPicker ? copy.selectAccountBody : copy.body}
        </p>
        {showAccountPicker && githubLogin ? <SignedInAsLine login={githubLogin} /> : null}
      </FirstRunHeading>

      <div className="mt-8 space-y-6">
        {loading ? (
          <ConnectScreenLoading />
        ) : (
          <ConnectScreenBody
            githubOrganization={githubOrganization}
            pendingInstallations={pendingInstallations}
            githubState={githubState}
            githubAppSlug={githubAppSlug}
            bindingInstallationId={bindingInstallationId}
            showAccountPicker={showAccountPicker}
            waitingForApproval={waitingForApproval}
            onConnectGitHub={onConnectGitHub}
            onUseInstallation={onUseInstallation}
          />
        )}
        {showAccountPicker ? null : <p className="text-[12px] text-muted-foreground">{copy.trust}</p>}
        {connectError && !waitingForApproval ? <p className="text-[13px] text-destructive">{connectError}</p> : null}
      </div>
    </FirstRunShell>
  );
}

/**
 * Placeholder while the picker data loads after a GitHub round trip, so the
 * screen does not flash the connect button before the picker.
 */
function ConnectScreenLoading() {
  return (
    <div className="space-y-3" data-testid="first-run-connect-loading" aria-hidden>
      <div className="h-14 animate-pulse rounded-md bg-accent/40" />
      <div className="h-9 w-40 animate-pulse rounded-md bg-accent/40" />
    </div>
  );
}

function ConnectScreenBody({
  githubOrganization,
  pendingInstallations,
  githubState,
  githubAppSlug,
  bindingInstallationId,
  showAccountPicker,
  waitingForApproval,
  onConnectGitHub,
  onUseInstallation,
}: {
  githubOrganization: string;
  pendingInstallations: PendingGitHubInstallation[];
  githubState: string;
  githubAppSlug: string;
  bindingInstallationId?: string;
  showAccountPicker: boolean;
  waitingForApproval: boolean;
  onConnectGitHub: () => void;
  onUseInstallation?: (installation: PendingGitHubInstallation) => void;
}) {
  if (showAccountPicker && onUseInstallation) {
    return (
      <FirstRunGitHubAccountPicker
        installations={pendingInstallations}
        githubAppSlug={githubAppSlug}
        githubState={githubState}
        githubOrganization={githubOrganization}
        waitingForApproval={waitingForApproval}
        bindingInstallationId={bindingInstallationId}
        onUseInstallation={onUseInstallation}
      />
    );
  }

  return (
    <>
      {waitingForApproval ? <FirstRunInstallRequested githubOrganization={githubOrganization} /> : null}
      <Button type="button" className="min-w-40" onClick={onConnectGitHub} data-testid="first-run-connect-github">
        {copy.connectGitHub}
      </Button>
    </>
  );
}
