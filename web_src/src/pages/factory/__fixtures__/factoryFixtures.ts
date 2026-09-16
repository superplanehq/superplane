import type { Automation, SoftwareFactoryPageData, VelocityData, WorkOrder, WorkOrderEvent } from "../factoryTypes";

const hoursAgo = (hours: number) => new Date(Date.now() - hours * 3_600_000).toISOString();
const daysAgo = (days: number) => new Date(Date.now() - days * 86_400_000).toISOString();

const pr = (repository: string, number: number) => ({
  provider: "github" as const,
  repository,
  number,
  url: `https://github.com/${repository}/pull/${number}`,
});

export const workOrders: WorkOrder[] = [
  {
    id: "wo-1",
    title: "Retry idempotency keys on gateway timeout",
    description:
      "Payments occasionally double-charge when the gateway times out and our client retries with a fresh idempotency key. Reuse one key per logical request.",
    state: "ready",
    attention: { reason: "Agent is asking whether to change retry behaviour for all callers.", since: hoursAgo(3) },
    currentAutomation: "Issue to pull request",
    pullRequests: [],
    createdAt: hoursAgo(6),
    updatedAt: hoursAgo(3),
  },
  {
    id: "wo-2",
    title: "Add refund reconciliation test",
    description: "Cover the refund reconciliation path — it regressed twice this quarter with no test to catch it.",
    state: "draft",
    attention: { reason: "Draft is waiting for approval before an Automation can pick it up.", since: hoursAgo(9) },
    pullRequests: [],
    createdAt: hoursAgo(9),
    updatedAt: hoursAgo(9),
  },
  {
    id: "wo-3",
    title: "Fix currency rounding in settlement report",
    description: "Settlement totals drift by cents because rounding happens per line rather than per settlement.",
    state: "ready",
    activity: "Waiting for CI on factory/settlement-rounding",
    currentAutomation: "Issue to pull request",
    pullRequests: [pr("acme/payments-api", 1843)],
    createdAt: hoursAgo(5),
    updatedAt: hoursAgo(1),
  },
  {
    id: "wo-4",
    title: "Harden webhook signature check",
    description: "Reject webhooks whose signature header is missing rather than treating them as unsigned.",
    state: "ready",
    activity: "Implementing changes",
    currentAutomation: "Security follow-ups",
    pullRequests: [],
    createdAt: hoursAgo(2),
    updatedAt: hoursAgo(1),
  },
  {
    id: "wo-5",
    title: "Cache Stripe customer lookups",
    description: "Customer lookups dominate checkout latency. Add a short-lived cache.",
    state: "successful",
    pullRequests: [pr("acme/payments-api", 1838)],
    createdAt: daysAgo(2),
    updatedAt: hoursAgo(20),
  },
  {
    id: "wo-6",
    title: "Migrate ledger export to the new schema",
    description: "Move the nightly ledger export onto the v2 schema and keep the old columns for one release.",
    state: "successful",
    pullRequests: [pr("acme/payments-api", 1829), pr("acme/ledger-service", 412)],
    createdAt: daysAgo(3),
    updatedAt: daysAgo(1),
  },
  {
    id: "wo-7",
    title: "Backfill missing merchant categories",
    description: "Fill in merchant category codes for records imported before the field existed.",
    state: "unsuccessful",
    pullRequests: [],
    createdAt: daysAgo(4),
    updatedAt: daysAgo(2),
  },
];

export const automations: Automation[] = [
  {
    id: "auto-1",
    name: "Issue to pull request",
    description: "Picks up factory-labeled issues, implements them, and opens a pull request.",
    trigger: "GitHub issue labeled `factory`",
    status: "active",
    currentActivity: "Running 2 Work Orders",
    recentSuccess: 0.82,
    lastRunAt: hoursAgo(1),
    repositories: ["acme/payments-api"],
  },
  {
    id: "auto-2",
    name: "Ready Work Order intake",
    description: "Listens for Work Orders that become ready and routes them to the right implementation Automation.",
    trigger: "Work Order → ready",
    status: "active",
    recentSuccess: 0.96,
    lastRunAt: hoursAgo(3),
    repositories: ["acme/payments-api", "acme/ledger-service"],
  },
  {
    id: "auto-3",
    name: "Security follow-ups",
    description: "Turns findings from the weekly scan into Work Orders and implements the low-risk ones.",
    trigger: "Schedule — weekly",
    status: "active",
    currentActivity: "Implementing 1 Work Order",
    recentSuccess: 0.74,
    lastRunAt: hoursAgo(2),
    repositories: ["acme/payments-api"],
  },
  {
    id: "auto-4",
    name: "Ledger schema migration",
    description: "One-off migration helper for the v2 ledger schema.",
    trigger: "Manual",
    status: "paused",
    recentSuccess: 1,
    lastRunAt: daysAgo(1),
    repositories: ["acme/ledger-service"],
  },
];

const throughputSeries: VelocityData["throughputSeries"] = [
  { date: "Jul 15", human: 3, factory: 1 },
  { date: "Jul 16", human: 2, factory: 2 },
  { date: "Jul 17", human: 4, factory: 1 },
  { date: "Jul 18", human: 1, factory: 3 },
  { date: "Jul 19", human: 0, factory: 0 },
  { date: "Jul 20", human: 0, factory: 1 },
  { date: "Jul 21", human: 3, factory: 2 },
  { date: "Jul 22", human: 2, factory: 3 },
  { date: "Jul 23", human: 3, factory: 2 },
  { date: "Jul 24", human: 1, factory: 4 },
  { date: "Jul 25", human: 2, factory: 3 },
  { date: "Jul 26", human: 0, factory: 1 },
  { date: "Jul 27", human: 1, factory: 2 },
  { date: "Jul 28", human: 2, factory: 4 },
];

export const velocity: VelocityData = {
  periodDays: 14,
  repositories: ["acme/payments-api", "acme/ledger-service"],
  selectedRepository: "acme/payments-api",
  cohorts: [
    {
      id: "team",
      label: "Team total",
      mergedPullRequests: 53,
      cycleTimeHours: 9,
      successRate: 0.89,
      trackedCostUsd: 41.18,
    },
    // PRD: human tracked cost is deliberately unavailable, never zero.
    {
      id: "human",
      label: "Human-authored",
      mergedPullRequests: 24,
      cycleTimeHours: 14,
      successRate: 0.92,
      trackedCostUsd: null,
    },
    {
      id: "factory",
      label: "Factory-authored",
      mergedPullRequests: 29,
      cycleTimeHours: 4,
      successRate: 0.86,
      trackedCostUsd: 41.18,
    },
  ],
  throughputSeries,
  costBreakdown: { tokensUsd: 28.4, computeUsd: 12.78 },
};

export const factoryPageData: SoftwareFactoryPageData = {
  factory: {
    id: "factory-1",
    name: "Payments Factory",
    description:
      "Delegated implementation work for the payments platform. Automations pick up labeled issues, open pull requests, and record the outcome on each Work Order.",
    status: "healthy",
    automationCount: automations.length,
  },
  summary: {
    throughput: { value: 29, unit: "Work Orders completed, 14 days", trend: [1, 2, 2, 3, 2, 4, 3, 5, 4, 6, 5, 7] },
    successRate: { value: 0.86, trend: [4, 5, 4, 5, 6, 5, 6, 6, 7, 6, 7, 8] },
    activeWorkOrders: 4,
    trackedCost: { tokensUsd: 28.4, computeUsd: 12.78 },
  },
  workOrders,
  automations,
  velocity,
};

/** A Factory immediately after creation — the PRD's "blank by default" state. */
export const newFactoryPageData: SoftwareFactoryPageData = {
  factory: {
    id: "factory-2",
    name: "Ledger Factory",
    description: undefined,
    status: "healthy",
    automationCount: 0,
  },
  summary: {
    throughput: { value: 0, unit: "Work Orders completed, 14 days", trend: [] },
    successRate: { value: 0, trend: [] },
    activeWorkOrders: 0,
    trackedCost: { tokensUsd: 0, computeUsd: 0 },
  },
  workOrders: [],
  automations: [],
  velocity: {
    ...velocity,
    cohorts: velocity.cohorts.map((cohort) => ({
      ...cohort,
      mergedPullRequests: 0,
      cycleTimeHours: 0,
      successRate: 0,
      trackedCostUsd: cohort.id === "human" ? null : 0,
    })),
    throughputSeries: throughputSeries.map((day) => ({ ...day, human: 0, factory: 0 })),
    costBreakdown: { tokensUsd: 0, computeUsd: 0 },
  },
};

const agent = { kind: "automation" as const, name: "Issue to pull request" };
const person = { kind: "human" as const, name: "dana@acme.com" };
const system = { kind: "system" as const, name: "SuperPlane" };

/** A full attempt that succeeded, then a reopen that appended a second attempt. */
export const workOrderEvents: WorkOrderEvent[] = [
  {
    id: "e1",
    kind: "created",
    at: hoursAgo(6),
    actor: system,
    summary: "Work Order created from issue #1822",
    automation: "Issue to pull request",
  },
  { id: "e2", kind: "approved", at: hoursAgo(5.6), actor: person, summary: "Approved — moved to ready" },
  {
    id: "e3",
    kind: "pickup",
    at: hoursAgo(5.5),
    actor: agent,
    summary: "Picked up by Issue to pull request",
    automation: "Issue to pull request",
  },
  {
    id: "e4",
    kind: "progress",
    at: hoursAgo(5.2),
    actor: agent,
    summary: "Read the gateway client and its retry wrapper",
    body: "internal/gateway/client.go — retry wrapper generates a fresh key per attempt\ninternal/gateway/idempotency.go — key derived from attempt number",
    automation: "Issue to pull request",
  },
  {
    id: "e5",
    kind: "approval-request",
    at: hoursAgo(4.5),
    actor: agent,
    summary: "Asked whether to change retry behaviour for all callers",
    body: "Reusing one key per logical request means a 409 from the gateway now signals a genuine duplicate rather than a retry artifact. That changes behaviour for every caller. Proceed?",
    automation: "Issue to pull request",
  },
  {
    id: "e6",
    kind: "steering",
    at: hoursAgo(4.2),
    actor: person,
    summary: "Answered the question",
    body: "Yes, reuse the key — but don't touch the retry ceiling, it's tuned for the gateway's rate limit.",
  },
  {
    id: "e7",
    kind: "decision",
    at: hoursAgo(4.1),
    actor: agent,
    summary: "Scope narrowed to key reuse and 409 handling",
    automation: "Issue to pull request",
  },
  {
    id: "e8",
    kind: "progress",
    at: hoursAgo(3.4),
    actor: agent,
    summary: "Implemented key reuse and duplicate handling",
    automation: "Issue to pull request",
  },
  {
    id: "e9",
    kind: "pull-request",
    at: hoursAgo(3),
    actor: agent,
    summary: "Opened a pull request",
    pullRequest: pr("acme/payments-api", 1841),
    automation: "Issue to pull request",
  },
  {
    id: "e10",
    kind: "outcome",
    at: hoursAgo(2.8),
    actor: agent,
    summary: "Marked successful",
    outcome: "successful",
    automation: "Issue to pull request",
  },
  {
    id: "e11",
    kind: "retry",
    at: hoursAgo(1.5),
    actor: person,
    summary: "Reopened — review found the 409 path untested",
    body: "The duplicate branch has no coverage. Reopening rather than filing a follow-up so the history stays in one place.",
  },
  {
    id: "e12",
    kind: "pickup",
    at: hoursAgo(1.4),
    actor: agent,
    summary: "Picked up again",
    automation: "Issue to pull request",
  },
  {
    id: "e13",
    kind: "progress",
    at: hoursAgo(0.5),
    actor: agent,
    summary: "Adding a table test for the duplicate path",
    automation: "Issue to pull request",
  },
];
