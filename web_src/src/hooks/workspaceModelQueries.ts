import type { QueryClient } from "@tanstack/react-query";
import type { CanvasesCanvas } from "@/api-client";

/** Refresh every model consumer after a workspace policy or source changes. */
export function invalidateWorkspaceModelQueries(queryClient: QueryClient, organizationId: string, factoryId: string) {
  return Promise.all([
    queryClient.invalidateQueries({ queryKey: ["factories", organizationId, factoryId, "llm-models"] }),
    queryClient.invalidateQueries({ queryKey: ["factories", organizationId, factoryId, "line-runner-models"] }),
    queryClient.invalidateQueries({ queryKey: ["factories", organizationId, factoryId, "owner-eligibility"] }),
    queryClient.invalidateQueries({ queryKey: ["organizations", organizationId, "selectable-llm-models", factoryId] }),
    queryClient.invalidateQueries({
      predicate: ({ queryKey }) =>
        queryKey[0] === "organizations" &&
        queryKey[1] === organizationId &&
        (queryKey[2] === "byok-models" || queryKey[2] === "hosted-llm-models") &&
        queryKey[4] === factoryId,
    }),
    queryClient.invalidateQueries({
      predicate: ({ queryKey, state }) =>
        queryKey[0] === "canvases" &&
        queryKey[1] === "detail" &&
        queryKey[2] === organizationId &&
        (state.data as CanvasesCanvas | undefined)?.metadata?.factoryId === factoryId,
    }),
  ]);
}
