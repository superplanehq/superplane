import { SmilePlus } from "lucide-react";
import { useState } from "react";

import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { Popover, PopoverContent, PopoverTrigger } from "@/ui/popover";

import { REACTION_EMOJI_OPTIONS, type WorkOrderReactionGroup } from "./workOrderReactionTypes";

interface WorkOrderReactionBarProps {
  /** Reaction groups in display order. Empty renders just the "Add reaction" affordance. */
  reactions: WorkOrderReactionGroup[];
  /** Emoji the current viewer reacted with, if any. Highlights the matching pill. */
  myReaction: string | null;
  availableEmojis?: readonly string[];
  /** Add mine, remove mine, or switch mine to a different emoji — the caller owns the state. */
  onToggle: (emoji: string) => void;
  /** Storybook-only: renders the picker open so the "Picker open" story doesn't need a click. */
  defaultPickerOpen?: boolean;
  className?: string;
}

/**
 * Reaction bar for a work order (not a comment). A row of emoji pills, each
 * grouped by emoji with a count and a tooltip listing who reacted, plus an
 * "Add reaction" affordance that opens a small curated picker. Only one
 * active reaction per viewer — picking a new emoji replaces the old one.
 */
export function WorkOrderReactionBar({
  reactions,
  myReaction,
  availableEmojis = REACTION_EMOJI_OPTIONS,
  onToggle,
  defaultPickerOpen = false,
  className,
}: WorkOrderReactionBarProps) {
  const [pickerOpen, setPickerOpen] = useState(defaultPickerOpen);

  return (
    <div className={cn("flex flex-wrap items-center gap-1.5", className)} data-testid="work-order-reaction-bar">
      {reactions.map((group) => (
        <ReactionPill key={group.emoji} group={group} isMine={group.emoji === myReaction} onToggle={onToggle} />
      ))}

      <Popover open={pickerOpen} onOpenChange={setPickerOpen}>
        <PopoverTrigger asChild>
          <button
            type="button"
            aria-label="Add reaction"
            data-testid="work-order-reaction-add"
            className="inline-flex h-6 items-center gap-1 rounded-full border border-dashed border-border px-2 text-[12px] text-muted-foreground transition-colors hover:border-solid hover:bg-accent hover:text-foreground"
          >
            <SmilePlus className="size-3.5" aria-hidden />
            {reactions.length === 0 ? "Add reaction" : null}
          </button>
        </PopoverTrigger>
        <PopoverContent align="start" sideOffset={6} className="w-auto p-1.5" data-testid="work-order-reaction-picker">
          <div className="flex items-center gap-0.5">
            {availableEmojis.map((emoji) => (
              <button
                key={emoji}
                type="button"
                aria-label={`React with ${emoji}`}
                aria-pressed={emoji === myReaction}
                data-testid={`work-order-reaction-option-${emoji}`}
                onClick={() => {
                  onToggle(emoji);
                  setPickerOpen(false);
                }}
                className={cn(
                  "flex size-8 items-center justify-center rounded-md text-base transition-colors hover:bg-accent",
                  emoji === myReaction && "bg-accent",
                )}
              >
                {emoji}
              </button>
            ))}
          </div>
        </PopoverContent>
      </Popover>
    </div>
  );
}

function ReactionPill({
  group,
  isMine,
  onToggle,
}: {
  group: WorkOrderReactionGroup;
  isMine: boolean;
  onToggle: (emoji: string) => void;
}) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          type="button"
          aria-pressed={isMine}
          aria-label={`${group.emoji} reaction from ${formatReactorNames(group.reactorNames)}`}
          data-testid={`work-order-reaction-pill-${group.emoji}`}
          onClick={() => onToggle(group.emoji)}
          className={cn(
            "inline-flex h-6 items-center gap-1 rounded-full border px-2 text-[12px] tabular-nums transition-colors",
            isMine
              ? "border-foreground/25 bg-accent text-foreground"
              : "border-border bg-background text-muted-foreground hover:bg-accent/60 hover:text-foreground",
          )}
        >
          <span aria-hidden>{group.emoji}</span>
          <span>{group.reactorNames.length}</span>
        </button>
      </TooltipTrigger>
      <TooltipContent>{formatReactorNames(group.reactorNames)}</TooltipContent>
    </Tooltip>
  );
}

function formatReactorNames(names: string[]): string {
  if (names.length <= 3) return names.join(", ");
  return `${names.slice(0, 3).join(", ")} and ${names.length - 3} more`;
}
