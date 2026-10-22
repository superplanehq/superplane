import type { AutomationRun } from "@/ui/AutomationRunHistory/types";

const TASKS = [
  "Keep plan drafts when switching tasks",
  "Allow a model override when creating a task",
  "Restore keyboard focus after saving settings",
];

export function planningHistoryPreviewRuns(): AutomationRun[] {
  return TASKS.map((title, index) => {
    const status = index === 0 ? "Running" : index === 1 ? "Passed" : "Failed";
    const startedAt = new Date(Date.now() - (index * 12 + 3) * 60_000).toISOString();
    const error = status === "Failed" ? "The planning agent could not read the repository." : undefined;
    return {
      id: `${(0xa3e1782d - index * 0x123456).toString(16)}-12ba-4c15-b283-b88376c352a0`,
      task: { id: `SUPER-${1001 + index}`, title },
      status,
      startedAt,
      finishedAt: status === "Running" ? undefined : new Date(Date.parse(startedAt) + 124_000).toISOString(),
      duration: status === "Running" ? "3m" : "2m 4s",
      error,
      activities: [
        {
          id: "on-task",
          name: "On Task",
          status: "Passed",
          recordedAt: startedAt,
          outputs: { workOrder: { id: `task-${1001 + index}`, title }, taskRefinementEnabled: true },
          defaultTab: "outputs",
          outputsLabel: "Event data",
        },
        {
          id: "refine-task-filter",
          name: "Refine task?",
          status: "Passed",
          recordedAt: startedAt,
          outputs: { true: [{}] },
          defaultTab: "outputs",
        },
        {
          id: "refine-task",
          name: "Refine Task",
          status,
          recordedAt: startedAt,
          error,
          logs: {
            state: "ready",
            text: `$ Plan task\nPreparing repository…\n${error ?? "Reading project instructions…\nInspecting the relevant code…\n" + (status === "Running" ? "Checking the proposed change against existing behavior…" : "Plan saved.\nClarity: green\nComplexity: green\nVerifiability: green")}`,
          },
          outputs: status === "Passed" ? { passed: [{ result: { title, planSaved: true } }] } : undefined,
        },
      ],
    };
  });
}
