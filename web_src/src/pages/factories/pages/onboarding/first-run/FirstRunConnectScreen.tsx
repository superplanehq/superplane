import { Button } from "@/components/ui/button";
import { Check, Clock } from "lucide-react";

import { IntegrationChoiceIcon } from "../onboardingSteps";
import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunHeading, FirstRunShell } from "./FirstRunShell";
import type { FirstRunChrome } from "./firstRunTypes";

export function FirstRunConnectScreen({
  githubConnected,
  installRequested = false,
  connectError,
  chrome,
  onConnectGitHub,
  onContinue,
}: {
  githubConnected: boolean;
  installRequested?: boolean;
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
              <div
                className="flex items-center gap-3 rounded-md bg-accent/40 px-3 py-2.5 text-left"
                data-testid="first-run-github-install-requested"
              >
                <IntegrationChoiceIcon name="github" />
                <span className="text-[13px] font-medium">{copy.installRequested}</span>
                <Clock className="ml-auto size-3.5 text-muted-foreground" aria-hidden />
              </div>
            ) : null}
            <Button type="button" className="min-w-40" onClick={onConnectGitHub} data-testid="first-run-connect-github">
              {copy.connectGitHub}
            </Button>
          </>
        )}
        <p className="text-[13px] text-muted-foreground">{copy.trust}</p>
        {waitingForApproval ? (
          <>
            <p className="text-[13px] text-muted-foreground">{copy.installRequestedBody}</p>
            <p className="text-[13px] text-muted-foreground">{copy.installRequestedNext}</p>
          </>
        ) : null}
        {connectError && !waitingForApproval ? <p className="text-[13px] text-destructive">{connectError}</p> : null}
      </div>
    </FirstRunShell>
  );
}
