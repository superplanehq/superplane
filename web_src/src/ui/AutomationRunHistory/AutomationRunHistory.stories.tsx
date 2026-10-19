import { withFactoriesTheme } from "@/pages/factories/__fixtures__/factoriesStoryTheme";
import { useEffect, useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { AutomationRunHistory } from "./AutomationRunHistory";
import { AutomationRunDetails } from "./AutomationRunDetails";
import { implementationRuns, verificationRuns } from "./fixtures";
import type { AutomationRun, RunLogs } from "./types";

interface PreviewProps {
  automation?: string;
  runs?: AutomationRun[];
  initialRun?: AutomationRun;
  loading?: boolean;
  error?: string;
  tasksLoading?: boolean;
  pagination?: "automatic" | "loading";
}

function HistoryPreview({
  automation = "Implement",
  runs = implementationRuns,
  initialRun,
  loading,
  error,
  tasksLoading,
  pagination,
}: PreviewProps) {
  const [selected, setSelected] = useState(initialRun);
  const [historyError, setHistoryError] = useState(error);
  const [rowCount, setRowCount] = useState(25);
  const [loadingMore, setLoadingMore] = useState(pagination === "loading");
  useEffect(() => {
    if (!loadingMore || pagination !== "automatic") return;
    const timer = window.setTimeout(() => {
      setRowCount((count) => count + 25);
      setLoadingMore(false);
    }, 1200);
    return () => window.clearTimeout(timer);
  }, [loadingMore, pagination]);
  const displayedRuns =
    pagination === "automatic"
      ? Array.from({ length: Math.min(rowCount, 75) }, (_, index) => ({
          ...runs[index % runs.length],
          id: `${(index + 1).toString(16).padStart(8, "0")}-7b41-4d2e-9c83-6a0295dce108`,
        }))
      : runs;
  return (
    <main className="min-h-screen bg-background px-6 py-10 text-foreground md:px-16">
      <div className="mx-auto max-w-[1440px] space-y-7">
        <div className="flex flex-wrap justify-between gap-3 text-xs text-muted-foreground">
          <span>SuperPlane Prod / Automations / {automation}</span>
        </div>
        {selected ? (
          <AutomationRunDetails
            key={selected.id}
            run={selected}
            tasksLoading={tasksLoading}
            onBack={() => setSelected(undefined)}
            onRetryLogs={(activityId) => {
              console.info("Retry logs", activityId);
              setSelected({
                ...selected,
                activities: selected.activities.map((activity) =>
                  activity.id === activityId
                    ? {
                        ...activity,
                        logs: {
                          state: "ready",
                          text: `Saved runner output loaded.\n${selected.error ?? "Agent output is available."}`,
                        },
                      }
                    : activity,
                ),
              });
            }}
          />
        ) : (
          <>
            <h1 className="text-2xl font-semibold tracking-tight">{automation}</h1>
            <div className="border-b border-border pb-3 text-sm font-medium">Run history</div>
            <AutomationRunHistory
              runs={displayedRuns}
              tasksLoading={tasksLoading}
              loadingMore={loadingMore}
              onLoadMore={
                pagination && (pagination === "loading" || rowCount < 75) ? () => setLoadingMore(true) : undefined
              }
              loading={loading}
              error={historyError}
              onRetry={() => {
                console.info("Retry history");
                setHistoryError(undefined);
              }}
              onSelectRun={(run) => {
                console.info("Open run", run.id);
                setSelected(run);
              }}
            />
          </>
        )}
      </div>
    </main>
  );
}

const meta = {
  title: "Factories/Pages/Automation run history",
  decorators: [withFactoriesTheme],
  component: HistoryPreview,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof HistoryPreview>;
export default meta;
type Story = StoryObj<typeof meta>;
export const Implement: Story = {};
export const AddressPRFeedback: Story = { args: { automation: "Address PR feedback", runs: verificationRuns } };
export const FailedRun: Story = { args: { initialRun: implementationRuns[1] } };
export const RunningRun: Story = { args: { initialRun: implementationRuns[0] } };
export const PassedRun: Story = { args: { initialRun: implementationRuns[2] } };
export const CancelledRun: Story = { args: { initialRun: implementationRuns[4] } };
export const NoLinkedTask: Story = {
  args: { automation: "Address PR feedback", runs: verificationRuns, initialRun: verificationRuns[0] },
};
export const EmptyHistory: Story = { args: { runs: [] } };
export const LoadingHistory: Story = { args: { runs: [], loading: true } };
export const LoadingTaskLinks: Story = {
  args: { tasksLoading: true, runs: implementationRuns.map((run) => ({ ...run, task: undefined })) },
};
export const AutomaticPagination: Story = { args: { pagination: "automatic" } };
export const LoadingMoreRuns: Story = { args: { pagination: "loading" } };
export const OvernightRun: Story = {
  args: {
    initialRun: {
      ...implementationRuns[2],
      startedAt: new Date(new Date().setHours(0, 0, 0, 0) - 60 * 60 * 1000).toISOString(),
      activities: implementationRuns[2].activities.map((activity, index) => ({
        ...activity,
        recordedAt: new Date(new Date().setHours(0, 0, 0, 0) + (index === 0 ? -60 : 30) * 60 * 1000).toISOString(),
      })),
    },
  },
};
export const HistoryError: Story = { args: { error: "Could not load run history. Try again." } };
function withLogs(logs: RunLogs): AutomationRun {
  return {
    ...implementationRuns[1],
    activities: implementationRuns[1].activities.map((activity) =>
      activity.id === "implement" ? { ...activity, logs } : activity,
    ),
  };
}
export const LoadingLogs: Story = { args: { initialRun: withLogs({ state: "loading" }) } };
export const UnavailableLogs: Story = {
  args: { initialRun: withLogs({ state: "error", message: "Saved runner logs could not be retrieved." }) },
};
export const EmptyLogs: Story = { args: { initialRun: withLogs({ state: "empty" }) } };
export const WaitingForLogs: Story = {
  args: {
    initialRun: {
      ...implementationRuns[0],
      activities: implementationRuns[0].activities.map((activity) =>
        activity.id === "implement" ? { ...activity, logs: { state: "waiting" } } : activity,
      ),
    },
  },
};
export const LongTaskTitle: Story = {
  args: {
    runs: [
      {
        ...implementationRuns[2],
        task: {
          id: "SP-8598",
          title:
            "Keep the task panel readable when an automation receives a task with a very long title from an external source and additional context",
        },
      },
    ],
  },
};
