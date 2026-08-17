import { useContext } from "react";

import { WorkOrderReactionsStoreContext } from "./reactionStoreContextValue";

export function useWorkOrderReactionsStore() {
  const value = useContext(WorkOrderReactionsStoreContext);
  if (!value) {
    throw new Error("useWorkOrderReactionsStore requires WorkOrderReactionsProvider");
  }
  return value;
}
