import { useCallback, useState, type ReactNode } from "react";
import { persistBoardColorTheme, readBoardColorTheme, type BoardColorTheme } from "./boardColorThemes";
import "./boardColorThemes.css";

import { BoardColorThemeContext } from "./boardColorThemeContext";

export function BoardColorThemeProvider({ boardKey, children }: { boardKey: string; children: ReactNode }) {
  return (
    <BoardColorThemeState key={boardKey} boardKey={boardKey}>
      {children}
    </BoardColorThemeState>
  );
}

function BoardColorThemeState({ boardKey, children }: { boardKey: string; children: ReactNode }) {
  const [theme, updateTheme] = useState(() => readBoardColorTheme(boardKey));
  const setTheme = useCallback(
    (next: BoardColorTheme) => {
      updateTheme(next);
      persistBoardColorTheme(boardKey, next);
    },
    [boardKey],
  );
  return <BoardColorThemeContext.Provider value={{ theme, setTheme }}>{children}</BoardColorThemeContext.Provider>;
}
