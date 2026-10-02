/**
 * Design-time types for the verification, suggestions, quality pack, and
 * codebase health Storybook designs. These model the concepts specified in
 * docs/prd/line-verification.md, docs/prd/verification-suggestions.md,
 * docs/prd/code-quality-pack.md, and docs/prd/codebase-health.md.
 *
 * Presentational only: no API client types, no fetching. The eventual
 * implementation replaces these with generated API types.
 */

export type QualityDomain = "type-safety" | "tests" | "secrets" | "dead-code" | "file-size" | "dependencies";

export const QUALITY_DOMAIN_LABELS: Record<QualityDomain, string> = {
  "type-safety": "Type safety",
  tests: "Tests",
  secrets: "Secrets",
  "dead-code": "Dead code",
  "file-size": "File size",
  dependencies: "Dependencies",
};

export type FindingSeverity = "high" | "medium" | "low";

export type RuleEnforcement = "blocking" | "advisory";

export interface Rule {
  id: string;
  name: string;
  domain: QualityDomain;
  description: string;
  severity: FindingSeverity;
  enforcement: RuleEnforcement;
}

export interface RuleSet {
  id: string;
  name: string;
  description: string;
  rules: Rule[];
}

export type CheckKind = "agent" | "command";

export type CheckOutcome = "running" | "passed" | "failed" | "skipped";

export interface VerificationCheck {
  id: string;
  name: string;
  domain: QualityDomain;
  kind: CheckKind;
  blocking: boolean;
}

export interface CheckResult {
  check: VerificationCheck;
  outcome: CheckOutcome;
  findingCount: number;
  /** Command line for command checks; shown so results are reproducible. */
  command?: string;
  summary?: string;
}

export type FindingStatus = "open" | "fixed" | "dismissed" | "accepted";

export interface FindingLocation {
  path: string;
  startLine?: number;
  endLine?: number;
}

export interface Finding {
  id: string;
  ruleId: string;
  ruleName: string;
  domain: QualityDomain;
  severity: FindingSeverity;
  enforcement: RuleEnforcement;
  location?: FindingLocation;
  description: string;
  remediation: string;
  status: FindingStatus;
}

export type VerificationRunStatus = "running" | "passed" | "failed";

export interface VerificationRun {
  id: string;
  suiteName: string;
  ruleSetName: string;
  status: VerificationRunStatus;
  startedAt: Date;
  finishedAt?: Date;
  checks: CheckResult[];
  findings: Finding[];
}

export type LineStepType = "runApp" | "verify";

export interface VerifyStepDraft {
  name: string;
  type: LineStepType;
  suiteId: string;
  ruleSetName: string;
  checks: VerificationCheck[];
}

export interface VerificationSuiteOption {
  id: string;
  name: string;
  ruleSetName: string;
  checks: VerificationCheck[];
}

export type FixDispatchTarget = "work-order" | "agent-run";

export type SuggestionFixStatus = "none" | "in-progress" | "fixed";

export interface Suggestion {
  id: string;
  finding: Finding;
  fixPrompt: string;
  sourceWorkOrderTitle: string;
  occurrences: number;
  fixStatus: SuggestionFixStatus;
}

export type OccurrenceTrend = "up" | "down" | "flat";

export interface RecurringSuggestionRow {
  id: string;
  patternName: string;
  ruleName: string;
  domain: QualityDomain;
  count: number;
  trend: OccurrenceTrend;
  lastSeenAt: Date;
}

export interface QualityTemplate {
  id: string;
  name: string;
  domain: QualityDomain;
  description: string;
  checksPerRun: number;
  installed: boolean;
}

export interface PresetLineStep {
  name: string;
  type: LineStepType;
  summary: string;
  /** Check names shown when the verify step is expanded. */
  checkNames?: string[];
}

export interface HealthSnapshot {
  score: number;
  /** Score change versus the previous period; negative means decline. */
  change: number;
  target?: number;
  /** Oldest-first score series for the sparkline. */
  series: number[];
}

export interface Streak {
  label: string;
  current: number;
  best: number;
  unit: string;
}

export interface Achievement {
  id: string;
  name: string;
  description: string;
  earnedAt?: Date;
  /** What remains to earn it; set only when not yet earned. */
  progressNote?: string;
}

export interface RecurringPattern {
  id: string;
  name: string;
  description: string;
  openCount: number;
  /** Oldest-first occurrence counts for the trend sparkline. */
  occurrenceSeries: number[];
  topFiles: { path: string; count: number }[];
  remediation: string;
}

export interface SeverityCounts {
  high: number;
  medium: number;
  low: number;
}

export interface RunSummaryReport {
  runLabel: string;
  detected: SeverityCounts;
  fixed: SeverityCounts;
  remaining: SeverityCounts;
  gatePassed: boolean;
  blockingCount: number;
}
