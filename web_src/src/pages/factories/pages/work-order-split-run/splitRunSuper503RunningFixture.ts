import { buildSplitRunFooter } from "./splitRunFooter";
import type { SplitRunFixture } from "./splitRunMocks";
import { SPLIT_RUN_SUPER503, SUPER503_ANALYSIS_PHASE, SUPER503_BACKLOG_PHASE } from "./splitRunSuper503Fixture";
import { SUPER503_AUTOMATION_NAMES, SUPER503_IMPLEMENT_RUN, super503Phase } from "./splitRunSuper503Shared";
import { SUPER503_MODEL, super503ImplementRunningStream } from "./splitRunSuper503Streams";

/**
 * SUPER-503 sixteen minutes in. Backlog passed, Implement is running, and
 * nothing has reached Verify or Done. Spend is the partial Implement run;
 * production records none for the Backlog run.
 */

/** Implement about eight minutes in: no branch, no pull request yet. */
const IMPLEMENT_RUNNING_PHASE = super503Phase({
  id: "implement",
  name: "Implement",
  componentName: SUPER503_AUTOMATION_NAMES.implement,
  status: "running",
  duration: "8m 12s",
  startedAt: "11:37:08",
  stream: super503ImplementRunningStream(),
  ...SUPER503_IMPLEMENT_RUN,
  stepIndex: 0,
  costCents: "291",
  totalTokens: "4612370",
  model: SUPER503_MODEL,
});

export const SPLIT_RUN_SUPER503_RUNNING: SplitRunFixture = {
  ...SPLIT_RUN_SUPER503,
  elapsed: "16m so far",
  startedLabel: "Started 16m ago",
  costUsd: "$2.91",
  tokensLabel: "4.6M tokens",
  usageByModel: [{ provider: "openrouter", model: "x-ai/grok-4.6", totalTokens: "4612370", costCents: "291" }],
  lineStatus: "running",
  currentPhaseId: "implement",
  openPhaseId: "implement",
  phases: [SUPER503_BACKLOG_PHASE, SUPER503_ANALYSIS_PHASE, IMPLEMENT_RUNNING_PHASE],
  footer: buildSplitRunFooter({
    kind: "running",
    note: {
      key: "running-step",
      headline: "Implement is running",
      text: "Implementation Agent works on this step now. The log shows live progress.",
    },
    run: SUPER503_IMPLEMENT_RUN,
  }),
  footerTone: "running",
};
