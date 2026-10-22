import type { IntakeRunOutcome, IntakeRunPreview } from "./intakeRunPreviewModel";

type ScheduledOutcome = Exclude<IntakeRunOutcome, "filtered" | "cancelled">;
const OUTCOMES: ScheduledOutcome[] = ["processing", "created", "no-proposal", "failed", "unknown"];
const SUMMARIES: Record<ScheduledOutcome, string> = {
  created: "The scheduled review proposed a task and added it to Backlog.",
  "no-proposal": "The scheduled review finished without proposing a task.",
  processing: "The agent is reviewing the repository for a useful task.",
  failed: "The scheduled review failed before it could propose a task.",
  unknown: "This run completed, but its proposal outcome is unavailable.",
};
const STATUS = {
  created: "Passed",
  "no-proposal": "Passed",
  processing: "Running",
  failed: "Failed",
  unknown: "Passed",
} as const;
const ERROR = "The agent could not fetch the repository.";

type Activity = IntakeRunPreview["activities"][number];

function scheduledAgent(outcome: ScheduledOutcome, startedAt: string, title: string): Activity {
  const endings: Record<ScheduledOutcome, string> = {
    created: `Proposed: ${title}\nSaved the proposal.`,
    "no-proposal": "No new task meets the review criteria.\nSaved an empty proposal.",
    processing: "Looking for a useful task…",
    failed: ERROR,
    unknown: "",
  };
  return {
    id: "scheduled-agent",
    name: "Agent",
    status: STATUS[outcome],
    recordedAt: startedAt,
    error: outcome === "failed" ? ERROR : undefined,
    logs: {
      state: "ready",
      text: `$ Review repository\nPreparing repository…\nReading project instructions…\nChecking recent changes and open tasks…\n${endings[outcome]}`,
    },
    outputs:
      outcome === "created" || outcome === "no-proposal"
        ? {
            passed: [
              {
                result:
                  outcome === "no-proposal"
                    ? {}
                    : { title, description: "Cover the expired-invitation case using the existing test helpers." },
              },
            ],
          }
        : undefined,
  };
}

function scheduledActivities(outcome: ScheduledOutcome, startedAt: string, title: string): Activity[] {
  const source: Activity = {
    id: "schedule",
    name: "Schedule",
    status: "Passed",
    recordedAt: startedAt,
    logs: { state: "empty" },
    outputs: { type: "schedule.tick", data: { timestamp: startedAt } },
    defaultTab: "outputs",
    outputsLabel: "Event data",
  };
  if (outcome === "unknown") return [source];
  const activities = [source, scheduledAgent(outcome, startedAt, title)];
  if (outcome === "created" || outcome === "no-proposal")
    activities.push({
      id: "has-proposal",
      name: "Has proposal?",
      status: "Passed",
      recordedAt: startedAt,
      logs: { state: "empty" },
      outputs: outcome === "no-proposal" ? { false: [{}] } : { true: [{}] },
      defaultTab: "outputs",
    });
  if (outcome === "created")
    activities.push({
      id: "create-task",
      name: "Create Task",
      status: "Passed",
      recordedAt: startedAt,
      logs: { state: "empty" },
      outputs: { default: [{ taskId: "task-1040" }] },
      defaultTab: "outputs",
    });
  return activities;
}

export function scheduledHistoryPreviewRuns(): IntakeRunPreview[] {
  return OUTCOMES.map((outcome, index) => {
    const startedAt = new Date(Date.now() - (index * 24 * 60 + 3) * 60_000).toISOString();
    const title =
      outcome === "created" ? "Add a missing regression test for expired invitations" : "Daily repository review";
    return {
      id: `${(0xb462987c - index * 0x123456).toString(16)}-12ba-4c15-b283-b88376c352a0`,
      source: { reference: "Daily review", title },
      event: "Schedule fired",
      outcome,
      summary: SUMMARIES[outcome],
      startedAt,
      finishedAt: outcome === "processing" ? undefined : new Date(Date.parse(startedAt) + 124_000).toISOString(),
      duration: outcome === "processing" ? "3m" : "2m 4s",
      executionStatus: STATUS[outcome],
      task: outcome === "created" ? { id: "SUPER-1040", title } : undefined,
      error: outcome === "failed" ? ERROR : undefined,
      activities: scheduledActivities(outcome, startedAt, title),
    };
  });
}
