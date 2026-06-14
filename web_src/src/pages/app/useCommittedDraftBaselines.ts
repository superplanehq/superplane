import type { CanvasesCanvas } from "@/api-client";
import { useEffect, useState } from "react";
import { isCancelledError, useQueryClient } from "@tanstack/react-query";

import { canvasKeys, fetchCanvasConsoleData } from "@/hooks/useCanvasData";
import type { ConsoleLayoutItem, ConsolePanel } from "@/hooks/useCanvasData";

import { fetchCanvasVersionWithSpec } from "./lib/repository-spec-files";

export type CommittedDraftBaselines = {
  canvasSpec?: CanvasesCanvas["spec"];
  console?: {
    panels: ConsolePanel[];
    layout: ConsoleLayoutItem[];
  };
  ready: boolean;
};

type UseCommittedDraftBaselinesOptions = {
  canvasId?: string;
  versionId?: string;
  enabled: boolean;
  /** Bumps after reset/commit remounts so committed snapshots reload from the server. */
  stagingResetNonce: number;
};

export function useCommittedDraftBaselines({
  canvasId,
  versionId,
  enabled,
  stagingResetNonce,
}: UseCommittedDraftBaselinesOptions): CommittedDraftBaselines {
  const queryClient = useQueryClient();
  const [baselines, setBaselines] = useState<CommittedDraftBaselines>({ ready: false });

  useEffect(() => {
    if (!enabled || !canvasId || !versionId) {
      setBaselines({ ready: false });
      return;
    }

    let cancelled = false;
    setBaselines({ ready: false });

    // Read the committed (stage=false) canvas and console through React Query so
    // the baselines reuse the cache the rest of the editor already populates.
    // The console read shares its key/fetcher with the draft console query, so
    // the two committed console.yaml reads are deduped into a single request.
    // Commit/discard invalidate these keys, so the nonce bump reloads fresh data.
    //
    // Wrapped in an async IIFE with try/catch so that CancelledError (raised
    // when an in-flight query is cancelled, e.g. by `queryClient.cancelQueries`
    // or rapid effect re-runs) does not surface as an unhandled rejection that
    // gets reported to Sentry.
    void (async () => {
      try {
        const [version, consoleData] = await Promise.all([
          queryClient.fetchQuery({
            queryKey: canvasKeys.versionDetail(canvasId, versionId),
            queryFn: () => fetchCanvasVersionWithSpec(canvasId, versionId, false),
            staleTime: 30_000,
          }),
          queryClient.fetchQuery({
            queryKey: canvasKeys.console(canvasId, versionId),
            queryFn: () => fetchCanvasConsoleData(canvasId, versionId, false),
            staleTime: 30_000,
          }),
        ]);

        if (cancelled) {
          return;
        }

        setBaselines({
          canvasSpec: version?.spec,
          console: consoleData
            ? {
                panels: consoleData.panels,
                layout: consoleData.layout,
              }
            : { panels: [], layout: [] },
          ready: true,
        });
      } catch (error) {
        if (cancelled || isCancelledError(error)) {
          return;
        }
        throw error;
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [canvasId, enabled, queryClient, stagingResetNonce, versionId]);

  return baselines;
}
