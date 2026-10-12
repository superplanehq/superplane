export const BOARD_COLOR_THEMES = [
  { id: "default", label: "Default" },
  { id: "flow", label: "SuperPlane Flow" },
  { id: "quiet", label: "Quiet Brand" },
  { id: "clear", label: "Clear System" },
  { id: "pastel", label: "Clean Pastel" },
  { id: "spectrum", label: "Cool Spectrum" },
] as const;

export type BoardColorTheme = (typeof BOARD_COLOR_THEMES)[number]["id"];
export type BoardPaletteRole = "backlog" | "implement" | "verify" | "done";

export function boardPaletteRole(columnKey: string): BoardPaletteRole {
  return columnKey === "backlog" || columnKey === "verify" || columnKey === "done" ? columnKey : "implement";
}

export function readBoardColorTheme(boardKey: string): BoardColorTheme {
  try {
    const saved = localStorage.getItem(`factories-board-theme:${boardKey}`);
    return BOARD_COLOR_THEMES.find((theme) => theme.id === saved)?.id ?? "default";
  } catch {
    return "default";
  }
}

export function persistBoardColorTheme(boardKey: string, theme: BoardColorTheme) {
  try {
    const key = `factories-board-theme:${boardKey}`;
    if (theme === "default") localStorage.removeItem(key);
    else localStorage.setItem(key, theme);
  } catch {
    // Keep the selection usable when browser storage is unavailable.
  }
}
