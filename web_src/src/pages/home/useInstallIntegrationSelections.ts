import { useEffect, useState } from "react";

import {
  syncSelectionsWithInstances,
  type IntegrationInstanceSummary,
  type IntegrationSelections,
} from "./homeIntegrationStatus";

/** Keep selections in sync and remember the instance created via Connect new / setup tab. */
export function useInstallIntegrationSelections({
  integrationData,
  selections,
  onSelectionsChange,
}: {
  integrationData: IntegrationInstanceSummary[];
  selections: IntegrationSelections;
  onSelectionsChange: (selections: IntegrationSelections) => void;
}) {
  const [preferredInstanceIds, setPreferredInstanceIds] = useState<Record<string, string>>({});

  const rememberPreferredInstance = (integrationName: string, integrationId: string) => {
    setPreferredInstanceIds((prev) => ({ ...prev, [integrationName]: integrationId }));
  };

  useEffect(() => {
    const synced = syncSelectionsWithInstances(integrationData, selections, preferredInstanceIds);
    if (synced) onSelectionsChange(synced);
  }, [integrationData, selections, preferredInstanceIds, onSelectionsChange]);

  return { rememberPreferredInstance };
}

/** GitHub setup opens in another tab; refresh when the user returns. */
export function useRefetchOnWindowFocus(refetch: () => unknown) {
  useEffect(() => {
    const refresh = () => {
      void refetch();
    };
    const onVisibility = () => {
      if (document.visibilityState === "visible") refresh();
    };
    window.addEventListener("focus", refresh);
    document.addEventListener("visibilitychange", onVisibility);
    return () => {
      window.removeEventListener("focus", refresh);
      document.removeEventListener("visibilitychange", onVisibility);
    };
  }, [refetch]);
}
