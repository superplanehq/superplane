import { useState, type ReactNode } from "react";
import { ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { CopyButton } from "@/ui/CopyButton";
import { LiveLogStateNotice } from "@/ui/CanvasPage/RunnerLiveLogDialog/LiveLogStateNotice";
import { RunIdCopy, RunStatusLabel } from "./RunIdentity";
import { RunTimestamp } from "./RunTimestamp";
import { IntakeRunHeading } from "./IntakeRunHeading";
import { Skeleton } from "@/ui/skeleton";
import type { AutomationRun, RunLogs, RunActivity } from "./types";

function formatRunTime(value: string) {
  if (!value || !Number.isFinite(Date.parse(value))) return "—";
  return new Intl.DateTimeFormat("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false,
    timeZoneName: "short",
  }).format(new Date(value));
}

function RunLogContent({ logs, onRetry }: { logs: RunLogs; onRetry: () => void }) {
  if (logs.state === "error")
    return <LiveLogStateNotice state="error" error={logs.message} willRetry={false} onRetry={onRetry} />;
  if (logs.state === "empty")
    return <p className="p-12 text-center text-sm text-muted-foreground">No logs were recorded for this activity.</p>;
  if (logs.state !== "ready") return <LiveLogStateNotice state={logs.state} />;
  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between text-xs text-muted-foreground">
        <span>Runner log</span>
        <CopyButton text={logs.text} ariaLabel="Copy logs" />
      </div>
      <pre className="max-h-[480px] overflow-auto whitespace-pre-wrap break-words rounded-md border border-border bg-slate-950 p-5 font-mono text-xs leading-6 text-slate-100">
        {logs.text}
      </pre>
    </div>
  );
}

export function AutomationRunDetails({
  run,
  onBack,
  onRetryLogs,
  renderLogs,
  tasksLoading,
  selectedActivityId,
  onSelectActivity,
}: {
  run: AutomationRun;
  onBack: () => void;
  onRetryLogs: (activityId: string) => void;
  renderLogs?: (activity: RunActivity) => ReactNode;
  tasksLoading?: boolean;
  selectedActivityId?: string | null;
  onSelectActivity?: (id: string) => void;
}) {
  const [selectedId, setSelectedId] = useState<string>();
  const activityId = onSelectActivity ? selectedActivityId : selectedId;
  const selected =
    run.activities.find((activity) => activity.id === activityId) ??
    run.activities.find(
      (activity) => activity.status === "Failed" || activity.status === "Running" || activity.status === "Pending",
    ) ??
    run.activities[0];
  return (
    <section className="space-y-6">
      <Button variant="ghost" className="px-0 text-muted-foreground" onClick={onBack}>
        <ArrowLeft className="size-4" />
        Run history
      </Button>
      {run.intake ? <IntakeRunHeading run={run} /> : <RunDetailsHeader run={run} tasksLoading={tasksLoading} />}
      <dl className="flex flex-wrap gap-x-16 gap-y-4 border-y border-border py-5 text-sm">
        {[
          ["Started", formatRunTime(run.startedAt)],
          ["Finished", run.finishedAt ? formatRunTime(run.finishedAt) : "—"],
          ["Duration", run.duration],
        ].map(([label, value]) => (
          <div key={label}>
            <dt className="mb-2 text-xs text-muted-foreground">{label}</dt>
            <dd>{value}</dd>
          </div>
        ))}
      </dl>
      {run.error && (
        <div role="alert" className="rounded-md border border-red-400/30 bg-red-500/10 px-5 py-4">
          <p className="mb-2 text-sm font-medium text-red-700 dark:text-red-300">Run failed</p>
          <p className="font-mono text-xs">{run.error}</p>
        </div>
      )}
      <div className="grid gap-7 md:grid-cols-[280px_minmax(0,1fr)]">
        <aside className="space-y-4 md:border-r md:border-border md:pr-7">
          <h2 className="text-sm font-semibold">Activity</h2>
          <div className="space-y-2">
            {run.activities.map((activity) => (
              <Button
                key={activity.id}
                variant="ghost"
                aria-pressed={selected?.id === activity.id}
                onClick={() => {
                  setSelectedId(activity.id);
                  onSelectActivity?.(activity.id);
                }}
                className={`h-auto w-full rounded-md flex-col items-start gap-2 whitespace-normal px-3 py-4 text-left ${selected?.id === activity.id ? "bg-muted" : ""}`}
              >
                <span>{activity.name}</span>
                <span className="flex w-full items-center justify-between gap-2">
                  <RunStatusLabel status={activity.status} />
                  <span className="text-xs font-normal tabular-nums text-muted-foreground">
                    <RunTimestamp value={activity.recordedAt} runStartedAt={run.startedAt} />
                  </span>
                </span>
              </Button>
            ))}
          </div>
        </aside>
        {selected ? (
          <div className="min-w-0 space-y-5">
            <div className="flex items-center justify-between gap-4">
              <h2 className="font-semibold">{selected.name}</h2>
              <RunStatusLabel status={selected.status} />
            </div>
            {selected.error && (
              <p role="alert" className="text-sm text-destructive">
                {selected.error}
              </p>
            )}
            <RunActivityTabs activity={selected} renderLogs={renderLogs} onRetryLogs={onRetryLogs} />
          </div>
        ) : (
          <p className="text-sm text-muted-foreground">No activity was recorded for this run.</p>
        )}
      </div>
    </section>
  );
}

function RunActivityTabs({
  activity,
  renderLogs,
  onRetryLogs,
}: {
  activity: RunActivity;
  renderLogs?: (activity: RunActivity) => ReactNode;
  onRetryLogs: (activityId: string) => void;
}) {
  // Late execution metadata can identify a runner and change the default tab.
  return (
    <Tabs
      key={`${activity.id}:${Boolean(activity.logs)}`}
      defaultValue={activity.logs ? (activity.defaultTab ?? "logs") : "outputs"}
    >
      <RunActivityTabList activity={activity} />
      {activity.logs ? (
        <TabsContent value="logs" className="pt-5">
          {renderLogs ? (
            renderLogs(activity)
          ) : (
            <RunLogContent logs={activity.logs} onRetry={() => onRetryLogs(activity.id)} />
          )}
        </TabsContent>
      ) : null}
      {activity.additionalInstructions ? (
        <TabsContent value="instructions" className="space-y-3 pt-5">
          <p className="text-xs text-muted-foreground">Additional instructions used for this activity.</p>
          <pre className="whitespace-pre-wrap break-words rounded-md border border-border bg-muted/30 p-5 font-sans text-sm leading-6">
            {activity.additionalInstructions}
          </pre>
        </TabsContent>
      ) : null}
      <TabsContent value="outputs" className="pt-5">
        {activity.outputs ? (
          <pre className="overflow-auto whitespace-pre-wrap break-words rounded-md border border-border bg-muted/30 p-5 text-xs leading-6">
            {JSON.stringify(activity.outputs, null, 2)}
          </pre>
        ) : (
          <p className="p-12 text-center text-sm text-muted-foreground">No outputs were recorded for this activity.</p>
        )}
      </TabsContent>
    </Tabs>
  );
}

function RunActivityTabList({ activity }: { activity: RunActivity }) {
  const tabs = activity.logs
    ? activity.defaultTab === "outputs"
      ? ["outputs", "logs"]
      : ["logs", "outputs"]
    : ["outputs"];
  return (
    <TabsList>
      {tabs.map((tab) => (
        <TabsTrigger key={tab} value={tab}>
          {tab === "logs" ? "Logs" : (activity.outputsLabel ?? "Outputs")}
        </TabsTrigger>
      ))}
      {activity.additionalInstructions ? <TabsTrigger value="instructions">Instructions</TabsTrigger> : null}
    </TabsList>
  );
}

function RunDetailsHeader({ run, tasksLoading }: { run: AutomationRun; tasksLoading?: boolean }) {
  return (
    <>
      <div className="flex flex-wrap items-center gap-4">
        {tasksLoading && !run.task ? (
          <Skeleton className="h-8 w-80 max-w-full" aria-label="Loading task" role="status" />
        ) : (
          <h1 className="text-2xl font-semibold tracking-tight">{run.task?.title ?? run.title ?? "Run details"}</h1>
        )}
        <RunStatusLabel status={run.status} />
      </div>
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          {run.task ? (
            <>
              <span>
                Task <code className="font-mono">{run.task.id}</code>
              </span>
              <CopyButton
                text={run.task.id}
                ariaLabel={`Copy task ID ${run.task.id}`}
                copiedAriaLabel="Task ID copied"
              />
            </>
          ) : tasksLoading ? (
            <Skeleton className="h-4 w-24" />
          ) : (
            "No linked task"
          )}
        </div>
        <RunIdCopy id={run.id} full />
      </div>
    </>
  );
}
