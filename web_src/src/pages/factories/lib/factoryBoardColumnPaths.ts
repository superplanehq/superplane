/** Active column on the phone line board. Value is a column key, not an index. */
export const BOARD_COLUMN_SEARCH_PARAM = "column";

/** Appends the board column key to a path. Returns the path unchanged when no key is given. */
export function withBoardColumn(path: string, columnKey?: string | null): string {
  const key = columnKey?.trim();
  if (!key) {
    return path;
  }
  const separator = path.includes("?") ? "&" : "?";
  return `${path}${separator}${new URLSearchParams({ [BOARD_COLUMN_SEARCH_PARAM]: key }).toString()}`;
}

export function boardColumnKeyFromSearch(search: string): string | null {
  const query = search.startsWith("?") ? search.slice(1) : search;
  return new URLSearchParams(query).get(BOARD_COLUMN_SEARCH_PARAM)?.trim() || null;
}
