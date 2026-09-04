import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Check, CircleHelp, Clock } from "lucide-react";

import { IntegrationChoiceIcon } from "../onboardingSteps";
import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunHeading, FirstRunPanel, FirstRunShell } from "./FirstRunShell";
import type { FirstRunChrome } from "./firstRunTypes";

export function FirstRunConnectScreen({
  githubConnected,
  installRequested = false,
  githubOrganization = "",
  connectError,
  chrome,
  onConnectGitHub,
  onContinue,
}: {
  githubConnected: boolean;
  installRequested?: boolean;
  githubOrganization?: string;
  connectError?: string;
  chrome?: FirstRunChrome;
  onConnectGitHub: () => void;
  onContinue: () => void;
}) {
  const copy = FIRST_RUN_COPY.connect;
  const waitingForApproval = installRequested && !githubConnected;

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
          </>
        ) : (
          <>
            {waitingForApproval ? (
              <FirstRunPanel>
                <div className="flex items-start gap-3" data-testid="first-run-github-install-requested">
                  <IntegrationChoiceIcon name="github" />
                  <div className="min-w-0 flex-1 text-left">
                    <p className="text-[13px] font-medium">{copy.installRequested}</p>
                    {githubOrganization ? (
                      <p
                        className="mt-0.5 text-[13px] text-muted-foreground"
                        data-testid="first-run-github-install-org"
                      >
                        {githubOrganization}
                      </p>
                    ) : null}
                  </div>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <button
                        type="button"
                        className="mt-0.5 text-muted-foreground transition-colors hover:text-foreground"
                        aria-label={copy.installRequestedHelp}
                        data-testid="first-run-github-install-help"
                      >
                        <CircleHelp className="size-3.5" aria-hidden />
                      </button>
                    </TooltipTrigger>
                    <TooltipContent side="top" className="max-w-xs space-y-1 text-left">
                      <p>{copy.installRequestedBody(githubOrganization)}</p>
                      <p>{copy.installRequestedNext}</p>
                    </TooltipContent>
                  </Tooltip>
                  <Clock className="mt-0.5 size-3.5 shrink-0 text-muted-foreground" aria-hidden />
                </div>
              </FirstRunPanel>
            ) : null}
            <Button type="button" className="min-w-40" onClick={onConnectGitHub} data-testid="first-run-connect-github">
              {copy.connectGitHub}
            </Button>
          </>
        )}
        <p className="text-[13px] text-muted-foreground">{copy.trust}</p>
        {connectError && !waitingForApproval ? <p className="text-[13px] text-destructive">{connectError}</p> : null}
      </div>
    </FirstRunShell>
  );
}
