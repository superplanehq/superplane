import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from "react";

import { assignWorkOrderToMission, type MissionCloseReason } from "./missionListModel";
import { SEEDED_MISSIONS, missionByWorkOrderId as seededAssignments, type FactoryMission } from "./missionMocks";

export interface MissionAssignmentContextValue {
  missions: FactoryMission[];
  missionByWorkOrderId: Record<string, string>;
  assignMission: (workOrderId: string, missionId: string | null) => void;
  closedByMissionId: Record<string, MissionCloseReason>;
  closeMission: (missionId: string, reason: MissionCloseReason) => void;
  reopenMission: (missionId: string) => void;
}

const MissionAssignmentContext = createContext<MissionAssignmentContextValue | null>(null);

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

export function useOptionalMissionAssignment() {
  return useContext(MissionAssignmentContext);
}

export function useMissionAssignment() {
  const value = useOptionalMissionAssignment();
  if (!value) {
    throw new Error("useMissionAssignment requires MissionAssignmentProvider");
  }
  return value;
}
