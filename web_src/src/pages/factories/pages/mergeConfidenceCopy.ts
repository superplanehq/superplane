import type { MergeConfidenceCheck } from "./mergeConfidenceChecks";

export const MERGE_CONFIDENCE_CHECK_COPY: ReadonlyArray<{
  id: MergeConfidenceCheck;
  label: string;
  helper: string;
}> = [
  {
    id: "risk",
    label: "Blast radius",
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
    label: "Drift from Specification",
    helper: "Check how far the change moved from the original task.",
  },
  {
    id: "reversibility",
    label: "Reversibility",
    helper: "Check how safely you can undo the change. A higher score means the change is easier to undo.",
  },
];

export const MERGE_CONFIDENCE_SETTINGS_COPY = {
  checksLabel: "Merge confidence checks",
  checksHelper: "Choose what the agent checks on each pull request.",
  save: "Save checks",
  saving: "Saving...",
} as const;
