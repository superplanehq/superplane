import { createContext } from "react";

import type { MissionCloseReason } from "./missionListModel";
import type { FactoryMission } from "./missionMocks";

export interface MissionAssignmentContextValue {
  missions: FactoryMission[];
  missionByWorkOrderId: Record<string, string>;
  assignMission: (workOrderId: string, missionId: string | null) => void;
  closedByMissionId: Record<string, MissionCloseReason>;
  closeMission: (missionId: string, reason: MissionCloseReason) => void;
  reopenMission: (missionId: string) => void;
}

export const MissionAssignmentContext = createContext<MissionAssignmentContextValue | null>(null);
