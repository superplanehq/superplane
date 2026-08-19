import { Avatar } from "@/components/Avatar/avatar";
import { getUserInitials } from "@/lib/orgUserDisplay";
import type { WorkOrderMentionCandidate } from "@/lib/workOrderMentions";
import { cn } from "@/lib/utils";

interface WorkOrderMentionMenuProps {
  suggestions: WorkOrderMentionCandidate[];
  highlightIndex: number;
  onHighlight: (index: number) => void;
  onSelect: (candidate: WorkOrderMentionCandidate) => void;
}

export function WorkOrderMentionMenu({
  suggestions,
  highlightIndex,
  onHighlight,
  onSelect,
}: WorkOrderMentionMenuProps) {
  if (suggestions.length === 0) {
    return null;
  }

  return (
    <ul
      role="listbox"
      aria-label="Mention a member"
      data-testid="work-order-mention-menu"
      className="absolute inset-x-0 bottom-full z-20 mb-1 max-h-56 overflow-y-auto rounded-lg border border-border bg-background py-1 shadow-md"
    >
      {suggestions.map((candidate, index) => {
        const highlighted = index === highlightIndex;
        return (
          <li key={candidate.id} role="option" aria-selected={highlighted}>
            <button
              type="button"
              data-testid={`work-order-mention-option-${candidate.id}`}
              className={cn(
                "flex w-full items-center gap-2 px-2 py-1.5 text-left",
                highlighted ? "bg-accent" : "hover:bg-accent/60",
              )}
              onMouseEnter={() => onHighlight(index)}
              onMouseDown={(event) => {
                event.preventDefault();
                onSelect(candidate);
              }}
            >
              <Avatar initials={getUserInitials(candidate.name)} alt={candidate.name} className="size-6" />
              <span className="min-w-0">
                <span className="block truncate text-[13px] font-medium text-foreground">{candidate.name}</span>
                {candidate.email ? (
                  <span className="block truncate text-[11px] text-muted-foreground">{candidate.email}</span>
                ) : null}
              </span>
            </button>
          </li>
        );
      })}
    </ul>
  );
}
