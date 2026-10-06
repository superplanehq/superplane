import type { CanvasesCanvasRun, FactoriesWorkOrderCheck } from "@/api-client";
import { useCallback, useEffect, useMemo, useState, type ReactNode } from "react";

import { ColumnAppCheckRunQueries } from "./ColumnAppCheckRunQueries";
import {
  columnAppCheckRunsToDescribe,
  type ColumnAppCanvasRunLookup,
  type ColumnAppCheckRunTarget,
  type SplitRunColumnApp,
} from "./splitRunMocks";

const EMPTY_LOOKUP: ColumnAppCanvasRunLookup = {
  runsById: new Map(),
  loadingIds: new Set(),
};

type ColumnAppRunReport = {
  run?: CanvasesCanvasRun;
  loading: boolean;
};

/**
 * Describes each column-app check run, including runs already linked on the
 * pull request. Callers render `queries` so each run id gets one
 * `useDescribeRun` request. Duration stays blank while an unlinked request
 * is loading.
 */
export function useColumnAppCheckRuns(
  checks: FactoriesWorkOrderCheck[] | undefined,
  columnApps: SplitRunColumnApp[],
): { lookup: ColumnAppCanvasRunLookup; queries: ReactNode } {
  const targets = useMemo(() => columnAppCheckRunsToDescribe(columnApps, checks), [checks, columnApps]);
  const [reports, setReports] = useState<Record<string, ColumnAppRunReport>>({});
  const reportRun = useCallback((runId: string, run: CanvasesCanvasRun | undefined, loading: boolean) => {
    setReports((current) => {
      const previous = current[runId];
      if (previous?.loading === loading && previous.run === run) {
        return current;
      }
      return { ...current, [runId]: { run, loading } };
    });
  }, []);

  useEffect(() => {
    setReports((current) => {
      const liveIds = new Set(targets.map((target) => target.runId));
      const nextEntries = Object.entries(current).filter(([runId]) => liveIds.has(runId));
      if (nextEntries.length === Object.keys(current).length) {
        return current;
      }
      return Object.fromEntries(nextEntries);
    });
  }, [targets]);

  const lookup = useMemo(() => lookupFromReports(targets, reports), [reports, targets]);
  const queries = <ColumnAppCheckRunQueries targets={targets} onRun={reportRun} />;
  return { lookup, queries };
}

function lookupFromReports(
  targets: ColumnAppCheckRunTarget[],
  reports: Record<string, ColumnAppRunReport>,
): ColumnAppCanvasRunLookup {
  if (targets.length === 0) {
    return EMPTY_LOOKUP;
  }
  const runsById = new Map<string, CanvasesCanvasRun>();
  const loadingIds = new Set<string>();
  for (const target of targets) {
    const report = reports[target.runId];
    if (report?.run) {
      runsById.set(target.runId, report.run);
      continue;
    }
    if (!report || report.loading) {
      loadingIds.add(target.runId);
    }
  }
  return { runsById, loadingIds };
}
