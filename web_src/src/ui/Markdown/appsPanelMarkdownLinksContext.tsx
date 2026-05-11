import React, { type ReactNode } from "react";

/**
 * When true (Apps / Launchpad markdown panels), prose links and blue-toned node
 * chips use the sky palette (`text-sky-600`, etc.) instead of blue.
 */
export const AppsPanelMarkdownLinksContext = React.createContext(false);

export function AppsPanelMarkdownLinksScope({ children }: { children: ReactNode }) {
  return <AppsPanelMarkdownLinksContext.Provider value={true}>{children}</AppsPanelMarkdownLinksContext.Provider>;
}
