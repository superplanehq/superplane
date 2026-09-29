import { createContext, useContext } from "react";

export type RunsSidebarHrefForRun = (runId: string) => string;

export const RunsSidebarHrefContext = createContext<RunsSidebarHrefForRun | undefined>(undefined);

export function useRunsSidebarRunHref(runId: string | undefined, fallbackHref: string): string {
  const hrefForRun = useContext(RunsSidebarHrefContext);
  if (!runId) {
    return fallbackHref;
  }
  return hrefForRun?.(runId) || fallbackHref;
}
