import { createContext, useContext } from "react";

export type FactoryTaskIdentity = { factoryId?: string; orderId?: string };
export const FactoryTaskContext = createContext<FactoryTaskIdentity>({});

export function useFactoryTaskIdentity() {
  return useContext(FactoryTaskContext);
}
