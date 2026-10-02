import type { FactoriesFactoryAgentResource } from "@/api-client";
import { useEffect, useRef, useState } from "react";

import { nextDisabledTools, workspaceDisabledTools } from "./mcpTools";

export function useMCPDisabledTools(resource: FactoriesFactoryAgentResource) {
  const serverDisabled = workspaceDisabledTools(resource);
  const serverKey = serverDisabled.join("\0");
  const [pending, setPending] = useState<string[] | undefined>();
  const disabledRef = useRef(serverDisabled);

  useEffect(() => {
    setPending((current) => (current && current.join("\0") === serverKey ? undefined : current));
  }, [resource.id, serverKey]);

  const disabledTools = pending ?? serverDisabled;
  disabledRef.current = disabledTools;
  return {
    disabledTools,
    applyToolToggle: (toolName: string, enabled: boolean) => {
      const next = nextDisabledTools(disabledRef.current, toolName, enabled);
      disabledRef.current = next;
      setPending(next);
      return next;
    },
  };
}
