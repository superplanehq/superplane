import type { TablePanelContent } from "../panelTypes";
import { useRowActionPayloadDrafts, type RowActionPayloadDrafts } from "./useRowActionPayloadDrafts";

/**
 * Table-flavored wrapper around {@link useRowActionPayloadDrafts}. Kept as a
 * dedicated entry point so table-form code reads naturally (`value.render`
 * is right there) and existing spec/import paths keep working. Any changes
 * to the underlying draft-tracking logic should live in
 * {@link useRowActionPayloadDrafts} so board and future row-action panels
 * stay in lockstep.
 */
export function useTablePanelPayloadDrafts(value: TablePanelContent) {
  return useRowActionPayloadDrafts(value.render.rowActions);
}

export type TablePanelPayloadDrafts = RowActionPayloadDrafts;
