import { LoadingButton } from "@/components/ui/loading-button";

import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunGithubStepper } from "./FirstRunGithubStepper";
import { FirstRunHeading, FirstRunShell } from "./FirstRunShell";
import type { FirstRunSphereProps } from "./FirstRunSpherePane";
import type { FirstRunChrome } from "./firstRunTypes";

const copy = FIRST_RUN_COPY.connect;

export function FirstRunConnectScreen({
  loading = false,
  connecting = false,
  connectError,
  chrome,
  sphere,
  onConnectGitHub,
}: {
  loading?: boolean;
  connecting?: boolean;
  connectError?: string;
  chrome?: FirstRunChrome;
  sphere?: FirstRunSphereProps;
  onConnectGitHub: () => void;
}) {
  return (
    <FirstRunShell testId="first-run-connect" chrome={chrome} busy={loading || connecting} sphere={sphere}>
      <FirstRunHeading headline={copy.headline}>
        <p className="text-[15px] leading-6 text-muted-foreground">{copy.body}</p>
      </FirstRunHeading>

      <div className="mt-8 space-y-6">
        {loading ? (
          <p className="text-[13px] text-muted-foreground" role="status">
            {copy.loadingAccounts}
          </p>
        ) : (
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
          />
        )}
        {connectError ? <p className="text-[13px] text-destructive">{connectError}</p> : null}
      </div>
    </FirstRunShell>
  );
}
