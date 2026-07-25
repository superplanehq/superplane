import type { CanvasesCanvas } from "@/api-client";
import { useEffect, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";

import { canvasKeys } from "@/hooks/useCanvasData";
import type { ConsoleLayoutItem, ConsolePanel } from "@/pages/app/lib/console-data";
import { parseConsoleDataFromVersion } from "@/pages/app/lib/console-data";

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

    void queryClient
      .fetchQuery({
        queryKey: canvasKeys.version(canvasId, versionId),
        queryFn: () => fetchCanvasVersionWithSpec(canvasId, versionId),
        staleTime: Number.POSITIVE_INFINITY,
      })
      .then((version) => {
        if (cancelled) {
          return;
        }

        const consoleData = parseConsoleDataFromVersion(canvasId, version);
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
