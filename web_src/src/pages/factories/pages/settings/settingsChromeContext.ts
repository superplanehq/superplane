import { createContext, useContext } from "react";

/** Storybook-only: factory settings use the redesigned chrome and pages. */
export const SettingsRedesignContext = createContext(false);

export function useSettingsRedesign() {
  return useContext(SettingsRedesignContext);
}
