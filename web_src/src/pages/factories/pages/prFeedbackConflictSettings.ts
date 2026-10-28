export const PR_FEEDBACK_MERGE_CONFLICT_HOURS_MIN = 1;
export const PR_FEEDBACK_MERGE_CONFLICT_HOURS_MAX = 168;
export const PR_FEEDBACK_DEFAULT_MERGE_CONFLICT_HOURS = 24;

export function isPRFeedbackMergeConflictHoursValid(value: number): boolean {
  return (
    Number.isInteger(value) &&
    value >= PR_FEEDBACK_MERGE_CONFLICT_HOURS_MIN &&
    value <= PR_FEEDBACK_MERGE_CONFLICT_HOURS_MAX
  );
}

export function mergeConflictDraft(checks?: { fixMergeConflicts?: boolean; mergeConflictHours?: number }) {
  return {
    fixMergeConflicts: checks?.fixMergeConflicts !== false,
    mergeConflictHours: checks?.mergeConflictHours ?? PR_FEEDBACK_DEFAULT_MERGE_CONFLICT_HOURS,
  };
}
