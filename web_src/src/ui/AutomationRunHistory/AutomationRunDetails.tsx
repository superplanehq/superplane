import { useState, type ReactNode } from "react";
import { ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { CopyButton } from "@/ui/CopyButton";
import { LiveLogStateNotice } from "@/ui/CanvasPage/RunnerLiveLogDialog/LiveLogStateNotice";
import { RunIdCopy, RunStatusLabel } from "./RunIdentity";
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
}: {
  run: AutomationRun;
  onBack: () => void;
  onRetryLogs: (activityId: string) => void;
  renderLogs?: (activity: RunActivity) => ReactNode;
}) {
  const [selectedId, setSelectedId] = useState(
    run.activities.find(
      (activity) => activity.status === "Failed" || activity.status === "Running" || activity.status === "Pending",
    )?.id ?? run.activities[0]?.id,
  );
  const selected = run.activities.find((activity) => activity.id === selectedId);
  return (
    <section className="space-y-6">
      <Button variant="ghost" className="px-0 text-muted-foreground" onClick={onBack}>
        <ArrowLeft className="size-4" />
        Run history
      </Button>
      <div className="flex flex-wrap items-center gap-4">
        <h1 className="text-2xl font-semibold tracking-tight">{run.task?.title ?? run.title ?? "Run details"}</h1>
        <RunStatusLabel status={run.status} />
      </div>
      <div className="flex flex-wrap items-center justify-between gap-4">
        <span className="text-sm text-muted-foreground">{run.task ? `Task ${run.task.id}` : "No linked task"}</span>
        <RunIdCopy id={run.id} full />
      </div>
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
                aria-pressed={selectedId === activity.id}
                onClick={() => setSelectedId(activity.id)}
                className={`h-auto w-full rounded-md flex-col items-start gap-2 whitespace-normal px-3 py-4 text-left ${selectedId === activity.id ? "bg-muted" : ""}`}
              >
                <span>{activity.name}</span>
                <span className="flex w-full items-center justify-between gap-2">
                  <RunStatusLabel status={activity.status} />
                  <span className="text-xs font-normal text-muted-foreground">{activity.time}</span>
                </span>
              </Button>
            ))}
          </div>
          <p className="pt-3 text-xs leading-5 text-muted-foreground">Only activity recorded for this run is shown.</p>
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
            <Tabs defaultValue="logs">
              <TabsList>
                <TabsTrigger value="logs">Logs</TabsTrigger>
                <TabsTrigger value="outputs">Outputs</TabsTrigger>
              </TabsList>
              <TabsContent value="logs" className="pt-5">
                {renderLogs ? (
                  renderLogs(selected)
                ) : (
                  <RunLogContent logs={selected.logs} onRetry={() => onRetryLogs(selected.id)} />
                )}
              </TabsContent>
              <TabsContent value="outputs" className="pt-5">
                {selected.outputs ? (
                  <pre className="overflow-auto whitespace-pre-wrap break-words rounded-md border border-border bg-muted/30 p-5 text-xs leading-6">
                    {JSON.stringify(selected.outputs, null, 2)}
                  </pre>
                ) : (
                  <p className="p-12 text-center text-sm text-muted-foreground">
                    No outputs were recorded for this activity.
                  </p>
                )}
              </TabsContent>
            </Tabs>
          </div>
        ) : (
          <p className="text-sm text-muted-foreground">No activity was recorded for this run.</p>
        )}
      </div>
    </section>
  );
}
