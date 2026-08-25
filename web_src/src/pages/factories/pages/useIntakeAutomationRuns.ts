import { useFactoryIntakeRuns } from "@/hooks/useFactoryIntakeData";
import { useMemo } from "react";

import { intakeRunsFromApi, type IntakeAutomationRun } from "./intakeSourceSettingsModel";
import type { ConfiguredLineIntakeSource } from "./lineIntakeModel";

interface LiveIntakeRuns {
  runs: IntakeAutomationRun[];
  isLoading: boolean;
  isError: boolean;
  retry: () => void;
}

export function useIntakeAutomationRuns(
  organizationId: string | undefined,
  factoryId: string | undefined,
  configuredSource: ConfiguredLineIntakeSource | undefined,
): LiveIntakeRuns {
  const query = useFactoryIntakeRuns(organizationId ?? "", factoryId ?? "", configuredSource?.intakeId);
  const runs = useMemo(
    () => intakeRunsFromApi(query.data ?? [], configuredSource?.appId),
    [query.data, configuredSource?.appId],
  );

  return {
    runs,
    isLoading: query.isLoading,
    isError: query.isError,
    retry: () => void query.refetch(),
  };
}
