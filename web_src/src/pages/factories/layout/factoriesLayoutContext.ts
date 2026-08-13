import type { FactoriesFactory } from "@/api-client";
import { createContext, useContext } from "react";

export interface FactoriesLayoutContextValue {
  organizationId: string;
  factoryId: string;
  factory: FactoriesFactory | null;
  factories: FactoriesFactory[];
  openCreateWorkOrder: () => void;
}

export const FactoriesLayoutContext = createContext<FactoriesLayoutContextValue | null>(null);

export function useFactoriesLayout(): FactoriesLayoutContextValue {
  const context = useContext(FactoriesLayoutContext);
  if (!context) {
    throw new Error("useFactoriesLayout must be used within FactoriesLayout");
  }
  return context;
}
