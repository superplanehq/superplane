import { type ReactNode } from "react";

import { type BacklogIntakeItemCatalog } from "./backlogIntakeItems";
import { BacklogIntakeItemsContext, EMPTY_BACKLOG_INTAKE_CATALOG } from "./useBacklogIntakeItemCatalog";

export function BacklogIntakeItemsProvider({
  catalog,
  children,
}: {
  catalog?: BacklogIntakeItemCatalog;
  children: ReactNode;
}) {
  return (
    <BacklogIntakeItemsContext.Provider value={catalog ?? EMPTY_BACKLOG_INTAKE_CATALOG}>
      {children}
    </BacklogIntakeItemsContext.Provider>
  );
}
