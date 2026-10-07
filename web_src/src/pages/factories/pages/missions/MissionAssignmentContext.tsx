import { useCallback, useMemo, useState, type ReactNode } from "react";

import { assignWorkOrderToMission, type MissionCloseReason } from "./missionListModel";
import { MissionAssignmentContext } from "./missionAssignmentContextValue";
import { SEEDED_MISSIONS, missionByWorkOrderId as seededAssignments } from "./missionMocks";

/** Storybook-only assignment store. The live work-order API does not receive this map. */
export function MissionAssignmentProvider({ children }: { children: ReactNode }) {
  const [missionByWorkOrderId, setMissionByWorkOrderId] = useState<Record<string, string>>(() => ({
    ...seededAssignments,
  }));
  const [closedByMissionId, setClosedByMissionId] = useState<Record<string, MissionCloseReason>>({});

  const assignMission = useCallback((workOrderId: string, missionId: string | null) => {
    setMissionByWorkOrderId((current) => assignWorkOrderToMission(current, workOrderId, missionId));
  }, []);

  const closeMission = useCallback((missionId: string, reason: MissionCloseReason) => {
    setClosedByMissionId((current) => ({ ...current, [missionId]: reason }));
  }, []);

  const reopenMission = useCallback((missionId: string) => {
    setClosedByMissionId((current) => {
      const next = { ...current };
      delete next[missionId];
      return next;
    });
  }, []);

  const value = useMemo(
    () => ({
      missions: SEEDED_MISSIONS,
      missionByWorkOrderId,
      assignMission,
      closedByMissionId,
      closeMission,
      reopenMission,
    }),
    [assignMission, closeMission, closedByMissionId, missionByWorkOrderId, reopenMission],
  );

  return <MissionAssignmentContext.Provider value={value}>{children}</MissionAssignmentContext.Provider>;
}
