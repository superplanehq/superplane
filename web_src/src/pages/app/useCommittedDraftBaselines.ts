import type { CanvasesCanvas } from "@/api-client";
import { useEffect, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";

import { getCanvasVersionQueryOptions } from "@/hooks/useCanvasData";
import { consoleSpecFromCanvasSpec } from "./lib/repository-spec-files";
import type { ConsoleLayoutItem, ConsolePanel } from "@/hooks/useCanvasData";

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

    void queryClient.fetchQuery(getCanvasVersionQueryOptions(canvasId, versionId)).then((version) => {
      if (cancelled) {
        return;
      }

      const consoleSpec = consoleSpecFromCanvasSpec(canvasId, version?.spec);
      setBaselines({
        canvasSpec: version?.spec,
        console: {
          panels: consoleSpec.panels,
          layout: consoleSpec.layout,
        },
        ready: true,
      });
    });

    return () => {
      cancelled = true;
    };
  }, [canvasId, enabled, queryClient, stagingResetNonce, versionId]);

  return baselines;
}
