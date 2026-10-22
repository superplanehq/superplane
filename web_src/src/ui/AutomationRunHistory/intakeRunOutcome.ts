export type IntakeRunOutcome =
  | "created"
  | "filtered"
  | "processing"
  | "failed"
  | "unknown"
  | "cancelled"
  | "no-proposal";

export const INTAKE_OUTCOMES: Record<IntakeRunOutcome, { label: string; dot: string }> = {
  "no-proposal": { label: "No task proposed", dot: "bg-muted-foreground/60" },
  created: { label: "Task created", dot: "bg-emerald-500" },
  filtered: { label: "Did not match filters", dot: "bg-muted-foreground/60" },
  processing: { label: "Processing", dot: "bg-blue-500" },
  failed: { label: "Failed", dot: "bg-red-500" },
  unknown: { label: "Outcome unavailable", dot: "bg-muted-foreground/60" },
  cancelled: { label: "Cancelled", dot: "bg-muted-foreground/60" },
};
