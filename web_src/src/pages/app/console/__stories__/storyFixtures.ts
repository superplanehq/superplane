import type { SuperplaneComponentsNode } from "@/api-client";
import type { CanvasMemoryEntry } from "@/hooks/useCanvasData";

import type { ConsoleContextValue, ConsoleNodeStatus } from "../ConsoleContext";
import type { WidgetChartRender, WidgetNumberRender, WidgetTableRender } from "../widget/types";

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

/** Execution-like rows for table widgets (status, duration, owner, links).
 *
 * `cost_budget` is a per-row target used by the progress-column story
 * (`ManyColumnsAndFormats`) so authors can see the bar respond to different
 * budgets. `run-tests` intentionally overshoots the 60 s SLA to exercise the
 * bar-clamping / overshoot-label behaviour. */
export const executionRows: Record<string, unknown>[] = [
  {
    id: "exec-1",
    name: "deploy-prod",
    status: "passed",
    service: "api",
    durationMs: 42_000,
    cost: 12.5,
    cost_budget: 15,
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
    cost_budget: 15,
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
    cost_budget: 10,
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
    cost_budget: 5,
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
    cost_budget: 5,
    createdAt: "2026-06-26T07:20:00Z",
    url: "https://example.com/runs/exec-5",
  },
];

/**
 * Rows for the dedicated `ProgressColumn` story. Each row exercises a
 * different branch of the progress cell so the story doubles as a visual
 * regression reference:
 *
 * - `mid` — 5/10 with the default `percent` label.
 * - `almost-full` — 8/10 with the `number` label.
 * - `overshoot` — 12/10, clamps at 100% but keeps `120%` / `12/10` in text.
 * - `zero` — 0/10, empty fill.
 * - `empty` — null current + target, renders the em-dash placeholder.
 */
export const progressRows: Record<string, unknown>[] = [
  { id: "prog-1", label: "Almost there", done: 5, total: 10 },
  { id: "prog-2", label: "Nearly done", done: 8, total: 10 },
  { id: "prog-3", label: "Over budget", done: 12, total: 10 },
  { id: "prog-4", label: "Just started", done: 0, total: 10 },
  { id: "prog-5", label: "Not started", done: null, total: null },
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
    { field: "author", label: "Author", format: "avatar" },
    {
      field: "risk_score",
      label: "Risk",
      format: "progress",
      progressTarget: "100",
      progressLabel: "number",
    },
    { field: "risk_level", label: "Level", format: "status" },
    { field: "last_checked_at", label: "When", format: "relative" },
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

___

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
