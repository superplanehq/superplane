import type { FactoriesFactory } from "@/api-client";
import { createContext, useContext } from "react";

export interface FactorySettingsLayoutContextValue {
  organizationId: string;
  factoryId: string;
  factory: FactoriesFactory;
}

export const FactorySettingsLayoutContext = createContext<FactorySettingsLayoutContextValue | null>(null);

export function useFactorySettingsLayout(): FactorySettingsLayoutContextValue {
  const context = useContext(FactorySettingsLayoutContext);
  if (!context) {
    throw new Error("useFactorySettingsLayout must be used within FactorySettingsLayout");
  }
  return context;
}
