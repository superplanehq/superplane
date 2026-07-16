import { useMemo } from "react";

import { renderNeedsRunNodeOutputs, useWidgetData } from "./useWidgetData";
import { useMemoryCatalog, sampleRowFromFields, type MemoryFieldSummary } from "./useMemoryCatalog";
import { EXECUTIONS_FIELDS, RUNS_FIELDS } from "./staticFieldCatalogs";
import type { WidgetDataSource, WidgetRender } from "./types";

// `live` — a real row was pulled off the data source.
// `catalog` — sample row synthesized from the field catalog (no live rows).
// `empty` — no data source, or catalog is empty.
export type WidgetExpressionContextOrigin = "live" | "catalog" | "empty";

export interface WidgetExpressionContextResult {
  row: Record<string, unknown>;
  origin: WidgetExpressionContextOrigin;
  isLoading: boolean;
  error?: string;
  fields: MemoryFieldSummary[];
}

const EMPTY_ROW: Record<string, unknown> = {};

// Resolve the expression context for a widget authoring surface: prefer
// the latest real row so hints and previews match runtime; otherwise fall
// back to a catalog-synthesized sample row so the editor still has
// something to autocomplete against.
export function useWidgetExpressionContext(args: {
  canvasId: string;
  dataSource: WidgetDataSource | undefined;
  render?: WidgetRender;
}): WidgetExpressionContextResult {
  const { canvasId, dataSource, render } = args;

  const needsNodeOutputs = useMemo(() => renderNeedsRunNodeOutputs(render), [render]);

  // Stable fallback so we can call `useWidgetData` unconditionally.
  const effectiveDataSource: WidgetDataSource = dataSource ?? { kind: "runs" };
  const { rows, isLoading, error } = useWidgetData(canvasId, effectiveDataSource, needsNodeOutputs);

  const memoryNamespace = dataSource?.kind === "memory" ? dataSource.namespace : undefined;
  const memoryCatalog = useMemoryCatalog(canvasId, memoryNamespace);

  return useMemo<WidgetExpressionContextResult>(() => {
    if (!dataSource) {
      return { row: EMPTY_ROW, origin: "empty", isLoading: false, fields: [] };
    }

    const fields = describeFields(dataSource, memoryCatalog.fields);
    const liveRow = rows.find((r): r is Record<string, unknown> => typeof r === "object" && r !== null);
    if (liveRow) return { row: liveRow, origin: "live", isLoading: false, fields, error };

    // While fetches are in flight, fall back to the catalog sample row so the
    // editor keeps its autocomplete and previews instead of blanking out.
    const isFetching = isLoading || memoryCatalog.isLoading;
    if (fields.length === 0) {
      return { row: EMPTY_ROW, origin: "empty", isLoading: isFetching, fields, error };
    }
    return {
      row: sampleRowFromFields(fields),
      origin: "catalog",
      isLoading: isFetching,
      fields,
      error,
    };
  }, [dataSource, rows, isLoading, memoryCatalog.fields, memoryCatalog.isLoading, error]);
}

function describeFields(dataSource: WidgetDataSource, memoryFields: MemoryFieldSummary[]): MemoryFieldSummary[] {
  if (dataSource.kind === "memory") return memoryFields;
  if (dataSource.kind === "executions") return EXECUTIONS_FIELDS;
  if (dataSource.kind === "runs") return RUNS_FIELDS;
  return [];
}
