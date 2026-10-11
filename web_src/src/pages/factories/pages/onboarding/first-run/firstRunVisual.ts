import { createContext, useContext } from "react";

export type FirstRunVisual = "app" | "preview";

export const FirstRunVisualContext = createContext<FirstRunVisual>("app");

export function useFirstRunVisual(): FirstRunVisual {
  return useContext(FirstRunVisualContext);
}
