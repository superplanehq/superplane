import { factoryHomePath } from "./factoryPagePaths";

/**
 * Mobile board column in view. Value is a column key (`backlog`, `phase-0`,
 * `verify`, `done`). Only the phone board reads it.
 */
export const BOARD_COLUMN_SEARCH_PARAM = "column";

export function boardColumnKeyFromSearch(search: string): string | null {
  const query = search.startsWith("?") ? search.slice(1) : search;
  return new URLSearchParams(query).get(BOARD_COLUMN_SEARCH_PARAM);
}

/**
 * Line board path with the mobile column in view. Without a column key the
 * path is the same as `factoryHomePath`.
 */
export function factoryHomeColumnPath(
  organizationId: string,
  factoryKey: string,
  lineId: string,
  columnKey?: string | null,
) {
  const path = factoryHomePath(organizationId, factoryKey, lineId);
  if (!columnKey) {
    return path;
  }
  return `${path}?${new URLSearchParams({ [BOARD_COLUMN_SEARCH_PARAM]: columnKey }).toString()}`;
}
