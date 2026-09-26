import { useEffect, useRef, useState } from "react";

import type { OrganizationsIntegration } from "@/api-client";
import {
  GITHUB_SETUP_COMPLETE_VALUE,
  GITHUB_SETUP_INTEGRATION_PARAM,
  GITHUB_SETUP_ORG_PARAM,
  GITHUB_SETUP_REQUEST_PARAM,
  GITHUB_SETUP_REQUEST_VALUE,
} from "@/lib/integrationSetupReturn";

import { FIRST_RUN_COPY } from "./first-run/firstRunCopy";

type GitHubCallbackSyncState = {
  key: string;
  status: "loading" | "success" | "error";
};

type SetSearchParams = (searchParams: URLSearchParams, options: { replace: boolean }) => void;
type GitHubConnectionSync = (integrationId: string) => Promise<OrganizationsIntegration>;
type StartedSync = { key: string; promise: ReturnType<GitHubConnectionSync> };

export function useGitHubCallbackSync({
  searchParams,
  setSearchParams,
  integrationId,
  syncGithubConnection,
}: {
  searchParams: URLSearchParams;
  setSearchParams: SetSearchParams;
  integrationId?: string;
  syncGithubConnection: GitHubConnectionSync;
}) {
  const setup = searchParams.get(GITHUB_SETUP_REQUEST_PARAM);
  const callbackKind =
    setup === GITHUB_SETUP_REQUEST_VALUE || setup === GITHUB_SETUP_COMPLETE_VALUE ? setup : undefined;
  const callbackKey = callbackKind && integrationId ? `${callbackKind}:${integrationId}` : "";
  const startedSync = useRef<StartedSync | undefined>(undefined);
  const [attempt, setAttempt] = useState(0);
  const [state, setState] = useState<GitHubCallbackSyncState>();

  useEffect(() => {
    if (!callbackKey || !integrationId) {
      startedSync.current = undefined;
      return;
    }
    const attemptKey = `${callbackKey}:${attempt}`;
    if (startedSync.current?.key !== attemptKey) {
      startedSync.current = { key: attemptKey, promise: syncGithubConnection(integrationId) };
    }
    const sync = startedSync.current.promise;

    let cancelled = false;
    setState({ key: callbackKey, status: "loading" });
    void sync.then(
      () => {
        if (!cancelled) setState({ key: callbackKey, status: "success" });
      },
      () => {
        if (!cancelled) setState({ key: callbackKey, status: "error" });
      },
    );
    return () => {
      cancelled = true;
    };
  }, [attempt, callbackKey, integrationId, syncGithubConnection]);

  // The synchronization updates the connected-integrations cache before its
  // promise resolves. Success is therefore the point at which the callback
  // marker is safe to remove; object identity is not stable because React
  // Query can structurally share cached response objects.
  const applied = state?.status === "success" && state.key === callbackKey;
  useEffect(() => {
    if (!callbackKey || !applied) return;
    const next = new URLSearchParams(searchParams);
    clearGitHubSetupParams(next);
    setSearchParams(next, { replace: true });
  }, [applied, callbackKey, searchParams, setSearchParams]);

  const stateMatches = state?.key === callbackKey;
  return {
    active: callbackKey !== "",
    loading: callbackKey !== "" && (!stateMatches || state.status === "loading" || state.status === "success"),
    error: stateMatches && state.status === "error" ? FIRST_RUN_COPY.connect.refreshError : undefined,
    retry: () => setAttempt((current) => current + 1),
  };
}

export function clearGitHubSetupParams(searchParams: URLSearchParams) {
  searchParams.delete(GITHUB_SETUP_REQUEST_PARAM);
  searchParams.delete(GITHUB_SETUP_ORG_PARAM);
  searchParams.delete(GITHUB_SETUP_INTEGRATION_PARAM);
}
