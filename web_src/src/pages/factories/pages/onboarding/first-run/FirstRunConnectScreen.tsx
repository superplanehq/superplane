import { LoadingButton } from "@/components/ui/loading-button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Clock } from "lucide-react";

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
              <span className="min-w-0 truncate font-medium">Waiting for approval for {organization}.</span>
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

export function FirstRunConnectScreen({
  loading = false,
  identityConnected = false,
  githubLogin = "",
  pendingOrganizations = [],
  synchronizing = false,
  appConfigured = true,
  connecting = false,
  connectError,
  chrome,
  sphere,
  onConnectGitHub,
}: {
  loading?: boolean;
  identityConnected?: boolean;
  githubLogin?: string;
  pendingOrganizations?: string[];
  synchronizing?: boolean;
  appConfigured?: boolean;
  connecting?: boolean;
  connectError?: string;
  chrome?: FirstRunChrome;
  sphere?: FirstRunSphereProps;
  onConnectGitHub: () => void;
}) {
  const current = identityConnected ? "grant" : "connect";
  const headline = identityConnected ? copy.grantHeadline : copy.headline;
  const body = identityConnected ? copy.grantBody : copy.body;
  const grantStatus = (
    <>
      {pendingOrganizations.length > 0 ? <PendingApprovalRows organizations={pendingOrganizations} /> : null}
      {synchronizing ? (
        <p className="text-[13px] text-muted-foreground" role="status">
          {copy.synchronizing}
        </p>
      ) : null}
    </>
  );

  return (
    <FirstRunShell testId="first-run-connect" chrome={chrome} busy={loading || connecting} sphere={sphere}>
      <FirstRunHeading headline={headline}>
        <p className="text-[15px] leading-6 text-muted-foreground">{body}</p>
        {identityConnected && githubLogin ? <SignedInAsLine login={githubLogin} /> : null}
      </FirstRunHeading>

      <div className="mt-8 space-y-6">
        {loading ? (
          <p className="text-[13px] text-muted-foreground" role="status">
            {copy.loadingAccounts}
          </p>
        ) : (
          <FirstRunGithubStepper
            current={current}
            grantStatus={grantStatus}
            action={
              <LoadingButton
                type="button"
                size="sm"
                onClick={onConnectGitHub}
                loading={connecting}
                loadingText={copy.openingGitHub}
                disabled={!appConfigured && identityConnected}
                data-testid="first-run-connect-github"
              >
                {identityConnected ? copy.grantAction : copy.connectAction}
              </LoadingButton>
            }
          />
        )}
        {connectError ? <p className="text-[13px] text-destructive">{connectError}</p> : null}
      </div>
    </FirstRunShell>
  );
}
