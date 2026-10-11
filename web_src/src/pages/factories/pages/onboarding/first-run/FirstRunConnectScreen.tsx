import { LoadingButton } from "@/components/ui/loading-button";

import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunGithubStepper } from "./FirstRunGithubStepper";
import { FirstRunHeading, FirstRunShell } from "./FirstRunShell";
import type { FirstRunSphereProps } from "./FirstRunSpherePane";
import type { FirstRunChrome } from "./firstRunTypes";

const copy = FIRST_RUN_COPY.connect;

function connectIntro(createApp: boolean): string {
  if (createApp) return copy.createAppBody;
  return copy.body;
}

function ConnectStepper({
  connecting,
  createApp,
  onConnectGitHub,
}: {
  connecting: boolean;
  createApp: boolean;
  onConnectGitHub: () => void;
}) {
  return (
    <FirstRunGithubStepper
      current="connect"
      action={
        <LoadingButton
          type="button"
          size="sm"
          onClick={onConnectGitHub}
          loading={connecting}
          loadingText={createApp ? copy.creatingGitHubApp : copy.openingGitHub}
          data-testid="first-run-connect-github"
        >
          {createApp ? copy.createAppAction : copy.connectAction}
        </LoadingButton>
      }
    />
  );
}

export function FirstRunConnectScreen({
  loading = false,
  connecting = false,
  createApp = false,
  connectError,
  chrome,
  sphere,
  onConnectGitHub,
}: {
  loading?: boolean;
  connecting?: boolean;
  createApp?: boolean;
  connectError?: string;
  chrome?: FirstRunChrome;
  sphere?: FirstRunSphereProps;
  onConnectGitHub: () => void;
}) {
  return (
    <FirstRunShell
      testId="first-run-connect"
      chrome={chrome}
      busy={loading || connecting}
      sphere={sphere}
      visual="preview"
    >
      <FirstRunHeading headline={copy.headline}>
        <p className="text-[15px] leading-6 text-muted-foreground">{connectIntro(createApp)}</p>
      </FirstRunHeading>

      <div className="mt-8 space-y-6">
        {loading ? (
          <p className="text-[13px] text-muted-foreground" role="status">
            {copy.loadingAccounts}
          </p>
        ) : (
          <ConnectStepper connecting={connecting} createApp={createApp} onConnectGitHub={onConnectGitHub} />
        )}
        {connectError ? <p className="text-[13px] text-destructive">{connectError}</p> : null}
      </div>
    </FirstRunShell>
  );
}
