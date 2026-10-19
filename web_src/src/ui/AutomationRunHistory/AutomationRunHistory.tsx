import { useEffect, useRef, useState } from "react";
import { Loader2, Search } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { RunIdCopy, RunStatusLabel } from "./RunIdentity";
import { Skeleton } from "@/ui/skeleton";
import { RunTimestamp } from "./RunTimestamp";
import type { AutomationRun } from "./types";

interface AutomationRunHistoryProps {
  runs: AutomationRun[];
  onSelectRun: (run: AutomationRun) => void;
  loading?: boolean;
  error?: string;
  onRetry?: () => void;
  onLoadMore?: () => void;
  loadingMore?: boolean;
  tasksLoading?: boolean;
}

export function AutomationRunHistory({
  runs,
  onSelectRun,
  loading,
  error,
  onRetry,
  onLoadMore,
  loadingMore,
  tasksLoading,
}: AutomationRunHistoryProps) {
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState("all");
  const filtered = runs.filter(
    (run) =>
      (status === "all" || status === run.status) &&
      `${run.id} ${run.task?.id ?? ""} ${run.task?.title ?? ""}`.toLowerCase().includes(query.trim().toLowerCase()),
  );
  return (
    <section aria-label="Run history" className="space-y-5">
      <div className="flex flex-wrap items-center gap-3">
        <div className="relative w-80">
          <Search aria-hidden className="absolute left-3 top-2.5 size-4 text-muted-foreground" />
          <Input
            aria-label="Search task or run ID"
            placeholder="Search task or run ID"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            className="pl-9"
          />
        </div>
        <Select value={status} onValueChange={setStatus}>
          <SelectTrigger aria-label="Filter by status" className="w-40">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {["all", "Pending", "Running", "Cancelling", "Passed", "Failed", "Cancelled", "Unknown"].map((value) => (
              <SelectItem key={value} value={value}>
                {value === "all" ? "All statuses" : value}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <div className="overflow-x-auto rounded-lg border border-border">
        <table className="w-full min-w-[850px] table-fixed text-left text-sm">
          <colgroup>
            <col className="w-40" />
            <col />
            <col className="w-36" />
            <col className="w-44" />
            <col className="w-28" />
          </colgroup>
          <thead className="border-b border-border bg-muted/30 text-xs text-muted-foreground">
            <tr>
              {["Run", "Task", "Status", "Started", "Duration"].map((label) => (
                <th key={label} scope="col" className="h-11 px-5 font-medium">
                  {label}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {loading && <RunHistorySkeleton />}
            {!loading &&
              !error &&
              filtered.map((run) => (
                <RunHistoryRow key={run.id} run={run} onSelectRun={onSelectRun} tasksLoading={tasksLoading} />
              ))}
          </tbody>
        </table>
        {loading && (
          <p role="status" className="sr-only">
            Loading run history…
          </p>
        )}
        {error && (
          <div role="alert" className="space-y-3 p-12 text-center text-sm">
            <p>{error}</p>
            <Button variant="outline" onClick={onRetry}>
              Try again
            </Button>
          </div>
        )}
        {!loading && !error && filtered.length === 0 && (
          <div className="space-y-2 p-12 text-center text-sm">
            <p>{runs.length ? "No matching runs" : "No runs yet"}</p>
            <p className="text-muted-foreground">
              {runs.length
                ? "Try a different task, run ID, or status."
                : "Runs will appear when this automation starts."}
            </p>
            {runs.length > 0 && (
              <Button
                variant="outline"
                onClick={() => {
                  setQuery("");
                  setStatus("all");
                }}
              >
                Clear filters
              </Button>
            )}
          </div>
        )}
      </div>
      {onLoadMore && !loading && !error && (
        <LoadMoreRuns
          onLoadMore={onLoadMore}
          loading={loadingMore}
          itemCount={runs.length}
          query={query}
          status={status}
        />
      )}
    </section>
  );
}

function RunHistoryRow({
  run,
  onSelectRun,
  tasksLoading,
}: {
  run: AutomationRun;
  onSelectRun: (run: AutomationRun) => void;
  tasksLoading?: boolean;
}) {
  return (
    <tr className="h-12 border-b border-border last:border-0 hover:bg-muted/40">
      <td className="px-5">
        <div className="flex items-center gap-2">
          <Button
            variant="link"
            onClick={() => onSelectRun(run)}
            className="h-auto p-0 font-mono text-xs"
            aria-label={`Open run ${run.id}`}
          >
            {run.id.slice(0, 8)}…
          </Button>
          <RunIdCopy id={run.id} />
        </div>
      </td>
      <td className="px-5">
        {run.task ? (
          <Button
            variant="ghost"
            onClick={() => onSelectRun(run)}
            className="-mx-2 h-auto w-[calc(100%+1rem)] justify-start gap-3 px-2 py-1 text-left font-normal"
          >
            <span className="shrink-0 font-mono text-xs text-muted-foreground">{run.task.id}</span>
            <span className="truncate" title={run.task.title}>
              {run.task.title}
            </span>
          </Button>
        ) : tasksLoading ? (
          <div role="status" aria-label="Loading task">
            <Skeleton className="h-4 w-3/4 max-w-80" />
          </div>
        ) : (
          <span className="text-muted-foreground">No linked task</span>
        )}
      </td>
      <td className="px-5">
        <RunStatusLabel status={run.status} />
      </td>
      <td className="whitespace-nowrap px-5 text-muted-foreground">
        <RunTimestamp value={run.startedAt} />
      </td>
      <td className="px-5 tabular-nums text-muted-foreground">{run.duration}</td>
    </tr>
  );
}

function LoadMoreRuns({
  onLoadMore,
  loading,
  itemCount,
  query,
  status,
}: {
  onLoadMore: () => void;
  loading?: boolean;
  itemCount: number;
  query: string;
  status: string;
}) {
  const sentinel = useRef<HTMLDivElement>(null);
  const loadMore = useRef(onLoadMore);
  loadMore.current = onLoadMore;
  useEffect(() => {
    if (loading || !sentinel.current || typeof IntersectionObserver === "undefined") return;
    let requested = false;
    const observer = new IntersectionObserver((entries) => {
      if (requested || !entries.some((entry) => entry.isIntersecting)) return;
      requested = true;
      loadMore.current();
    });
    observer.observe(sentinel.current);
    return () => observer.disconnect();
  }, [loading, itemCount, query, status]);
  return (
    <div ref={sentinel} className="flex min-h-12 items-center justify-center text-sm text-muted-foreground">
      {loading ? (
        <div role="status" aria-label="Loading more runs" className="flex items-center gap-2">
          <Loader2 aria-hidden className="size-4 animate-spin motion-reduce:animate-none" />
          Loading more runs…
        </div>
      ) : (
        <Button variant="ghost" onClick={onLoadMore} className="sr-only focus:not-sr-only">
          Load more runs
        </Button>
      )}
    </div>
  );
}

function RunHistorySkeleton() {
  return Array.from({ length: 6 }, (_, index) => (
    <tr key={index} aria-hidden className="h-12 border-b border-border last:border-0">
      {Array.from({ length: 5 }, (_, column) => (
        <td key={column} className="px-5">
          <Skeleton className="h-4 w-3/4" />
        </td>
      ))}
    </tr>
  ));
}
