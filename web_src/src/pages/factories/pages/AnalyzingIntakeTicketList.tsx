import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { Loader2 } from "lucide-react";

import { LINE_INTAKE_COPY, type LineIntakeAnalyzingTicket } from "./lineIntakeModel";

interface AnalyzingIntakeTicketListProps {
  tickets: LineIntakeAnalyzingTicket[];
  openTicketId: string | null;
  onOpenTicket?: (ticket: LineIntakeAnalyzingTicket) => void;
  loading?: boolean;
  error?: boolean;
  onRetry?: () => void;
}

export function AnalyzingIntakeTicketList({
  tickets,
  openTicketId,
  onOpenTicket,
  loading = false,
  error = false,
  onRetry,
}: AnalyzingIntakeTicketListProps) {
  if (loading) {
    return (
      <p className="workspace-body-text px-3 pb-3 pl-9 text-muted-foreground" data-testid="line-intake-loading">
        Loading intake runs.
      </p>
    );
  }

  if (error) {
    return (
      <div className="flex items-center justify-between gap-2 px-3 pb-3 pl-9" data-testid="line-intake-error">
        <p className="workspace-body-text text-destructive">SuperPlane could not load intake runs.</p>
        <Button type="button" size="sm" variant="outline" onClick={onRetry}>
          Try again
        </Button>
      </div>
    );
  }

  if (tickets.length === 0) {
    return (
      <p className="workspace-body-text px-3 pb-3 pl-9 text-muted-foreground" data-testid="line-intake-empty">
        {LINE_INTAKE_COPY.analyzingEmpty}
      </p>
    );
  }

  return (
    <ul
      className="divide-y divide-border/70 border-t border-border/70"
      data-testid="line-intake-analyzing"
      aria-label={LINE_INTAKE_COPY.analyzingTitle}
    >
      {tickets.map((ticket) => {
        const selected = openTicketId === ticket.id;
        return (
          <li key={ticket.id} data-testid={`line-intake-analyzing-ticket-${ticket.id}`}>
            <button
              type="button"
              onClick={() => onOpenTicket?.(ticket)}
              aria-label={`Open ${ticket.title}`}
              aria-pressed={selected}
              className={cn(
                "flex w-full items-start gap-2.5 px-3 py-3.5 text-left transition-colors hover:bg-accent/70",
                selected && "bg-accent",
              )}
            >
              <Loader2
                className="mt-0.5 size-3.5 shrink-0 animate-spin text-muted-foreground"
                aria-hidden
                data-testid="line-intake-analyzing-spinner"
              />
              <span className="min-w-0 flex-1 text-left text-[13px] font-medium leading-snug tracking-[-0.01em] text-foreground">
                {ticket.title}
              </span>
            </button>
          </li>
        );
      })}
    </ul>
  );
}
