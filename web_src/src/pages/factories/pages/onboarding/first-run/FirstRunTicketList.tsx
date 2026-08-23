import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

import { FIRST_RUN_COPY } from "./firstRunCopy";
import type { FirstRunScoredTicket } from "./firstRunTypes";

function scoreTone(confidencePct: number): string {
  if (confidencePct >= 85) return "text-foreground";
  if (confidencePct >= 75) return "text-foreground/80";
  return "text-muted-foreground";
}

export function FirstRunTicketList({
  tickets,
  interactive = false,
  approvedIds,
  onApprove,
}: {
  tickets: FirstRunScoredTicket[];
  interactive?: boolean;
  approvedIds?: ReadonlySet<string>;
  onApprove?: (ticketId: string) => void;
}) {
  return (
    <ul
      className="divide-y divide-border rounded-lg border border-border text-left"
      data-testid="first-run-ticket-list"
    >
      {tickets.map((ticket) => {
        const approved = approvedIds?.has(ticket.id) ?? false;
        return (
          <li key={ticket.id} className="flex items-center gap-3 px-4 py-3">
            <div className="min-w-0 flex-1">
              <div className="truncate text-[13px] font-medium tracking-[-0.01em]">{ticket.title}</div>
              <p className="mt-0.5 truncate text-[12px] text-muted-foreground">{ticket.source}</p>
            </div>
            <span className={cn("shrink-0 text-[13px] font-medium tabular-nums", scoreTone(ticket.confidencePct))}>
              {ticket.confidencePct}%
            </span>
            {interactive ? (
              <Button
                type="button"
                size="sm"
                variant={approved ? "outline" : "default"}
                disabled={approved}
                onClick={() => onApprove?.(ticket.id)}
              >
                {approved ? FIRST_RUN_COPY.results.approved : FIRST_RUN_COPY.results.approve}
              </Button>
            ) : null}
          </li>
        );
      })}
    </ul>
  );
}
