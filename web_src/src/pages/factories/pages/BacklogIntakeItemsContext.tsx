import { createContext, useContext, type ReactNode } from "react";

import { type BacklogIntakeItemCatalog } from "./backlogIntakeItems";

const EMPTY_CATALOG: BacklogIntakeItemCatalog = { items: [] };

const BacklogIntakeItemsContext = createContext<BacklogIntakeItemCatalog>(EMPTY_CATALOG);

export function BacklogIntakeItemsProvider({
  catalog,
  children,
}: {
  catalog?: BacklogIntakeItemCatalog;
  children: ReactNode;
}) {
  return (
    <BacklogIntakeItemsContext.Provider value={catalog ?? EMPTY_CATALOG}>{children}</BacklogIntakeItemsContext.Provider>
  );
}

export function useBacklogIntakeItemCatalog(): BacklogIntakeItemCatalog {
  return useContext(BacklogIntakeItemsContext);
}
