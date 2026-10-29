import type { ReactNode } from "react";
import { FactoryTaskContext, type FactoryTaskIdentity } from "./factoryTaskContext";

export function FactoryTaskScope({ children, ...identity }: FactoryTaskIdentity & { children: ReactNode }) {
  return <FactoryTaskContext.Provider value={identity}>{children}</FactoryTaskContext.Provider>;
}
