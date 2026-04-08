import type { CanvasesCanvasEventWithExecutions } from "@/api-client";
import { TimeAgo } from "@/components/TimeAgo";
import { getAggregateStatus, getStatusBadgeProps } from "@/pages/workflowv2/lib/canvas-runs";
import { cn } from "@/lib/utils";
import { Link, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { toast } from "sonner";

interface RunsSidebarProps {
  events: CanvasesCanvasEventWithExecutions[];
  selectedEventId: string | null;
  onSelectRun: (eventId: string) => void;
  hasNextPage?: boolean;
  isFetchingNextPage?: boolean;
  onLoadMore?: () => void;
  isLoading?: boolean;
}

function RunStatusBadge({ status }: { status: string }) {
  const { badgeColor, label } = getStatusBadgeProps(status);
  return (
    <span
      className={cn(
        "shrink-0 uppercase text-[10px] py-[1px] px-[4px] font-semibold rounded tracking-wide text-white",
        badgeColor,
      )}
    >
      {label}
    </span>
  );
}

export function RunsSidebar({
  events,
  selectedEventId,
  onSelectRun,
  hasNextPage,
  isFetchingNextPage,
  onLoadMore,
  isLoading,
}: RunsSidebarProps) {
  return (
    <div className="flex w-64 shrink-0 flex-col border-r border-slate-200 bg-white">
      <div className="flex h-10 shrink-0 items-center border-b border-slate-200 px-3">
        <span className="text-sm font-medium text-gray-700">Runs</span>
      </div>

      <div className="flex-1 overflow-y-auto">
        {isLoading ? (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="h-5 w-5 animate-spin text-gray-400" />
          </div>
        ) : events.length === 0 ? (
          <div className="px-3 py-6 text-center text-xs text-gray-400">No runs yet</div>
        ) : (
          <>
            {events.map((event) => {
              const executions = event.executions || [];
              const status = executions.length > 0 ? getAggregateStatus(executions) : "queued";
              const isSelected = event.id === selectedEventId;

              return (
                <button
                  key={event.id}
                  type="button"
                  onClick={() => event.id && onSelectRun(event.id)}
                  className={cn(
                    "group flex w-full flex-col gap-1 border-b border-slate-100 px-3 py-2.5 text-left transition-colors",
                    isSelected ? "bg-sky-50" : "hover:bg-gray-50",
                  )}
                >
                  <div className="flex items-center gap-2">
                    <RunStatusBadge status={status} />
                    <span className="font-mono text-xs text-gray-500 truncate">#{event.id?.slice(0, 6)}</span>
                    <button
                      type="button"
                      title="Copy link to run"
                      className="ml-auto hidden shrink-0 rounded p-0.5 text-gray-400 hover:bg-gray-200 hover:text-gray-600 group-hover:block"
                      onClick={(e) => {
                        e.stopPropagation();
                        const url = new URL(window.location.href);
                        url.searchParams.set("run", event.id || "");
                        navigator.clipboard.writeText(url.toString());
                        toast.success("Run link copied");
                      }}
                    >
                      <Link className="h-3 w-3" />
                    </button>
                  </div>
                  <div className="flex items-center gap-1.5">
                    {event.customName ? (
                      <span className="text-xs font-medium text-gray-800 truncate">{event.customName}</span>
                    ) : null}
                    <span className="text-xs text-gray-400 tabular-nums whitespace-nowrap ml-auto">
                      {event.createdAt ? <TimeAgo date={event.createdAt} /> : ""}
                    </span>
                  </div>
                  {executions.length > 0 && (
                    <span className="text-[11px] text-gray-400">
                      {executions.length} {executions.length === 1 ? "step" : "steps"}
                    </span>
                  )}
                </button>
              );
            })}
            {hasNextPage && onLoadMore ? (
              <div className="px-3 py-2">
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  className="w-full text-xs"
                  onClick={onLoadMore}
                  disabled={isFetchingNextPage}
                >
                  {isFetchingNextPage ? <Loader2 className="mr-1 h-3 w-3 animate-spin" /> : null}
                  Load more
                </Button>
              </div>
            ) : null}
          </>
        )}
      </div>
    </div>
  );
}
