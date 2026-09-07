import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Check, Clock } from "lucide-react";

import { hostedGitHubInstallURL, type PendingGitHubInstallation } from "@/lib/hostedGitHubInstall";

import { ConnectOptionRow, IntegrationChoiceIcon } from "../onboardingSteps";
import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunHeading, FirstRunPanel, FirstRunShell } from "./FirstRunShell";
import type { FirstRunChrome } from "./firstRunTypes";

export type FirstRunGithubConnection = {
  id: string;
  name: string;
  owner?: string;
};

const copy = FIRST_RUN_COPY.connect;

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

function FirstRunGitHubAccountPicker({
  installations,
  githubAppSlug,
  githubState,
  bindingInstallationId,
  onUseInstallation,
}: {
  installations: PendingGitHubInstallation[];
  githubAppSlug: string;
  githubState: string;
  bindingInstallationId?: string;
  onUseInstallation: (installation: PendingGitHubInstallation) => void;
}) {
  const binding = bindingInstallationId !== undefined;
  return (
    <div className="space-y-3" data-testid="first-run-github-account-picker">
      <FirstRunPanel>
        <p className="text-[13px] font-medium">{copy.selectAccount}</p>
        <p className="mt-0.5 text-[13px] text-muted-foreground">{copy.selectAccountBody}</p>
      </FirstRunPanel>
      {installations.map((installation) => (
        <LoadingButton
          key={installation.id}
          type="button"
          className="w-full justify-start"
          data-testid={`first-run-github-use-${installation.accountLogin}`}
          loading={bindingInstallationId === installation.id}
          disabled={binding}
          onClick={() => onUseInstallation(installation)}
        >
          {copy.useAccount(installation.accountLogin)}
        </LoadingButton>
      ))}
      {githubAppSlug !== "" ? (
        <a
          href={hostedGitHubInstallURL(githubAppSlug, githubState)}
          className="inline-block text-[13px] text-primary hover:underline"
          data-testid="first-run-github-install-other"
        >
          {copy.installDifferentAccount}
        </a>
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
  const showAccountPicker = pendingInstallations.length >= 1 && githubState !== "";
  // A picker that offers the requested organization means the request is
  // approved, so the waiting state must not show next to it.
  const requestApproved = pendingInstallations.some(
    (installation) => installation.accountLogin.toLowerCase() === githubOrganization.toLowerCase(),
  );
  return {
    showAccountPicker,
    waitingForApproval: installRequested && !(showAccountPicker && requestApproved),
  };
}

function connectionLabel(connection: FirstRunGithubConnection): string {
  return connection.owner || connection.name || copy.connected;
}

export function FirstRunConnectScreen({
  githubConnected,
  installRequested = false,
  githubOrganization = "",
  pendingInstallations = [],
  githubState = "",
  githubAppSlug = "",
  bindingInstallationId,
  readyConnections = [],
  selectedConnectionId,
  connectError,
  chrome,
  onConnectGitHub,
  onUseInstallation,
  onSelectConnection,
  onConnectAnother,
  onContinue,
}: {
  githubConnected: boolean;
  installRequested?: boolean;
  githubOrganization?: string;
  pendingInstallations?: PendingGitHubInstallation[];
  githubState?: string;
  githubAppSlug?: string;
  bindingInstallationId?: string;
  readyConnections?: FirstRunGithubConnection[];
  selectedConnectionId?: string;
  connectError?: string;
  chrome?: FirstRunChrome;
  onConnectGitHub: () => void;
  onUseInstallation?: (installation: PendingGitHubInstallation) => void;
  onSelectConnection?: (id: string) => void;
  onConnectAnother?: () => void;
  onContinue: () => void;
}) {
  const { showAccountPicker, waitingForApproval } = connectScreenState({
    installRequested,
    githubOrganization,
    pendingInstallations,
    githubState,
  });

  return (
    <FirstRunShell testId="first-run-connect" chrome={chrome}>
      <FirstRunHeading headline={copy.headline}>
        <p className="text-[15px] leading-6 text-muted-foreground">{copy.body}</p>
      </FirstRunHeading>

      <div className="mt-8 space-y-3">
        <ConnectScreenBody
          githubConnected={githubConnected}
          githubOrganization={githubOrganization}
          pendingInstallations={pendingInstallations}
          githubState={githubState}
          githubAppSlug={githubAppSlug}
          bindingInstallationId={bindingInstallationId}
          readyConnections={readyConnections}
          selectedConnectionId={selectedConnectionId}
          showAccountPicker={showAccountPicker}
          waitingForApproval={waitingForApproval}
          onConnectGitHub={onConnectGitHub}
          onUseInstallation={onUseInstallation}
          onSelectConnection={onSelectConnection}
          onConnectAnother={onConnectAnother}
          onContinue={onContinue}
        />
        <p className="text-[13px] text-muted-foreground">{copy.trust}</p>
        {connectError && !waitingForApproval ? <p className="text-[13px] text-destructive">{connectError}</p> : null}
      </div>
    </FirstRunShell>
  );
}

function ConnectScreenBody({
  githubConnected,
  githubOrganization,
  pendingInstallations,
  githubState,
  githubAppSlug,
  bindingInstallationId,
  readyConnections,
  selectedConnectionId,
  showAccountPicker,
  waitingForApproval,
  onConnectGitHub,
  onUseInstallation,
  onSelectConnection,
  onConnectAnother,
  onContinue,
}: {
  githubConnected: boolean;
  githubOrganization: string;
  pendingInstallations: PendingGitHubInstallation[];
  githubState: string;
  githubAppSlug: string;
  bindingInstallationId?: string;
  readyConnections: FirstRunGithubConnection[];
  selectedConnectionId?: string;
  showAccountPicker: boolean;
  waitingForApproval: boolean;
  onConnectGitHub: () => void;
  onUseInstallation?: (installation: PendingGitHubInstallation) => void;
  onSelectConnection?: (id: string) => void;
  onConnectAnother?: () => void;
  onContinue: () => void;
}) {
  if (showAccountPicker && onUseInstallation) {
    return (
      <>
        {waitingForApproval ? <FirstRunInstallRequested githubOrganization={githubOrganization} /> : null}
        <FirstRunGitHubAccountPicker
          installations={pendingInstallations}
          githubAppSlug={githubAppSlug}
          githubState={githubState}
          bindingInstallationId={bindingInstallationId}
          onUseInstallation={onUseInstallation}
        />
      </>
    );
  }

  if (waitingForApproval) {
    return (
      <>
        <FirstRunInstallRequested githubOrganization={githubOrganization} />
        {githubConnected ? (
          <Button type="button" className="min-w-40" onClick={onContinue} data-testid="first-run-github-continue">
            {copy.continue}
          </Button>
        ) : (
          <Button type="button" className="min-w-40" onClick={onConnectGitHub} data-testid="first-run-connect-github">
            {copy.connectGitHub}
          </Button>
        )}
      </>
    );
  }

  if (readyConnections.length > 0) {
    return (
      <>
        <FirstRunPanel>
          <div className="space-y-3" data-testid="first-run-github-connections">
            {readyConnections.map((connection) => (
              <ConnectOptionRow
                key={connection.id}
                icon={<IntegrationChoiceIcon name="github" />}
                title={connectionLabel(connection)}
                detail={copy.connected}
                selected={connection.id === selectedConnectionId}
                onSelect={() => onSelectConnection?.(connection.id)}
              />
            ))}
          </div>
        </FirstRunPanel>
        <Button
          type="button"
          className="min-w-40"
          disabled={!selectedConnectionId}
          onClick={onContinue}
          data-testid="first-run-github-continue"
        >
          {copy.continue}
        </Button>
        <ConnectAnotherButton onConnectAnother={onConnectAnother} />
      </>
    );
  }

  if (githubConnected) {
    return (
      <>
        <div
          className="flex items-center gap-3 rounded-md bg-accent/40 px-3 py-2.5 text-left"
          data-testid="first-run-github-connected"
        >
          <IntegrationChoiceIcon name="github" />
          <span className="text-[13px] font-medium">{copy.connected}</span>
          <Check className="ml-auto size-3.5" strokeWidth={2.5} aria-hidden />
        </div>
        <Button type="button" className="min-w-40" onClick={onContinue} data-testid="first-run-github-continue">
          {copy.continue}
        </Button>
        <ConnectAnotherButton onConnectAnother={onConnectAnother} />
      </>
    );
  }

  return (
    <Button type="button" className="min-w-40" onClick={onConnectGitHub} data-testid="first-run-connect-github">
      {copy.connectGitHub}
    </Button>
  );
}

function ConnectAnotherButton({ onConnectAnother }: { onConnectAnother?: () => void }) {
  if (!onConnectAnother) {
    return null;
  }

  return (
    <Button
      type="button"
      variant="ghost"
      className="text-[13px] text-primary hover:text-primary"
      onClick={onConnectAnother}
      data-testid="first-run-github-connect-another"
    >
      {copy.connectAnother}
    </Button>
  );
}
