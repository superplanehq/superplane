import { useEffect, useState } from "react";

/** Opens integration links from canvas and console markdown. */
export function useCanvasIntegrationRequest(onCreate: (integrationName: string) => void) {
  const [integrationId, setIntegrationId] = useState<string | null>(null);
  useEffect(() => {
    const handler = (event: Event) => {
      const { integrationName, instanceId } = (
        event as CustomEvent<{
          integrationName?: string;
          instanceId?: string;
        }>
      ).detail;
      if (instanceId) {
        setIntegrationId(instanceId);
        return;
      }
      if (integrationName) onCreate(integrationName);
    };
    window.addEventListener("canvas:open-integration", handler);
    return () => window.removeEventListener("canvas:open-integration", handler);
  }, [onCreate]);
  return { integrationId, closeIntegration: () => setIntegrationId(null) };
}
