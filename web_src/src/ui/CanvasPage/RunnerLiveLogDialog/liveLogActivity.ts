import {
  emptyAgentActivityState,
  reduceAgentActivityRecord,
  type AgentActivity,
  type AgentActivityRecord,
} from "@/lib/agentActivity";

import { shouldSkipUnindexedLiveLogReplay } from "./liveLogSections";
import type { CommandSection, LogState } from "./types";

export function applyLiveLogActivityRecord(
  state: LogState,
  record: AgentActivityRecord,
  commandIndex: number | undefined,
  reconnecting: boolean,
): LogState {
  if (record.schema_version !== 2) {
    return state;
  }
  if (shouldSkipUnindexedLiveLogReplay(reconnecting, commandIndex, hasFinishedCommandSection(state))) {
    return state;
  }

  const previous = state.activityState ?? emptyAgentActivityState;
  const activityState = reduceAgentActivityRecord(previous, record);
  if (activityState === previous) {
    return state;
  }

  const activityId = record.activity_id;
  if (!activityId) {
    return { ...state, activityState };
  }
  const activity = activityState.activities.find((item) => item.id === activityId);
  if (!activity) {
    return { ...state, activityState };
  }

  return {
    ...state,
    activityState,
    sections: attachActivityToSections(state.sections, activity, commandIndex),
  };
}

function hasFinishedCommandSection(state: LogState): boolean {
  return state.sections.some((section) => section.status !== "running");
}

function attachActivityToSections(
  sections: CommandSection[],
  activity: AgentActivity,
  commandIndex: number | undefined,
): CommandSection[] {
  if (sections.length === 0) {
    return sections;
  }
  const pos =
    commandIndex === undefined
      ? latestRunningSectionPosition(sections)
      : sections.findIndex((section) => section.index === commandIndex);
  if (pos < 0) {
    return sections;
  }
  return sections.map((section, index) => {
    const activities = section.activities ?? [];
    if (index !== pos) {
      if (!activities.some((item) => item.id === activity.id)) {
        return section;
      }
      return { ...section, activities: activities.filter((item) => item.id !== activity.id) };
    }
    const existing = activities.findIndex((item) => item.id === activity.id);
    if (existing < 0) {
      return { ...section, activities: [...activities, activity] };
    }
    const next = [...activities];
    next[existing] = activity;
    return { ...section, activities: next };
  });
}

function latestRunningSectionPosition(sections: CommandSection[]): number {
  for (let index = sections.length - 1; index >= 0; index -= 1) {
    if (sections[index].status === "running") {
      return index;
    }
  }
  return sections.length - 1;
}
