import { useEffect, useRef, useState, type ReactNode } from "react";
import { ChevronRight, Loader2, Search } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Skeleton } from "@/ui/skeleton";
import { RunIdCopy } from "@/ui/AutomationRunHistory/RunIdentity";
import { RunTimestamp } from "@/ui/AutomationRunHistory/RunTimestamp";

export interface RunHistoryRowData {
  id: string;
  subject?: { key: string; title: string };
  status: string;
  statusContent: ReactNode;
  startedAt: string;
  duration: string;
  searchText?: string;
}

export function RunHistoryTable({
  runs,
  loading,
  error,
  onRetry,
  onSelectRun,
  onLoadMore,
  loadingMore,
  subjectsLoading,
}: {
  runs: RunHistoryRowData[];
  loading?: boolean;
  error?: string;
  onRetry?: () => void;
  onLoadMore?: () => void;
  loadingMore?: boolean;
  subjectsLoading?: boolean;
  onSelectRun: (id: string) => void;
}) {
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState("all");
  const statuses = Array.from(new Set(runs.map((run) => run.status)));
  const filtered = runs.filter((run) => matchesRun(run, query, status));
  return (
    <section aria-label="Run history" className="space-y-5">
      <div className="flex flex-wrap items-center gap-3">
        <div className="relative w-full sm:w-96">
          <Search aria-hidden className="pointer-events-none absolute left-3 top-2.5 size-4 text-muted-foreground" />
          <Input
            name="run-search"
            aria-label="Search task, source, or run ID"
            placeholder="Search task, source, or run ID"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            className="pl-9 max-sm:text-base"
          />
        </div>
        <Select value={status} onValueChange={setStatus}>
          <SelectTrigger aria-label="Filter by status" className="w-56">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All statuses</SelectItem>
            {statuses.map((value) => (
              <SelectItem key={value} value={value}>
                {value}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <div className="overflow-x-auto rounded-lg border border-border">
        <table className="w-full min-w-[56rem] table-fixed text-left text-sm" aria-label="Runs">
          <colgroup>
            <col className="w-32" />
            <col />
            <col className="w-48" />
            <col className="w-36" />
            <col className="w-24" />
          </colgroup>
          <thead className="border-b border-border bg-muted/30 text-xs text-muted-foreground">
            <tr>
              {["Run", "Task / source", "Status", "Started", "Duration"].map((label) => (
                <th key={label} scope="col" className="h-11 whitespace-nowrap px-4 font-medium">
                  {label}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <RunRowsSkeleton />
            ) : (
              !error &&
              filtered.map((run) => (
                <RunHistoryRow key={run.id} run={run} onSelectRun={onSelectRun} subjectsLoading={subjectsLoading} />
              ))
            )}
          </tbody>
        </table>
        {loading ? (
          <p role="status" className="sr-only">
            Loading run history.
          </p>
        ) : null}
        {!loading && error ? <RunHistoryError message={error} onRetry={onRetry} /> : null}
        {!loading && !error && filtered.length === 0 ? (
          <RunHistoryEmptyState
            hasRuns={runs.length > 0}
            onClear={() => {
              setQuery("");
              setStatus("all");
            }}
          />
        ) : null}
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

function matchesRun(run: RunHistoryRowData, query: string, status: string) {
  const text = [run.id, run.subject?.key, run.subject?.title, run.searchText].join(" ").toLowerCase();
  return (status === "all" || run.status === status) && text.includes(query.trim().toLowerCase());
}

function RunHistoryRow({
  run,
  onSelectRun,
  subjectsLoading,
}: {
  run: RunHistoryRowData;
  onSelectRun: (id: string) => void;
  subjectsLoading?: boolean;
}) {
  return (
    <tr
      className="group h-12 cursor-pointer border-b border-border last:border-0 hover:bg-muted/40"
      onClick={(event) => {
        if (event.target instanceof Element && event.target.closest("button,a")) return;
        onSelectRun(run.id);
      }}
    >
      <td className="px-4">
        <div className="flex items-center gap-2">
          <Button
            type="button"
            variant="link"
            aria-label={`Open run ${run.id}`}
            className="h-auto p-0 font-mono text-xs"
            onClick={() => onSelectRun(run.id)}
          >
            {run.id.slice(0, 8)}…
          </Button>
          <RunIdCopy id={run.id} />
        </div>
      </td>
      <td className="min-w-0 px-4">
        {run.subject ? (
          <Button
            type="button"
            variant="ghost"
            aria-label={`View ${run.subject.key}`}
            onClick={() => onSelectRun(run.id)}
            className="-mx-2 h-9 w-[calc(100%+1rem)] min-w-0 justify-start gap-3 px-2 text-left font-normal"
          >
            <code className="shrink-0 text-xs text-muted-foreground">{run.subject.key}</code>
            <span className="truncate" title={run.subject.title}>
              {run.subject.title}
            </span>
          </Button>
        ) : subjectsLoading ? (
          <div role="status" aria-label="Loading task">
            <Skeleton className="h-4 w-3/4 max-w-80" />
          </div>
        ) : (
          <span className="text-muted-foreground">No task or source</span>
        )}
      </td>
      <td className="whitespace-nowrap px-4">{run.statusContent}</td>
      <td className="whitespace-nowrap px-4 text-xs tabular-nums text-muted-foreground">
        <RunTimestamp value={run.startedAt} />
      </td>
      <td className="px-4 text-xs tabular-nums text-muted-foreground">
        <div className="flex items-center justify-between gap-2">
          <span>{run.duration}</span>
          <ChevronRight aria-hidden className="size-4 shrink-0 group-hover:text-foreground" />
        </div>
      </td>
    </tr>
  );
}

function RunRowsSkeleton() {
  return Array.from({ length: 6 }, (_, row) => (
    <tr key={row} aria-hidden className="h-12 border-b border-border last:border-0">
      {Array.from({ length: 5 }, (_, column) => (
        <td key={column} className="px-4">
          <Skeleton className="h-4 w-3/4" />
        </td>
      ))}
    </tr>
  ));
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
  const automatic = query.trim() === "" && status === "all";
  const sentinel = useRef<HTMLDivElement>(null);
  const loadMore = useRef(onLoadMore);
  loadMore.current = onLoadMore;
  useEffect(() => {
    if (!automatic || loading || !sentinel.current || typeof IntersectionObserver === "undefined") return;
    let requested = false;
    const observer = new IntersectionObserver((entries) => {
      if (requested || !entries.some((entry) => entry.isIntersecting)) return;
      requested = true;
      loadMore.current();
    });
    observer.observe(sentinel.current);
    return () => observer.disconnect();
  }, [automatic, loading, itemCount]);
  return (
    <div ref={sentinel} className="flex min-h-12 items-center justify-center text-sm text-muted-foreground">
      {loading ? (
        <div role="status" aria-label="Loading more runs" className="flex items-center gap-2">
          <Loader2 aria-hidden className="size-4 animate-spin motion-reduce:animate-none" />
          Loading more runs…
        </div>
      ) : (
        <Button variant="ghost" onClick={onLoadMore} className={automatic ? "sr-only focus:not-sr-only" : undefined}>
          Load more runs
        </Button>
      )}
    </div>
  );
}

function RunHistoryError({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <div role="alert" className="flex flex-col items-center gap-3 p-12 text-sm">
      <p>{message}</p>
      <Button type="button" variant="outline" size="sm" onClick={onRetry}>
        Try again
      </Button>
    </div>
  );
}

function RunHistoryEmptyState({ hasRuns, onClear }: { hasRuns: boolean; onClear: () => void }) {
  return (
    <div className="flex flex-col items-center gap-3 p-12 text-center text-sm">
      <p>{hasRuns ? "No matching runs" : "No runs yet"}</p>
      <p className="text-pretty text-muted-foreground">
        {hasRuns ? "Try another task, source, run ID, or status." : "Runs appear when this automation starts."}
      </p>
      {hasRuns ? (
        <Button type="button" variant="outline" size="sm" onClick={onClear}>
          Clear filters
        </Button>
      ) : null}
    </div>
  );
}
