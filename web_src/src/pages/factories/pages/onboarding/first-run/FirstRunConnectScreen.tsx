import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Check, Clock } from "lucide-react";

import {
  hostedGitHubBindPath,
  hostedGitHubInstallURL,
  type PendingGitHubInstallation,
} from "@/lib/hostedGitHubInstall";

import { IntegrationChoiceIcon } from "../onboardingSteps";
import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunHeading, FirstRunPanel, FirstRunShell } from "./FirstRunShell";
import type { FirstRunChrome } from "./firstRunTypes";

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
  githubState,
  githubAppSlug,
  installRequested,
}: {
  installations: PendingGitHubInstallation[];
  githubState: string;
  githubAppSlug: string;
  installRequested: boolean;
}) {
  return (
    <div className="space-y-3" data-testid="first-run-github-account-picker">
      <FirstRunPanel>
        <p className="text-[13px] font-medium">{copy.selectAccount}</p>
        <p className="mt-0.5 text-[13px] text-muted-foreground">{copy.selectAccountBody}</p>
      </FirstRunPanel>
      {installations.map((installation) => (
        <Button
          key={installation.id}
          type="button"
          className="w-full justify-start"
          data-testid={`first-run-github-use-${installation.accountLogin}`}
          onClick={() => {
            window.location.assign(hostedGitHubBindPath(githubState, installation.id));
          }}
        >
          {copy.useAccount(installation.accountLogin)}
        </Button>
      ))}
      {githubAppSlug !== "" && installRequested ? (
        <a
          href={hostedGitHubInstallURL(githubAppSlug, githubState)}
          className="inline-block text-[13px] text-primary hover:underline"
          data-testid="first-run-github-cancel-request"
        >
          {copy.cancelRequest}
        </a>
      ) : null}
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

export function FirstRunConnectScreen({
  githubConnected,
  installRequested = false,
  githubOrganization = "",
  pendingInstallations = [],
  githubState = "",
  githubAppSlug = "",
  connectError,
  chrome,
  onConnectGitHub,
  onContinue,
}: {
  githubConnected: boolean;
  installRequested?: boolean;
  githubOrganization?: string;
  pendingInstallations?: PendingGitHubInstallation[];
  githubState?: string;
  githubAppSlug?: string;
  connectError?: string;
  chrome?: FirstRunChrome;
  onConnectGitHub: () => void;
  onContinue: () => void;
}) {
  const waitingForApproval = installRequested && !githubConnected;
  const showAccountPicker =
    !githubConnected &&
    githubState !== "" &&
    (pendingInstallations.length >= 2 || (pendingInstallations.length >= 1 && installRequested));

  return (
    <FirstRunShell testId="first-run-connect" chrome={chrome}>
      <FirstRunHeading headline={copy.headline}>
        <p className="text-[15px] leading-6 text-muted-foreground">{copy.body}</p>
      </FirstRunHeading>

      <div className="mt-8 space-y-3">
        {githubConnected ? (
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
            {githubAppSlug !== "" ? (
              <a
                href={hostedGitHubInstallURL(githubAppSlug, githubState)}
                className="inline-block text-[13px] text-primary hover:underline"
                data-testid="first-run-github-manage-accounts"
              >
                {copy.manageAccounts}
              </a>
            ) : null}
          </>
        ) : (
          <>
            {waitingForApproval ? <FirstRunInstallRequested githubOrganization={githubOrganization} /> : null}
            {showAccountPicker ? (
              <FirstRunGitHubAccountPicker
                installations={pendingInstallations}
                githubState={githubState}
                githubAppSlug={githubAppSlug}
                installRequested={installRequested}
              />
            ) : (
              <Button
                type="button"
                className="min-w-40"
                onClick={onConnectGitHub}
                data-testid="first-run-connect-github"
              >
                {copy.connectGitHub}
              </Button>
            )}
          </>
        )}
        <p className="text-[13px] text-muted-foreground">{copy.trust}</p>
        {connectError && !waitingForApproval ? <p className="text-[13px] text-destructive">{connectError}</p> : null}
      </div>
    </FirstRunShell>
  );
}
