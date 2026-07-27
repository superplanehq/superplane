import type { Meta, StoryObj } from "@storybook/react-vite";
import { fn } from "storybook/test";

import { FactoryHomePage } from "./FactoryHomePage";
import type { FactoryHomeData, FactoryStartingTask } from "./factoryHomeTypes";

/**
 * Project homepage for an installed Software Factory — where `Pages/HomePage →
 * Fresh Org` lands once **Setup Factory** finishes.
 *
 * Section order is deliberate: work blocked on a person first, then what the
 * agents are doing right now, then the outcome telemetry and audit trail that
 * say whether the factory earns more autonomy. All data arrives via props, so
 * every state below is a fixture rather than a live backend.
 */
const meta = {
  title: "Pages/Factory Home",
  component: FactoryHomePage,
  parameters: {
    layout: "fullscreen",
  },
} satisfies Meta<typeof FactoryHomePage>;

export default meta;

type Story = StoryObj<typeof meta>;

const minutesAgo = (minutes: number) => new Date(Date.now() - minutes * 60_000).toISOString();

const startingTasks: FactoryStartingTask[] = [
  {
    id: "fix-bug",
    label: "Find and fix a bug",
    description: "Scan for a high-confidence defect and ship a minimal fix.",
  },
  {
    id: "unit-test",
    label: "Improve test coverage",
    description: "Add one focused test to untested core logic.",
  },
  {
    id: "improve-ci",
    label: "Set up or improve CI",
    description: "One small improvement to the existing pipeline.",
  },
  {
    id: "improve-agents-md",
    label: "Improve AGENTS.md",
    description: "Refresh repo conventions so future runs start better informed.",
  },
];

const baseData: FactoryHomeData = {
  project: {
    name: "Payments API",
    description:
      "Agents pick up factory-labeled issues, open pull requests, and babysit CI until the checks are green.",
    repository: "acme/payments-api",
    repositoryUrl: "https://github.com/acme/payments-api",
    defaultBranch: "main",
    owner: "Platform team",
    health: "healthy",
    integrations: [
      { id: "github", label: "GitHub", connected: true },
      { id: "claude", label: "Claude", connected: true },
    ],
  },
  startingTasks,
  needsReview: [
    {
      id: "review-1",
      title: "Retry idempotency keys on gateway timeout",
      reason: "Agent finished and requested review before merge.",
      waitingSince: minutesAgo(42),
      pullRequest: { number: 1841, url: "https://github.com/acme/payments-api/pull/1841" },
    },
    {
      id: "review-2",
      title: "Add refund reconciliation test",
      reason: "Touches a protected path — merge needs an approver.",
      waitingSince: minutesAgo(190),
      pullRequest: { number: 1839, url: "https://github.com/acme/payments-api/pull/1839" },
    },
  ],
  inFlight: [
    {
      id: "run-1",
      title: "Fix currency rounding in settlement report",
      status: "running",
      stage: "Waiting on CI",
      branch: "factory/settlement-rounding",
      startedAt: minutesAgo(11),
      pullRequest: { number: 1843, url: "https://github.com/acme/payments-api/pull/1843" },
    },
    {
      id: "run-2",
      title: "Improve AGENTS.md",
      status: "running",
      stage: "Writing changes",
      branch: "factory/agents-md-refresh",
      startedAt: minutesAgo(3),
    },
    {
      id: "run-3",
      title: "Harden webhook signature check",
      status: "failed",
      stage: "CI failed — agent retrying",
      branch: "factory/webhook-signature",
      startedAt: minutesAgo(58),
      pullRequest: { number: 1840, url: "https://github.com/acme/payments-api/pull/1840" },
    },
  ],
  outcomes: [
    {
      id: "merged",
      label: "Pull requests merged",
      value: "34",
      delta: "+9",
      deltaPeriod: "prior 14 days",
      betterDirection: "up",
      trend: [1, 2, 2, 3, 2, 4, 3, 5, 4, 6, 5, 7],
    },
    {
      id: "lead-time",
      label: "Issue to merge",
      value: "3h 12m",
      delta: "-41m",
      deltaPeriod: "prior 14 days",
      betterDirection: "down",
      trend: [8, 7, 7, 6, 6, 5, 5, 4, 4, 4, 3, 3],
    },
    {
      id: "first-pass",
      label: "Merged without rework",
      value: "72%",
      delta: "+6%",
      deltaPeriod: "prior 14 days",
      betterDirection: "up",
      trend: [4, 5, 4, 5, 6, 5, 6, 6, 7, 6, 7, 8],
    },
    {
      id: "review-wait",
      label: "Median review wait",
      value: "48m",
      delta: "+12m",
      deltaPeriod: "prior 14 days",
      betterDirection: "down",
      trend: [3, 3, 4, 3, 4, 5, 4, 5, 6, 5, 6, 7],
    },
  ],
  activity: [
    { id: "a1", summary: "Merged #1838 — cache Stripe customer lookups", actor: "Factory agent", at: minutesAgo(26) },
    { id: "a2", summary: "Opened #1843 — fix currency rounding", actor: "Factory agent", at: minutesAgo(64) },
    { id: "a3", summary: "Approved #1836", actor: "dana@acme.com", at: minutesAgo(133) },
    { id: "a4", summary: "Closed #1830 — superseded by #1836", actor: "Factory agent", at: minutesAgo(410) },
  ],
};

/** Spies so every interaction is visible in the Actions panel. */
const handlers = {
  onStartTask: fn(),
  onOpenRun: fn(),
  onOpenReview: fn(),
  onResolveHealth: fn(),
};

/** Steady state: agents mid-flight, two pull requests parked on a human. */
export const Default: Story = {
  args: { data: baseData, ...handlers },
};

/**
 * The moment after **Setup Factory** completes — nothing has run yet, so every
 * panel shows its empty state and the only meaningful action is starting a task.
 */
export const FirstDay: Story = {
  args: {
    ...handlers,
    data: {
      ...baseData,
      needsReview: [],
      inFlight: [],
      activity: [],
      outcomes: baseData.outcomes.map((metric) => ({
        ...metric,
        value: "—",
        delta: undefined,
        deltaPeriod: undefined,
        trend: undefined,
      })),
    },
  },
};

/**
 * Error state with a resolution path: the GitHub connection expired, so runs
 * cannot open pull requests until someone reconnects it.
 */
export const NeedsAttention: Story = {
  args: {
    ...handlers,
    data: {
      ...baseData,
      project: {
        ...baseData.project,
        health: "degraded",
        healthDetail: "The GitHub connection expired — agents can commit but cannot open pull requests.",
        integrations: [
          { id: "github", label: "GitHub", connected: false },
          { id: "claude", label: "Claude", connected: true },
        ],
      },
      inFlight: baseData.inFlight.map((run) =>
        run.status === "running" ? { ...run, status: "failed" as const, stage: "Blocked — no GitHub access" } : run,
      ),
    },
  },
};
