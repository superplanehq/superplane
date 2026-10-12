import { createContext, useContext } from "react";
import type { BoardColorTheme } from "./boardColorThemes";

export const BoardColorThemeContext = createContext<{
  theme: BoardColorTheme;
  setTheme: (theme: BoardColorTheme) => void;
} | null>(null);

export function useBoardColorTheme() {
  return useContext(BoardColorThemeContext);
}

export function useBoardColorThemeName() {
  return useBoardColorTheme()?.theme ?? "default";
}
