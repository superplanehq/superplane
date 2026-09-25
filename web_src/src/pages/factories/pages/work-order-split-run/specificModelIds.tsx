/* eslint-disable react-refresh/only-export-components -- provider and hooks share one context module */
import { createContext, useContext, useMemo, type ReactNode } from "react";

import { useSelectableLLMModels } from "@/hooks/useSelectableLLMModels";

const SpecificModelIdsContext = createContext<readonly string[]>([]);

export function useSpecificModelIds(): readonly string[] {
  return useContext(SpecificModelIdsContext);
}

/** Versioned model ids from the organization key, used to replace Claude aliases. */
export function SpecificModelIdsProvider({
  organizationId,
  children,
}: {
  organizationId?: string;
  children: ReactNode;
}) {
  const query = useSelectableLLMModels(organizationId, { enabled: Boolean(organizationId) });
  const ids = useMemo(() => (query.data ?? []).map((model) => model.model.id), [query.data]);
  return <SpecificModelIdsContext.Provider value={ids}>{children}</SpecificModelIdsContext.Provider>;
}
