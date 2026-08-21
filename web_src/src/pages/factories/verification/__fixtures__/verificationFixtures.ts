/**
 * Shared mock data for the verification, suggestions, quality pack, and
 * codebase health Storybook designs. Story-side only; never import from
 * application components.
 */

import type {
  Achievement,
  CheckResult,
  Finding,
  HealthSnapshot,
  PresetLineStep,
  QualityTemplate,
  RecurringPattern,
  RecurringSuggestionRow,
  Rule,
  RuleSet,
  RunSummaryReport,
  Streak,
  Suggestion,
  VerificationCheck,
  VerificationRun,
  VerificationSuiteOption,
} from "../types";

const now = Date.now();
const minutesAgo = (minutes: number) => new Date(now - minutes * 60_000);
const daysAgo = (days: number) => new Date(now - days * 86_400_000);

export const PRODUCTION_RULES: Rule[] = [
  {
    id: "type-safety/no-untyped-values",
    name: "No untyped values",
    domain: "type-safety",
    description: "Every value has an explicit or inferred type. Escape hatches need a documented reason.",
    severity: "high",
    enforcement: "blocking",
  },
  {
    id: "type-safety/no-unsafe-casts",
    name: "No unsafe casts",
    domain: "type-safety",
    description: "Replace casts with runtime checks or parser functions at trust boundaries.",
    severity: "medium",
    enforcement: "blocking",
  },
  {
    id: "tests/changed-code-is-tested",
    name: "Changed code is tested",
    domain: "tests",
    description: "Changed functions have tests that assert behavior, not only execution.",
    severity: "high",
    enforcement: "blocking",
  },
  {
    id: "tests/no-empty-assertions",
    name: "No empty assertions",
    domain: "tests",
    description: "A test must fail when the behavior it covers breaks.",
    severity: "medium",
    enforcement: "advisory",
  },
  {
    id: "secrets/no-committed-secrets",
    name: "No committed secrets",
    domain: "secrets",
    description: "No credentials in code, configuration, or history. Rotate any value that was committed.",
    severity: "high",
    enforcement: "blocking",
  },
  {
    id: "dead-code/no-unused-exports",
    name: "No unused exports",
    domain: "dead-code",
    description: "Remove exports with no references. Version control preserves the history.",
    severity: "low",
    enforcement: "advisory",
  },
  {
    id: "file-size/max-file-length",
    name: "Files stay under 500 lines",
    domain: "file-size",
    description: "Split files that grow past 500 lines into focused modules.",
    severity: "low",
    enforcement: "advisory",
  },
  {
    id: "dependencies/every-dependency-is-used",
    name: "Every dependency is used",
    domain: "dependencies",
    description: "Remove packages with no imports. Replace single-use packages with local code when small.",
    severity: "medium",
    enforcement: "advisory",
  },
];

export const PRODUCTION_RULE_SET: RuleSet = {
  id: "rule-set-production",
  name: "Production",
  description: "Default rules for production repositories.",
  rules: PRODUCTION_RULES,
};

export const EMPTY_RULE_SET: RuleSet = {
  id: "rule-set-new",
  name: "New rule set",
  description: "",
  rules: [],
};

export const DEFAULT_SUITE_CHECKS: VerificationCheck[] = [
  { id: "check-type-safety", name: "Type safety review", domain: "type-safety", kind: "agent", blocking: true },
  { id: "check-type-command", name: "Type check", domain: "type-safety", kind: "command", blocking: true },
  { id: "check-tests", name: "Test coverage review", domain: "tests", kind: "agent", blocking: true },
  { id: "check-test-command", name: "Test run", domain: "tests", kind: "command", blocking: true },
  { id: "check-secrets", name: "Secret scan", domain: "secrets", kind: "command", blocking: true },
  { id: "check-dead-code", name: "Dead code review", domain: "dead-code", kind: "agent", blocking: false },
  { id: "check-file-size", name: "File size review", domain: "file-size", kind: "agent", blocking: false },
  { id: "check-dependencies", name: "Dependency audit", domain: "dependencies", kind: "command", blocking: false },
];

export const SUITE_OPTIONS: VerificationSuiteOption[] = [
  {
    id: "suite-default-quality",
    name: "Default quality",
    ruleSetName: "Production",
    checks: DEFAULT_SUITE_CHECKS,
  },
  {
    id: "suite-secrets-only",
    name: "Secrets only",
    ruleSetName: "Production",
    checks: DEFAULT_SUITE_CHECKS.filter((check) => check.domain === "secrets"),
  },
];

export const OPEN_FINDINGS: Finding[] = [
  {
    id: "finding-1",
    ruleId: "type-safety/no-untyped-values",
    ruleName: "No untyped values",
    domain: "type-safety",
    severity: "high",
    enforcement: "blocking",
    location: { path: "web_src/src/pages/orders/orderMapper.ts", startLine: 42, endLine: 58 },
    description: "The webhook payload is used without a type. Three fields are read from it unchecked.",
    remediation: "Define a payload interface and parse the payload with a runtime check before use.",
    status: "open",
  },
  {
    id: "finding-2",
    ruleId: "tests/changed-code-is-tested",
    ruleName: "Changed code is tested",
    domain: "tests",
    severity: "high",
    enforcement: "blocking",
    location: { path: "pkg/billing/invoice.go", startLine: 118, endLine: 164 },
    description: "The new proration branch has no test. A rounding regression would not be caught.",
    remediation: "Add a test for the proration branch with one whole-period case and one partial-period case.",
    status: "open",
  },
  {
    id: "finding-3",
    ruleId: "dead-code/no-unused-exports",
    ruleName: "No unused exports",
    domain: "dead-code",
    severity: "low",
    enforcement: "advisory",
    location: { path: "web_src/src/lib/formatDuration.ts", startLine: 27 },
    description: "formatDurationShort has no references after the timeline rewrite.",
    remediation: "Remove the export and its tests. Restore from history if it is needed again.",
    status: "open",
  },
  {
    id: "finding-4",
    ruleId: "file-size/max-file-length",
    ruleName: "Files stay under 500 lines",
    domain: "file-size",
    severity: "low",
    enforcement: "advisory",
    location: { path: "pkg/workers/dispatcher.go" },
    description: "The file is 742 lines and holds routing, retries, and metrics together.",
    remediation: "Extract the retry policy and the metrics recorder into their own files.",
    status: "open",
  },
];

export const RESOLVED_FINDINGS: Finding[] = [
  {
    id: "finding-5",
    ruleId: "secrets/no-committed-secrets",
    ruleName: "No committed secrets",
    domain: "secrets",
    severity: "high",
    enforcement: "blocking",
    location: { path: "scripts/deploy.sh", startLine: 12 },
    description: "A provider token was committed in a deploy script.",
    remediation: "Move the token to a secret store and rotate it.",
    status: "fixed",
  },
  {
    id: "finding-6",
    ruleId: "type-safety/no-unsafe-casts",
    ruleName: "No unsafe casts",
    domain: "type-safety",
    severity: "medium",
    enforcement: "blocking",
    location: { path: "web_src/src/pages/reports/exportCsv.ts", startLine: 88 },
    description: "A double cast bypasses the row type on export.",
    remediation: "Map each row field through a constructor instead of casting the array.",
    status: "accepted",
  },
];

export const FAILED_VERIFICATION_CHECKS: CheckResult[] = [
  {
    check: DEFAULT_SUITE_CHECKS[0],
    outcome: "failed",
    findingCount: 1,
    summary: "1 blocking finding in changed files.",
  },
  {
    check: DEFAULT_SUITE_CHECKS[1],
    outcome: "passed",
    findingCount: 0,
    command: "npx tsc --noEmit",
  },
  {
    check: DEFAULT_SUITE_CHECKS[2],
    outcome: "failed",
    findingCount: 1,
    summary: "1 changed function without tests.",
  },
  {
    check: DEFAULT_SUITE_CHECKS[3],
    outcome: "passed",
    findingCount: 0,
    command: "go test ./...",
  },
  {
    check: DEFAULT_SUITE_CHECKS[4],
    outcome: "passed",
    findingCount: 0,
    command: "gitleaks detect --source .",
  },
  {
    check: DEFAULT_SUITE_CHECKS[5],
    outcome: "passed",
    findingCount: 1,
    summary: "1 advisory finding.",
  },
  {
    check: DEFAULT_SUITE_CHECKS[6],
    outcome: "passed",
    findingCount: 1,
    summary: "1 advisory finding.",
  },
  {
    check: DEFAULT_SUITE_CHECKS[7],
    outcome: "skipped",
    findingCount: 0,
    command: "npm audit --omit dev",
    summary: "No dependency changes in this work order.",
  },
];

export const FAILED_VERIFICATION_RUN: VerificationRun = {
  id: "verification-run-failed",
  suiteName: "Default quality",
  ruleSetName: "Production",
  status: "failed",
  startedAt: minutesAgo(18),
  finishedAt: minutesAgo(11),
  checks: FAILED_VERIFICATION_CHECKS,
  findings: OPEN_FINDINGS,
};

export const PASSED_VERIFICATION_RUN: VerificationRun = {
  id: "verification-run-passed",
  suiteName: "Default quality",
  ruleSetName: "Production",
  status: "passed",
  startedAt: minutesAgo(45),
  finishedAt: minutesAgo(38),
  checks: FAILED_VERIFICATION_CHECKS.map((result) => ({
    ...result,
    outcome: result.outcome === "skipped" ? "skipped" : "passed",
    findingCount: result.check.blocking ? 0 : result.findingCount,
    summary: result.check.blocking ? undefined : result.summary,
  })),
  findings: OPEN_FINDINGS.filter((finding) => finding.enforcement === "advisory"),
};

export const RUNNING_VERIFICATION_RUN: VerificationRun = {
  id: "verification-run-running",
  suiteName: "Default quality",
  ruleSetName: "Production",
  status: "running",
  startedAt: minutesAgo(3),
  checks: FAILED_VERIFICATION_CHECKS.map((result, index) => ({
    ...result,
    outcome: index < 3 ? result.outcome : "running",
    findingCount: index < 3 ? result.findingCount : 0,
    summary: index < 3 ? result.summary : undefined,
  })),
  findings: [],
};

export const SUGGESTIONS: Suggestion[] = OPEN_FINDINGS.map((finding, index) => ({
  id: `suggestion-${finding.id}`,
  finding,
  fixPrompt:
    `Fix the finding "${finding.ruleName}" at ${finding.location?.path ?? "the repository"}. ` + finding.remediation,
  sourceWorkOrderTitle: "Add proration to invoice billing",
  occurrences: index === 0 ? 3 : 1,
  fixStatus: index === 1 ? "in-progress" : "none",
}));

export const RECURRING_SUGGESTION_ROWS: RecurringSuggestionRow[] = [
  {
    id: "recurring-1",
    patternName: "Untyped API response handling",
    ruleName: "No untyped values",
    domain: "type-safety",
    count: 14,
    trend: "up",
    lastSeenAt: minutesAgo(11),
  },
  {
    id: "recurring-2",
    patternName: "New branches without tests",
    ruleName: "Changed code is tested",
    domain: "tests",
    count: 9,
    trend: "flat",
    lastSeenAt: daysAgo(1),
  },
  {
    id: "recurring-3",
    patternName: "Stale exports after rewrites",
    ruleName: "No unused exports",
    domain: "dead-code",
    count: 6,
    trend: "down",
    lastSeenAt: daysAgo(3),
  },
  {
    id: "recurring-4",
    patternName: "Worker files that keep growing",
    ruleName: "Files stay under 500 lines",
    domain: "file-size",
    count: 4,
    trend: "down",
    lastSeenAt: daysAgo(6),
  },
];

export const QUALITY_TEMPLATES: QualityTemplate[] = [
  {
    id: "template-type-safety",
    name: "Type Safety Review",
    domain: "type-safety",
    description: "Finds untyped values and unsafe casts in changed files, then runs the type check command.",
    checksPerRun: 4,
    installed: true,
  },
  {
    id: "template-tests",
    name: "Test Coverage Review",
    domain: "tests",
    description: "Finds changed code without tests and weak assertions, then runs the test command.",
    checksPerRun: 4,
    installed: false,
  },
  {
    id: "template-secrets",
    name: "Secret Scan",
    domain: "secrets",
    description: "Scans source, configuration, and history for committed credentials.",
    checksPerRun: 4,
    installed: false,
  },
  {
    id: "template-dead-code",
    name: "Dead Code Review",
    domain: "dead-code",
    description: "Finds unused exports, orphaned files, and stale annotations, then runs the build command.",
    checksPerRun: 4,
    installed: false,
  },
  {
    id: "template-file-size",
    name: "File Size Review",
    domain: "file-size",
    description: "Finds oversized files and modules with several responsibilities.",
    checksPerRun: 3,
    installed: false,
  },
  {
    id: "template-dependencies",
    name: "Dependency Audit",
    domain: "dependencies",
    description: "Finds unused and single-use packages, then runs the vulnerability audit command.",
    checksPerRun: 3,
    installed: false,
  },
];

export const VERIFIED_DELIVERY_STEPS: PresetLineStep[] = [
  { name: "build", type: "runApp", summary: "Run the build app for the work order branch." },
  {
    name: "verify",
    type: "verify",
    summary: "Run the quality suite. Blocking findings stop the line.",
    checkNames: QUALITY_TEMPLATES.map((template) => template.name),
  },
  { name: "approval", type: "runApp", summary: "Request approval from the reviewers." },
  { name: "close", type: "runApp", summary: "Close the work order as completed." },
];

export const HEALTH_SNAPSHOT: HealthSnapshot = {
  score: 82,
  change: 6,
  target: 90,
  series: [61, 64, 63, 68, 71, 74, 73, 78, 80, 82],
};

export const STREAKS: Streak[] = [
  { label: "Work orders without blocking findings", current: 12, best: 21, unit: "work orders" },
  { label: "Days without blocking findings", current: 8, best: 15, unit: "days" },
];

export const ACHIEVEMENTS: Achievement[] = [
  {
    id: "achievement-first-pass",
    name: "First verification passed",
    description: "A work order passed the full verification suite.",
    earnedAt: daysAgo(41),
  },
  {
    id: "achievement-type-safety-30",
    name: "30 days without a type-safety finding",
    description: "No new type-safety findings for 30 days in a row.",
    earnedAt: daysAgo(4),
  },
  {
    id: "achievement-pattern-resolved",
    name: "Recurring pattern resolved: Stale exports after rewrites",
    description: "The open count for this pattern reached zero.",
    earnedAt: daysAgo(12),
  },
  {
    id: "achievement-secrets-90",
    name: "Zero secrets findings for 90 days",
    description: "No secrets findings for 90 days in a row.",
    progressNote: "34 days remain.",
  },
  {
    id: "achievement-streak-25",
    name: "25 work orders without blocking findings",
    description: "25 work orders in a row passed verification.",
    progressNote: "13 work orders remain.",
  },
];

export const RECURRING_PATTERNS: RecurringPattern[] = [
  {
    id: "pattern-untyped-responses",
    name: "Untyped API response handling",
    description: "External payloads are read without a parse step, so type errors surface at run time.",
    openCount: 14,
    occurrenceSeries: [4, 6, 7, 9, 11, 12, 14],
    topFiles: [
      { path: "web_src/src/pages/orders/orderMapper.ts", count: 5 },
      { path: "web_src/src/pages/reports/exportCsv.ts", count: 4 },
      { path: "pkg/integrations/payments/webhook.go", count: 3 },
    ],
    remediation: "Parse every external payload with a typed parser function at the boundary.",
  },
  {
    id: "pattern-untested-branches",
    name: "New branches without tests",
    description: "New conditional branches ship without a test that exercises them.",
    openCount: 9,
    occurrenceSeries: [11, 10, 10, 9, 9, 9, 9],
    topFiles: [
      { path: "pkg/billing/invoice.go", count: 4 },
      { path: "pkg/workers/dispatcher.go", count: 3 },
    ],
    remediation: "Add one test per new branch before review. Cover the boundary values.",
  },
];

export const RUN_SUMMARY_FAILED: RunSummaryReport = {
  runLabel: "Verification for work order: Add proration to invoice billing",
  detected: { high: 2, medium: 0, low: 2 },
  fixed: { high: 1, medium: 1, low: 0 },
  remaining: { high: 2, medium: 0, low: 2 },
  gatePassed: false,
  blockingCount: 2,
};

export const RUN_SUMMARY_PASSED: RunSummaryReport = {
  runLabel: "Verification for work order: Update the export columns",
  detected: { high: 0, medium: 0, low: 1 },
  fixed: { high: 2, medium: 0, low: 1 },
  remaining: { high: 0, medium: 0, low: 3 },
  gatePassed: true,
  blockingCount: 0,
};
