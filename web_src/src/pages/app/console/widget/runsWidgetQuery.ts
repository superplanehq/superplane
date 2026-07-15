import { canvasKeys } from "@/hooks/useCanvasData";

import type { WidgetDataSource, WidgetRender } from "./types";

/**
 * Stable string identifier for the shared `useInfiniteCanvasRuns` cache
 * entry a widget participates in. Passed to `useEagerInfinitePagination`
 * so N widgets on the same key coordinate through a single-flight mutex
 * and can't race `fetchNextPage()` on the same cursor.
 *
 * We serialize the exact `canvasKeys.infiniteRuns` array so widgets on
 * different filter combinations (e.g. `{}` vs `states=STATE_STARTED`)
 * get distinct flight keys.
 */
export function makeRunsFlightKey(canvasId: string, filters: Parameters<typeof canvasKeys.infiniteRuns>[1]): string {
  return JSON.stringify(canvasKeys.infiniteRuns(canvasId, filters));
}

/**
 * True when a runs-backed number/scorecard render only needs the API
 * `totalCount` — i.e. a `count` aggregation with no row filters and no
 * sparkline. In that case the widget can skip eager pagination because
 * page 1 of the shared infinite query already carries `totalCount`.
 *
 * Table and chart renders always need rows, so this returns `false` for
 * them regardless of aggregation. Executions data sources are excluded
 * because their `totalCount` (when reported) is per-run, not per-execution.
 */
export function runsRenderIsTotalCountOnly(dataSource: WidgetDataSource, render: WidgetRender | undefined): boolean {
  if (dataSource.kind !== "runs") return false;
  if (!render) return false;
  if (render.kind !== "number" && render.kind !== "scorecard") return false;
  if (render.aggregation !== "count") return false;
  if (render.filters && render.filters.length > 0) return false;
  if ("sparklineField" in render && render.sparklineField) return false;
  return true;
}
