import type { ReactNode } from "react";

import { RunsSidebarHrefContext, type RunsSidebarHrefForRun } from "./useRunsSidebarRunHref";

export type { RunsSidebarHrefForRun };

export function RunsSidebarHrefProvider({
  hrefForRun,
  children,
}: {
  hrefForRun?: RunsSidebarHrefForRun;
  children: ReactNode;
}) {
  return <RunsSidebarHrefContext.Provider value={hrefForRun}>{children}</RunsSidebarHrefContext.Provider>;
}
