import { LoadingButton } from "@/components/ui/loading-button";
import { useState, type FormEvent } from "react";

import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunGithubStepper } from "./FirstRunGithubStepper";
import { FirstRunHeading, FirstRunShell } from "./FirstRunShell";
import type { FirstRunSphereProps } from "./FirstRunSpherePane";
import type { FirstRunChrome } from "./firstRunTypes";

const copy = FIRST_RUN_COPY.connect;

function connectIntro(addLogin: boolean, createApp: boolean): string {
  if (addLogin) return copy.addLoginBody;
  if (createApp) return copy.createAppBody;
  return copy.body;
}

export function FirstRunConnectScreen({
  loading = false,
  connecting = false,
  createApp = false,
  addLogin = false,
  savingLogin = false,
  loginError,
  connectError,
  chrome,
  sphere,
  onConnectGitHub,
  onSaveLogin,
}: {
  loading?: boolean;
  connecting?: boolean;
  createApp?: boolean;
  addLogin?: boolean;
  savingLogin?: boolean;
  loginError?: string;
  connectError?: string;
  chrome?: FirstRunChrome;
  sphere?: FirstRunSphereProps;
  onConnectGitHub: () => void;
  onSaveLogin?: (clientId: string, clientSecret: string) => void;
}) {
  const [clientId, setClientId] = useState("");
  const [clientSecret, setClientSecret] = useState("");
  const canSaveLogin = clientId.trim() !== "" && clientSecret.trim() !== "";
  const saveLogin = (event: FormEvent) => {
    event.preventDefault();
    if (!canSaveLogin || !onSaveLogin) return;
    onSaveLogin(clientId.trim(), clientSecret.trim());
  };

  return (
    <FirstRunShell
      testId="first-run-connect"
      chrome={chrome}
      busy={loading || connecting}
      sphere={sphere}
      visual="preview"
    >
      <FirstRunHeading headline={copy.headline}>
        <p className="text-[15px] leading-6 text-muted-foreground">{connectIntro(addLogin, createApp)}</p>
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
              addLogin ? (
                <LoadingButton
                  type="submit"
                  form="first-run-github-login"
                  size="sm"
                  loading={savingLogin}
                  loadingText={copy.savingLogin}
                  disabled={!canSaveLogin}
                  data-testid="first-run-save-github-login"
                >
                  {copy.saveLoginAction}
                </LoadingButton>
              ) : (
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
              )
            }
          >
            {addLogin ? (
              <form id="first-run-github-login" className="w-full space-y-3" onSubmit={saveLogin}>
                <label className="block space-y-1.5 text-[13px] text-muted-foreground">
                  {copy.clientIdLabel}
                  <input
                    className="h-9 w-full rounded-md border border-[#34322b] bg-[#201f1a] px-3 text-[14px] text-foreground"
                    autoComplete="off"
                    value={clientId}
                    onChange={(event) => setClientId(event.target.value)}
                    data-testid="first-run-github-client-id"
                  />
                </label>
                <label className="block space-y-1.5 text-[13px] text-muted-foreground">
                  {copy.clientSecretLabel}
                  <input
                    className="h-9 w-full rounded-md border border-[#34322b] bg-[#201f1a] px-3 text-[14px] text-foreground"
                    type="password"
                    autoComplete="off"
                    value={clientSecret}
                    onChange={(event) => setClientSecret(event.target.value)}
                    data-testid="first-run-github-client-secret"
                  />
                </label>
              </form>
            ) : null}
          </FirstRunGithubStepper>
        )}
        {loginError ? <p className="text-[13px] text-destructive">{loginError}</p> : null}
        {connectError ? <p className="text-[13px] text-destructive">{connectError}</p> : null}
      </div>
    </FirstRunShell>
  );
}
