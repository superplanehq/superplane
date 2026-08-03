import type { WidgetDataSourceKind, WidgetTableColumn } from "./types";

const RUN_ID_FIELD = "id";

/**
 * Whether a column prints a run id that should link to the run inspector.
 *
 * Only `runs` rows have an `id` that is a run id — `memory` and `executions`
 * rows have their own — and we only take over cells that would otherwise be
 * plain text, so an explicit `href` or a presentational format wins.
 */
export function isRunIdColumn(col: WidgetTableColumn, dataSourceKind: WidgetDataSourceKind | undefined): boolean {
  if (dataSourceKind !== "runs") return false;
  if (col.field !== RUN_ID_FIELD) return false;
  if (col.href) return false;
  return col.format === undefined || col.format === "text";
}
