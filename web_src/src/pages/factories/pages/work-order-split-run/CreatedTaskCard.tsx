import { ArrowUpRight } from "lucide-react";

import { Link } from "@/components/Link/link";

import { WorkOrderStatusIcon } from "../../workOrders/WorkOrderStatusIcon";
import type { CreateWithAgentTaskMessage } from "../createWithAgentTypes";

/** Builds the permalink for a task the agent created; `undefined` when the task has no number yet. */
export type CreatedTaskHref = (task: CreateWithAgentTaskMessage) => string | undefined;

/**
 * A task the agent split off the draft, shown in the chat at the moment it
 * was created. It reads like a compact board card on the agent side so the
 * user can open it without leaving the thread.
 */
export function CreatedTaskCard({ task, href }: { task: CreateWithAgentTaskMessage; href?: string }) {
  return (
    <div className="sp-text-reveal flex w-full px-2 py-1.5" data-testid="split-run-intent-created-task">
      <div className="flex min-w-0 max-w-[85%] flex-1 items-center gap-3 rounded-xl border bg-card px-3 py-2.5 shadow-xs">
        <WorkOrderStatusIcon status="draft" aria-hidden />
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-1.5 text-[11px] leading-4 text-muted-foreground">
            <span>New task</span>
            {task.key ? (
              <>
                <span aria-hidden>·</span>
                <span className="font-mono">{task.key}</span>
              </>
            ) : null}
          </div>
          <div className="truncate text-[14px] font-medium leading-5 text-foreground">{task.title}</div>
        </div>
        {href ? (
          <Link
            href={href}
            aria-label={`Open ${task.key || task.title}`}
            className="inline-flex shrink-0 items-center gap-0.5 rounded-md px-1.5 py-1 text-[12px] font-medium text-foreground hover:bg-accent"
          >
            Open
            <ArrowUpRight className="size-3.5" aria-hidden />
          </Link>
        ) : null}
      </div>
    </div>
  );
}
