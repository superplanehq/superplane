import { createContext } from "react";
import type { ResolvedTheme, ThemePreference } from "@/lib/themePreference";

export type ThemeContextType = {
  preference: ThemePreference;
  resolvedTheme: ResolvedTheme;
  setPreference: (preference: ThemePreference) => void;
};

export const ThemeContext = createContext<ThemeContextType | null>(null);
