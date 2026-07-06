import type { CanvasesCanvas } from "@/api-client";
import { useEffect, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";

import { canvasKeys, fetchCanvasConsoleData } from "@/hooks/useCanvasData";
import type { ConsoleLayoutItem, ConsolePanel } from "@/hooks/useCanvasData";

import { fetchLiveCommittedCanvasVersionWithSpec } from "./lib/repository-spec-files";

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

    // Read the live committed canvas and console (no stage=true). Staged/effective
    // reads use ?stage=true, so baselines must use the live endpoint — not
    // version_id — or local diffs can false-negative after the repository API split.
    void Promise.all([
      queryClient.fetchQuery({
        queryKey: [...canvasKeys.versionDetail(canvasId, versionId), "committed-baseline"] as const,
        queryFn: () => fetchLiveCommittedCanvasVersionWithSpec(canvasId),
        staleTime: 30_000,
      }),
      queryClient.fetchQuery({
        queryKey: [...canvasKeys.console(canvasId, versionId), "committed-baseline"] as const,
        queryFn: () => fetchCanvasConsoleData(canvasId, undefined, false),
        staleTime: 30_000,
      }),
    ]).then(([version, consoleData]) => {
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
    });

    return () => {
      cancelled = true;
    };
  }, [canvasId, enabled, queryClient, stagingResetNonce, versionId]);

  return baselines;
}
