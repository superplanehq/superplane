import { createContext, useContext } from "react";

import { type BacklogIntakeItemCatalog } from "./backlogIntakeItems";

export const EMPTY_BACKLOG_INTAKE_CATALOG: BacklogIntakeItemCatalog = { items: [] };

export const BacklogIntakeItemsContext = createContext<BacklogIntakeItemCatalog>(EMPTY_BACKLOG_INTAKE_CATALOG);

export function useBacklogIntakeItemCatalog(): BacklogIntakeItemCatalog {
  return useContext(BacklogIntakeItemsContext);
}
