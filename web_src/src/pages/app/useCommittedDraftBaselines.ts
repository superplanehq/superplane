import type { CanvasesCanvas } from "@/api-client";
import { useEffect, useState } from "react";

import type { ConsoleLayoutItem, ConsolePanel } from "@/hooks/useCanvasData";

import { fetchCanvasVersionWithSpec, fetchConsoleSpecFromRepository } from "./lib/repository-spec-files";

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
  const [baselines, setBaselines] = useState<CommittedDraftBaselines>({ ready: false });

  useEffect(() => {
    if (!enabled || !canvasId || !versionId) {
      setBaselines({ ready: false });
      return;
    }

    let cancelled = false;
    setBaselines({ ready: false });

    void Promise.all([
      fetchCanvasVersionWithSpec(canvasId, versionId, false),
      fetchConsoleSpecFromRepository(canvasId, versionId, false),
    ]).then(([version, consoleSpec]) => {
      if (cancelled) {
        return;
      }

      setBaselines({
        canvasSpec: version?.spec,
        console: consoleSpec
          ? {
              panels: consoleSpec.panels,
              layout: consoleSpec.layout,
            }
          : { panels: [], layout: [] },
        ready: true,
      });
    });

    return () => {
      cancelled = true;
    };
  }, [canvasId, enabled, stagingResetNonce, versionId]);

  return baselines;
}
