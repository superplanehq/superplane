import { useFactoryIntakeRuns } from "@/hooks/useFactoryData";
import { useMemo } from "react";

import { analyzingTicketsFromApi } from "./intakeSourceSettingsModel";
import type { ConfiguredLineIntakeSource, LineIntakeAnalyzingTicket } from "./lineIntakeModel";

interface LiveIntakeTickets {
  tickets: LineIntakeAnalyzingTicket[];
  isLoading: boolean;
  isError: boolean;
  retry: () => void;
}

export function useLiveIntakeTickets(
  organizationId: string | undefined,
  factoryId: string | undefined,
  configuredSource: ConfiguredLineIntakeSource | undefined,
  enabled: boolean,
): LiveIntakeTickets {
  const query = useFactoryIntakeRuns(organizationId ?? "", factoryId ?? "", configuredSource?.intakeId, enabled);
  const tickets = useMemo(
    () => analyzingTicketsFromApi(query.data ?? [], configuredSource?.appId),
    [query.data, configuredSource?.appId],
  );

  return {
    tickets,
    isLoading: query.isLoading,
    isError: query.isError,
    retry: () => void query.refetch(),
  };
}
