import type { SuperplaneComponentsNode } from "@/api-client";
import type { CanvasMemoryEntry } from "@/hooks/useCanvasData";

import type { ConsoleContextValue, ConsoleNodeStatus } from "../ConsoleContext";
import type { WidgetChartRender, WidgetNumberRender, WidgetTableRender } from "../widget/types";
import type { ScorecardThreshold } from "../widget/WidgetScorecard";

/**
 * Shared fixtures for console panel Storybook stories. Panel renderers take
 * plain props, so stories feed static rows directly and never touch
 * `useWidgetData`.
 */

/** No-op callbacks for Storybook mocks. */
export function storyNoop(..._args: unknown[]): void {}

/** Sample canvas nodes used for node-reference resolution in the node panels. */
export const sampleNodes: SuperplaneComponentsNode[] = [
  { id: "node-deploy", name: "deploy-prod", type: "TYPE_TRIGGER", component: "deploy" },
  { id: "node-build", name: "build-image", type: "TYPE_ACTION", component: "build" },
  { id: "node-tests", name: "run-tests", type: "TYPE_TRIGGER", component: "tests" },
  { id: "node-notify", name: "notify-slack", type: "TYPE_ACTION", component: "slack" },
  { id: "trigger-check-pr", name: "trigger-check-pr", type: "TYPE_TRIGGER", component: "trigger" },
];

const sampleNodeStatuses: Record<string, ConsoleNodeStatus> = {
  "node-deploy": "running",
  "node-build": "passed",
  "node-tests": "failed",
  "node-notify": "pending",
};

/** Default mock console context for node panel stories. */
export const mockConsoleContextValue: ConsoleContextValue = {
  canvasId: "canvas-story",
  organizationId: "org-story",
  nodes: sampleNodes,
  nodeStatuses: sampleNodeStatuses,
  canRunNodes: true,
  onTriggerNode: storyNoop,
  onOpenNode: storyNoop,
};

/** Execution-like rows for table widgets (status, duration, owner, links). */
export const executionRows: Record<string, unknown>[] = [
  {
    id: "exec-1",
    name: "deploy-prod",
    status: "passed",
    service: "api",
    durationMs: 42_000,
    cost: 12.5,
    createdAt: "2026-06-26T09:12:00Z",
    url: "https://example.com/runs/exec-1",
  },
  {
    id: "exec-2",
    name: "build-image",
    status: "running",
    service: "api",
    durationMs: 8_000,
    cost: 3.2,
    createdAt: "2026-06-26T09:30:00Z",
    url: "https://example.com/runs/exec-2",
  },
  {
    id: "exec-3",
    name: "run-tests",
    status: "failed",
    service: "web",
    durationMs: 65_000,
    cost: 9.9,
    createdAt: "2026-06-26T08:55:00Z",
    url: "https://example.com/runs/exec-3",
  },
  {
    id: "exec-4",
    name: "notify-slack",
    status: "passed",
    service: "web",
    durationMs: 1_200,
    cost: 0.4,
    createdAt: "2026-06-26T09:40:00Z",
    url: "https://example.com/runs/exec-4",
  },
  {
    id: "exec-5",
    name: "lint",
    status: "cancelled",
    service: "infra",
    durationMs: 3_500,
    cost: 1.1,
    createdAt: "2026-06-26T07:20:00Z",
    url: "https://example.com/runs/exec-5",
  },
];

/** Aggregated per-service rows for bar / donut charts. */
export const serviceRows: Record<string, unknown>[] = [
  { service: "api", errors: 12, cost: 320, requests: 9800 },
  { service: "web", errors: 5, cost: 210, requests: 15400 },
  { service: "infra", errors: 21, cost: 540, requests: 4200 },
  { service: "data", errors: 3, cost: 130, requests: 2600 },
];

/** Time-series rows for line / area charts. */
export const timeseriesRows: Record<string, unknown>[] = [
  { day: "Mon", passed: 18, failed: 2 },
  { day: "Tue", passed: 22, failed: 4 },
  { day: "Wed", passed: 17, failed: 1 },
  { day: "Thu", passed: 25, failed: 6 },
  { day: "Fri", passed: 30, failed: 3 },
  { day: "Sat", passed: 12, failed: 0 },
  { day: "Sun", passed: 9, failed: 1 },
];

/** Single-metric rows for number widgets (sparkline-friendly). */
export const metricRows: Record<string, unknown>[] = timeseriesRows.map((row, idx) => ({
  index: idx,
  total: (row.passed as number) + (row.failed as number),
  passed: row.passed,
}));

/**
 * Fixtures for the prototype `WidgetScorecard` panel. Sparkline series trend
 * up / down / flat, and the band sets cover both goal directions.
 */
export const scorecardSparklineUp = [62, 65, 61, 70, 74, 78, 84, 91];
export const scorecardSparklineDown = [140, 132, 128, 121, 118, 96, 88, 74];
export const scorecardSparklineFlat = [50, 51, 49, 50, 52, 49, 50, 51];

/**
 * Real data behind the scorecard editor examples. Each series stands in for
 * `rows.map(seriesField)` in the preview (chronological oldest -> newest where
 * the metric is time-based), sampled from live canvases in the org so the
 * fields, sources, and values all make sense together.
 */

/** "Papercut Analysis" canvas — `Format Message` node output `total_open`, last week of daily runs (Runs source). */
export const papercutTotalOpen = [127, 127, 118, 125, 111, 108, 92, 98];

/** Lower-is-better bands for the open-papercut count: good <= 80, warn <= 120, else bad. */
export const papercutOpenBands: ScorecardThreshold[] = [
  { at: 80, status: "good" },
  { at: 120, status: "warn" },
  { at: 100000, status: "bad" },
];

/** "LLM Cost Tracker 2" — per-account daily `cost_usd` from the `aws_costs_by_account` memory namespace (Memory source). */
export const cloudSpendByAccount = [29.6283, 0.0693, 9.8003, 3.8124, 2.3818, 2.0191, 0.6986, 23.5465, 2.7158, 2.5305];

/** "Papercut Analysis" — real `Fetch GitHub Stats` execution durations in ms, oldest -> newest (Executions source). */
export const fetchDurationsMs = [120463, 120112, 120100, 121065, 120632, 120692];

/** Lower-is-better duration bands (ms): good <= 90s, warn <= 150s, else bad. */
export const fetchTimeBands: ScorecardThreshold[] = [
  { at: 90000, status: "good" },
  { at: 150000, status: "warn" },
  { at: 100000000, status: "bad" },
];

/** Higher-is-better bands (e.g. success rate %): good >= 95, warn >= 85, else bad. */
export const scorecardHigherBands: ScorecardThreshold[] = [
  { at: 0, status: "bad" },
  { at: 85, status: "warn" },
  { at: 95, status: "good" },
];

/** Lower-is-better bands (e.g. error rate %): good <= 1, warn <= 5, else bad. */
export const scorecardLowerBands: ScorecardThreshold[] = [
  { at: 1, status: "good" },
  { at: 5, status: "warn" },
  { at: 100, status: "bad" },
];

/** Memory entries for the composite (multi-source) number widget. */
export const memoryEntries: CanvasMemoryEntry[] = [
  { id: "m1", namespace: "deploys", values: { count: 14 }, source: "node" },
  { id: "m2", namespace: "rollbacks", values: { count: 2 }, source: "node" },
];

export const baseTableRender: WidgetTableRender = {
  kind: "table",
  columns: [
    { field: "name", label: "Node" },
    { field: "status", label: "Status", format: "status" },
    { field: "service", label: "Service", format: "badge" },
    { field: "durationMs", label: "Duration", format: "duration" },
    { field: "createdAt", label: "Started", format: "relative" },
  ],
  emptyMessage: "No executions yet.",
};

export const baseBarChartRender: WidgetChartRender = {
  kind: "chart",
  type: "bar",
  xField: "service",
  series: [{ field: "errors", label: "Errors" }],
  yLabel: "Errors",
};

export const baseNumberRender: WidgetNumberRender = {
  kind: "number",
  aggregation: "sum",
  field: "total",
  label: "Total runs",
};

/** Memory rows for the `prRiskChecks` namespace (`checks-table` panel). */
export const prRiskCheckRows: Record<string, unknown>[] = [
  {
    id: "check-1",
    pr_number: 42,
    title: "Fix auth middleware",
    author: "alice",
    risk_score: 15,
    risk_level: "low",
    repository: "acme/api",
    last_checked_at: "2026-06-26T09:12:00Z",
  },
  {
    id: "check-2",
    pr_number: 87,
    title: "Refactor billing webhooks",
    author: "bob",
    risk_score: 52,
    risk_level: "medium",
    repository: "acme/billing",
    last_checked_at: "2026-06-26T08:05:00Z",
  },
  {
    id: "check-3",
    pr_number: 103,
    title: "Remove legacy session store",
    author: "carol",
    risk_score: 88,
    risk_level: "critical",
    repository: "acme/api",
    last_checked_at: "2026-06-25T22:40:00Z",
  },
  {
    id: "check-4",
    pr_number: 19,
    title: "Docs: update onboarding guide",
    author: "dave",
    risk_score: 4,
    risk_level: "very low",
    repository: "acme/docs",
    last_checked_at: "2026-06-25T18:20:00Z",
  },
  {
    id: "check-5",
    pr_number: 64,
    title: "Add rate limiting to public API",
    author: "erin",
    risk_score: 71,
    risk_level: "high",
    repository: "acme/api",
    last_checked_at: "2026-06-24T14:10:00Z",
  },
];

/** Render config from the org `checks-table` panel (`pr-risk-review` console). */
export const prRiskChecksTableRender: WidgetTableRender = {
  kind: "table",
  columns: [
    { field: "pr_number", label: "PR" },
    {
      field: "title",
      label: "Title",
      format: "link",
      href: "https://github.com/{repository}/pull/{pr_number}",
    },
    { field: "author", label: "Author" },
    { field: '{{ string(int(risk_score)) + "/100" }}', label: "Risk" },
    { field: "risk_level", label: "Level", format: "status" },
    { field: "last_checked_at", label: "Last check", format: "relative" },
  ],
  rowActions: [
    {
      kind: "trigger",
      label: "Re-check",
      node: "trigger-check-pr",
      icon: "refresh",
      variant: "default",
      payload: {
        action: "manual",
        pull_number: "{{ pr_number }}",
        repository: "{{ repository }}",
      },
    },
  ],
  rowStyles: [
    { field: "risk_level", op: "eq", value: "very low", tone: "green" },
    { field: "risk_level", op: "eq", value: "low", tone: "green" },
    { field: "risk_level", op: "eq", value: "medium", tone: "yellow" },
    { field: "risk_level", op: "eq", value: "high", tone: "orange" },
    { field: "risk_level", op: "eq", value: "critical", tone: "red" },
  ],
  sort: { field: "last_checked_at", order: "desc" },
};

/** Markdown body from the org `readme` panel (`pr-risk-review` console). */
export const prRiskReviewMarkdownBody = `**PR Risk Review** uses a **Claude Managed Agent** to assess pull request risk, upsert a GitHub PR comment, optionally request reviewers, and notify Discord.

<details>
<summary>Quick start</summary>

1. Create a Claude Managed Agent + environment in Anthropic (must clone repos and review diffs).
2. Connect **Claude** and bind it on **Assess PR Risk**.
3. Connect **GitHub** and bind the integration on triggers, reviewers, and comment nodes.
4. Add **\`GITHUB_TOKEN\`** to the \`app-codeowners\` secret (injected into the agent session for private repos).
5. Optional: connect **Discord** on **Discord review posted**.
6. Use **Check pull request** below to run a check manually.

</details>

<details>
<summary>Manual check</summary>

Enter \`owner/repo\` and the pull request number, then click **Run**.

You can also re-check from the **Recent checks** table using the row action.

</details>

<details>
<summary>How review works</summary>

1. **Assess PR Risk** (\`claude.runAgent\`) runs your Managed Agent with PR context
2. **Format PR review** parses the JSON response
3. **Create PR comment** or **Update PR comment** posts via the GitHub integration
4. **Record check** saves results to the table below
5. **Publish PR Risk Review status** reports risk on the PR head commit
6. **Discord review posted** sends a notification when a review completes

</details>

<details>
<summary>Risk score</summary>

Risk score (\`0–100\`) and level appear in the PR comment, GitHub commit status, and console.

</details>

<details>
<summary>Discord notification</summary>

Posted when a review completes. Format:

\`[Fix auth middleware](<https://github.com/acme/api/pull/42>) - alice - Risk 15/100 (low)\`

</details>

<details>
<summary>Which branches?</summary>

Automatic checks run only on pull requests into **\`main\`** or **\`master\`**.

Draft PRs are skipped until marked **ready for review** (or updated with a new push).

Manual checks run for any PR.

</details>

<details>
<summary>What triggers a check?</summary>

A check runs when a pull request is **opened**, **updated**, **reopened**, or marked **ready for review**.

</details>`;

/** Layout size for the wide `checks-table` panel (grid w:8, h:15). */
export const prRiskChecksPanelSize = { width: 720, height: 420 } as const;

/** Layout size for the tall `readme` markdown panel (grid w:4, h:11). */
export const prRiskReviewMarkdownPanelSize = { width: 380, height: 360 } as const;

/** Rows exercising every `WidgetColumnFormat` for the table format showcase story. */
export const columnFormatShowcaseRows: Record<string, unknown>[] = [
  {
    id: "run-1",
    avatarUrl: "https://github.com/torvalds.png",
    summary: "Deploy api v2.14.0 to production",
    status: "passed",
    environment: "production",
    requests: 18_420,
    successRate: 0.998,
    rollout: 1,
    weekOverWeek: 12,
    durationMs: 124_500,
    scheduledOn: "2026-07-01",
    startedAt: "2026-07-08T06:12:00Z",
    updatedAt: "2026-07-08T06:14:05Z",
    runId: "exec-a1b2c3d4",
    linkLabel: "View logs",
    logsUrl: "https://example.com/runs/exec-a1b2c3d4",
  },
  {
    id: "run-2",
    avatarUrl: "https://github.com/gaearon.png",
    summary: "Roll out feature flags for checkout experiment",
    status: "running",
    environment: "staging",
    requests: 4_210,
    successRate: 0.942,
    rollout: 0.6,
    weekOverWeek: -0.084,
    durationMs: 8_200,
    scheduledOn: "2026-07-05",
    startedAt: "2026-07-08T08:30:00Z",
    updatedAt: "2026-07-08T08:31:12Z",
    runId: "exec-b7e8f901",
    linkLabel: "Open run",
    logsUrl: "https://example.com/runs/exec-b7e8f901",
  },
  {
    id: "run-3",
    avatarUrl: "https://github.com/sindresorhus.png",
    summary: "Run integration tests after billing webhook change",
    status: "failed",
    environment: "production",
    requests: 9_870,
    successRate: 0.871,
    rollout: 0.35,
    weekOverWeek: -5,
    durationMs: 287_000,
    scheduledOn: "2026-06-28",
    startedAt: "2026-07-07T14:05:00Z",
    updatedAt: "2026-07-07T14:09:47Z",
    runId: "exec-c3d4e5f6",
    linkLabel: "Debug failure",
    logsUrl: "https://example.com/runs/exec-c3d4e5f6",
  },
  {
    id: "run-4",
    avatarUrl: "https://github.com/yyx990803.png",
    summary: "Rebuild web assets and purge CDN cache",
    status: "passed",
    environment: "production",
    requests: 31_200,
    successRate: 99.6,
    rollout: 0.85,
    weekOverWeek: 0,
    durationMs: 45_800,
    scheduledOn: "2026-07-02",
    startedAt: "2026-07-08T04:00:00Z",
    updatedAt: "2026-07-08T04:00:46Z",
    runId: "exec-d4e5f6a7",
    linkLabel: "CDN report",
    logsUrl: "https://example.com/runs/exec-d4e5f6a7",
  },
  {
    id: "run-5",
    avatarUrl: "https://github.com/kentcdodds.png",
    summary: "Rotate database credentials for analytics cluster",
    status: "cancelled",
    environment: "infra",
    requests: 640,
    successRate: 1,
    rollout: 0.1,
    weekOverWeek: 0.21,
    durationMs: 3_500,
    scheduledOn: "2026-06-20",
    startedAt: "2026-07-06T11:20:00Z",
    updatedAt: "2026-07-06T11:20:18Z",
    runId: "exec-e5f6a7b8",
    linkLabel: "Audit trail",
    logsUrl: "https://example.com/runs/exec-e5f6a7b8",
  },
  {
    id: "run-6",
    avatarUrl: "",
    summary: "Backfill customer usage metrics for June billing close",
    status: "pending",
    environment: "data",
    requests: 156_000,
    successRate: 0.965,
    rollout: 0.72,
    weekOverWeek: -18,
    durationMs: 1_842_000,
    scheduledOn: "2026-07-08",
    startedAt: "2026-07-08T01:00:00Z",
    updatedAt: "2026-07-08T01:30:42Z",
    runId: "exec-f6a7b8c9",
    linkLabel: "Job details",
    logsUrl: "https://example.com/runs/exec-f6a7b8c9",
  },
];

/** Table render config with one column per `WidgetColumnFormat`. */
export const columnFormatShowcaseRender: WidgetTableRender = {
  kind: "table",
  columns: [
    { field: "avatarUrl", label: "Avatar", format: "avatar" },
    { field: "summary", label: "Text", format: "text" },
    { field: "status", label: "Status", format: "status" },
    { field: "environment", label: "Badge", format: "badge" },
    { field: "requests", label: "Number", format: "number" },
    { field: "successRate", label: "Percent", format: "percent" },
    { field: "rollout", label: "Progress", format: "progress" },
    { field: "weekOverWeek", label: "Trend", format: "trend" },
    { field: "durationMs", label: "Duration", format: "duration" },
    { field: "scheduledOn", label: "Date", format: "date" },
    { field: "startedAt", label: "Datetime", format: "datetime" },
    { field: "updatedAt", label: "Relative", format: "relative" },
    { field: "runId", label: "Code", format: "code" },
    { field: "linkLabel", label: "Link", format: "link", href: "{{ logsUrl }}" },
  ],
  rowActions: [{ kind: "trigger", node: "deploy-prod", label: "Run", icon: "play", variant: "primary" }],
  rowStyles: [
    { field: "status", op: "eq", value: "failed", tone: "red-soft" },
    { field: "status", op: "eq", value: "running", tone: "blue-soft" },
    { field: "status", op: "eq", value: "cancelled", tone: "dimmed" },
  ],
  emptyMessage: "No runs to display.",
};

/** Wide panel size for the column-format showcase (grid w:12, h:15). */
export const columnFormatShowcasePanelSize = { width: 1040, height: 440 } as const;
