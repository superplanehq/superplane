import type { AutomationRun, RunStatus } from "./types";

const titles = [
  "Use the default branch when starting a task",
  "Keep MCP tool timeouts as retries",
  "Remove the redundant task panel",
  "Show billing usage for the current period",
  "Use the skill name in the activity feed",
  "Keep the bot field when syncing tasks",
];
const taskIds = ["SP-8549", "SP-8578", "SP-8598", "SP-8606", "SP-8617", "SP-8593"];
const statuses: RunStatus[] = ["Running", "Failed", "Passed", "Passed", "Cancelled", "Passed"];
const prefixes = ["a8c19e42", "f13d08a6", "9ce27b10", "7b40d192", "36af812e", "c0945e12"];
const times = ["14:32", "14:06", "13:48", "12:21", "11:52", "10:40"];
const durations = ["2m 18s", "12m 42s", "8m 06s", "16m 11s", "3m 09s", "6m 32s"];
const implementationLog =
  "$ Run implementation agent\nPreparing repository…\nRepository ready.\nStarting agent…\nConnecting to MCP servers…";

export const implementationRuns: AutomationRun[] = titles.map((title, index) => {
  const status = statuses[index];
  const error = status === "Failed" ? "Agent process exited with code 1." : undefined;
  return {
    id: `${prefixes[index]}-7b41-4d2e-9c83-6a0295dce108`,
    task: { id: taskIds[index], title },
    status,
    started: `Today, ${times[index]}`,
    startedAt: `2026-10-19T${times[index]}:00Z`,
    finishedAt:
      status === "Running"
        ? undefined
        : new Date(
            Date.parse(`2026-10-19T${times[index]}:00Z`) + [138, 762, 486, 971, 189, 392][index] * 1000,
          ).toISOString(),
    duration: durations[index],
    error,
    activities: [
      {
        id: "start",
        name: "Run started",
        status: "Passed",
        time: times[index],
        logs: { state: "empty" },
        outputs: { task: { id: taskIds[index], title } },
      },
      {
        id: "implement",
        name: "Implement task",
        status,
        time: times[index],
        logs: {
          state: "ready",
          text: `${implementationLog}\n\n${error ? `Error: could not list MCP tools\n${error}` : status === "Running" ? "Inspecting repository files…" : status === "Cancelled" ? "Agent process stopped." : "Changes committed and pushed.\nAgent process exited with code 0."}`,
        },
        outputs:
          status === "Passed" ? { branch: `factory/${taskIds[index].toLowerCase()}`, result: "completed" } : undefined,
      },
      ...(error
        ? [
            {
              id: "record",
              name: "Record implementation failure",
              status: "Passed" as const,
              time: "14:18",
              logs: { state: "empty" as const },
              outputs: { result: "failed", message: error },
            },
          ]
        : []),
    ],
  };
});

export const verificationRuns: AutomationRun[] = implementationRuns.map((run, index) => ({
  ...run,
  id: `${["87fa4cf7", "f9f57a26", "941e2292", "b4438989", "24384c64", "0a2f43d8"][index]}-314a-4c15-b283-b88376c352a0`,
  task: index === 0 ? undefined : run.task,
  status: index === 0 ? "Passed" : index === 1 ? "Running" : run.status,
  error: index === 1 ? undefined : run.error,
  finishedAt: index === 0 ? "2026-10-19T14:32:02Z" : index === 1 ? undefined : run.finishedAt,
  duration: index === 0 ? "2s" : run.duration,
  activities:
    index === 0
      ? [
          {
            id: "review",
            name: "Pull request review received",
            status: "Passed",
            time: "14:32",
            logs: { state: "empty" },
          },
          {
            id: "find",
            name: "Find pull request",
            status: "Passed",
            time: "14:32",
            logs: { state: "empty" },
            outputs: { channel: "notFound", type: "pullRequest.notFound" },
          },
        ]
      : run.activities
          .filter((activity) => index !== 1 || activity.id !== "record")
          .map((activity) => ({
            ...activity,
            name: activity.id === "implement" ? "Address PR feedback" : activity.name,
            status: index === 1 && activity.id === "implement" ? "Running" : activity.status,
            logs:
              activity.id === "implement"
                ? {
                    state: "ready",
                    text:
                      "$ Address pull request feedback\nFetching review comments…\nReview comments loaded.\n" +
                      (index === 1 ? "Updating the requested changes…" : "Feedback processing finished."),
                  }
                : activity.logs,
          })),
}));
