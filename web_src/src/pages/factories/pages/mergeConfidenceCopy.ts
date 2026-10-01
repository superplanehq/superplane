import type { MergeConfidenceCheck } from "./mergeConfidenceChecks";

export const MERGE_CONFIDENCE_CHECK_COPY: ReadonlyArray<{
  id: MergeConfidenceCheck;
  label: string;
  helper: string;
}> = [
  {
    id: "risk",
    label: "Risk",
    helper: "Score how risky the change is. A higher score means more risk.",
  },
  {
    id: "performance",
    label: "Performance",
    helper: "Check the change against the performance practices in this application.",
  },
  {
    id: "security",
    label: "Security",
    helper: "Check the change against the security practices in this application.",
  },
  {
    id: "drift",
    label: "Drift",
    helper: "Check how far the change moved from the original task.",
  },
];

export const MERGE_CONFIDENCE_SETTINGS_COPY = {
  checksLabel: "Merge confidence checks",
  checksHelper: "Choose what the agent checks on each pull request.",
  save: "Save checks",
  saving: "Saving...",
} as const;
