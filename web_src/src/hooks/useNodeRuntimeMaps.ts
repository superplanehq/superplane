import { useMemo } from "react";
import { useNodeExecutionStore } from "@/stores/nodeExecutionStore";
import { buildNodeRuntimeMaps, EMPTY_NODE_RUNTIME_MAPS, type NodeRuntimeMaps } from "@/lib/nodeRuntimeMaps";

export function useNodeRuntimeMaps(enabled: boolean): NodeRuntimeMaps {
  const data = useNodeExecutionStore((state) => (enabled ? state.data : null));
  return useMemo(() => {
    if (data == null) {
      return EMPTY_NODE_RUNTIME_MAPS;
    }
    return buildNodeRuntimeMaps(data);
  }, [data]);
}
