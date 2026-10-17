import { isColumnKey, type ColumnKey } from "./columnKeys";

/**
 * Active column on the phone board. Value is a column key (`backlog`,
 * `phase-N`, `verify`, or `done`). The phone board reads it on mount to open
 * the column the user was viewing before a task page.
 */
export const BOARD_COLUMN_SEARCH_PARAM = "column";

export function boardColumnKeyFromSearch(search: string): ColumnKey | null {
  const query = search.startsWith("?") ? search.slice(1) : search;
  const value = new URLSearchParams(query).get(BOARD_COLUMN_SEARCH_PARAM);
  return isColumnKey(value) ? value : null;
}
