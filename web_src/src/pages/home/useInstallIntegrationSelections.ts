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
  manualSelectionNames,
  loading,
  initialPreferredIntegrationId,
}: {
  integrationData: IntegrationInstanceSummary[];
  selections: IntegrationSelections;
  onSelectionsChange: (selections: IntegrationSelections) => void;
  /** Integration names that are never auto-selected; the user must pick an instance. */
  manualSelectionNames?: readonly string[];
  /** True while the connection list still loads; sync waits so it does not drop a saved selection. */
  loading?: boolean;
  /** Integration that returned from an external setup flow before this hook mounted. */
  initialPreferredIntegrationId?: string | null;
}) {
  const [preferredInstanceIds, setPreferredInstanceIds] = useState<Record<string, string>>({});
  const [returningIntegrationId] = useState(initialPreferredIntegrationId);

  const rememberPreferredInstance = (integrationName: string, integrationId: string) => {
    setPreferredInstanceIds((prev) => ({ ...prev, [integrationName]: integrationId }));
  };

  useEffect(() => {
    if (loading) return;
    const returningIntegration = integrationData.find((data) =>
      data.allInstances.some((instance) => instance.metadata?.id === returningIntegrationId),
    );
    const effectivePreferredIds =
      returningIntegration && returningIntegrationId && !preferredInstanceIds[returningIntegration.name]
        ? { ...preferredInstanceIds, [returningIntegration.name]: returningIntegrationId }
        : preferredInstanceIds;
    const synced = syncSelectionsWithInstances(
      integrationData,
      selections,
      effectivePreferredIds,
      manualSelectionNames,
    );
    if (synced) onSelectionsChange(synced);
  }, [
    integrationData,
    selections,
    preferredInstanceIds,
    onSelectionsChange,
    manualSelectionNames,
    loading,
    returningIntegrationId,
  ]);

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
