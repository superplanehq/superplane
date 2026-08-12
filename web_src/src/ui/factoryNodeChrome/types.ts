/**
 * Factory card footer statuses.
 * Keeps classic ComponentBase event states (queued, cancelled, …) — not only
 * Storybook’s four tones — so colors/labels match production executions.
 */
export type FactoryNodeStatus =
  | "passed"
  | "failed"
  | "running"
  | "pending"
  | "queued"
  | "cancelled"
  | "cancelling"
  | "error"
  | "triggered";
