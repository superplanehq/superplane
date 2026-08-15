import { useContext } from "react";

import { MissionAssignmentContext } from "./missionAssignmentContextValue";

/** Returns null outside the Storybook missions harness. */
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
