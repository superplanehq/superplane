import type { FactoriesFactory } from "@/api-client";
import { createContext, useContext } from "react";

import { factoryRouteSegment } from "../lib/factoryKeyResolution";

export interface FactoriesLayoutContextValue {
  organizationId: string;
  /** Real database id — use for API calls, mutations, and the websocket subscription. */
  factoryId: string;
  /** Canonical workspace key (e.g. `SP`) — use for task identifiers. */
  factoryKey: string;
  /** Canonical workspace URL segment (`sp-k7m2xqab`) — use for building links. */
  routeSegment?: string;
  factory: FactoriesFactory | null;
  factories: FactoriesFactory[];
  openCreateWorkOrder: () => void;
}

/** Layout after `useFactoriesLayout`, with a concrete URL segment for path builders. */
export type ResolvedFactoriesLayout = FactoriesLayoutContextValue & { routeSegment: string };

export const FactoriesLayoutContext = createContext<FactoriesLayoutContextValue | null>(null);

function withRouteSegment(context: FactoriesLayoutContextValue): ResolvedFactoriesLayout {
  return {
    ...context,
    routeSegment: context.routeSegment || factoryRouteSegment(context.factory) || context.factoryKey,
  };
}

export function useFactoriesLayout(): ResolvedFactoriesLayout {
  const context = useOptionalFactoriesLayout();
  if (!context) {
    throw new Error("useFactoriesLayout must be used within FactoriesLayout");
  }
  return withRouteSegment(context);
}

export function useOptionalFactoriesLayout(): ResolvedFactoriesLayout | null {
  const context = useContext(FactoriesLayoutContext);
  if (!context) {
    return null;
  }
  return withRouteSegment(context);
}
