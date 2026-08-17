import { PermissionTooltip } from "@/components/PermissionGate";
import { Popover, PopoverContent, PopoverTrigger } from "@/ui/popover";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { SmilePlus } from "lucide-react";
import { useState } from "react";

import { emojiSlug, WORK_ORDER_REACTION_EMOJIS, type WorkOrderReaction } from "./lib/workOrderReactions";

interface WorkOrderReactionBarProps {
  reactions: WorkOrderReaction[];
  /** Whether the current viewer can add/remove reactions. Mirrors `PermissionTooltip` elsewhere. */
  canReact: boolean;
  /** Emoji currently mid-toggle, shown with a brief pending look (prototype only, no real network call). */
  pendingEmoji?: string | null;
  onToggleReaction: (emoji: string) => void;
}

export function WorkOrderReactionBar({
  reactions,
  canReact,
  pendingEmoji = null,
  onToggleReaction,
}: WorkOrderReactionBarProps) {
  return (
    <div className="flex flex-wrap items-center gap-1.5" data-testid="work-order-reaction-bar">
      {reactions.map((reaction) => (
        <ReactionChip
          key={reaction.emoji}
          reaction={reaction}
          canReact={canReact}
          isPending={pendingEmoji === reaction.emoji}
          onToggle={() => onToggleReaction(reaction.emoji)}
        />
      ))}
      <AddReactionControl canReact={canReact} onPick={onToggleReaction} />
    </div>
  );
}

function ReactionChip({
  reaction,
  canReact,
  isPending,
  onToggle,
}: {
  reaction: WorkOrderReaction;
  canReact: boolean;
  isPending: boolean;
  onToggle: () => void;
}) {
  const { emoji, count, mine, reactorNames } = reaction;

  const chip = (
    <button
      type="button"
      onClick={canReact ? onToggle : undefined}
      disabled={!canReact}
      aria-pressed={mine}
      aria-label={`${emoji} reaction, ${count} ${count === 1 ? "person" : "people"}${mine ? ", including you" : ""}`}
      data-testid={`work-order-reaction-chip-${emojiSlug(emoji)}`}
      data-mine={mine ? "true" : "false"}
      className={cn(
        "inline-flex h-7 items-center gap-1 rounded-full border px-2 text-[12px] font-medium transition-colors",
        mine
          ? "border-primary/40 bg-primary/10 text-foreground"
          : "border-border bg-background text-muted-foreground hover:bg-accent hover:text-foreground",
        !canReact && "cursor-not-allowed hover:bg-background hover:text-muted-foreground",
        isPending && "animate-pulse opacity-60",
      )}
    >
      <span aria-hidden>{emoji}</span>
      <span>{count}</span>
    </button>
  );

  if (!reactorNames || reactorNames.length === 0) {
    return chip;
  }

  return (
    <Tooltip>
      <TooltipTrigger asChild>{chip}</TooltipTrigger>
      <TooltipContent side="top">{reactorNames.join(", ")}</TooltipContent>
    </Tooltip>
  );
}

function AddReactionControl({ canReact, onPick }: { canReact: boolean; onPick: (emoji: string) => void }) {
  const [open, setOpen] = useState(false);

  return (
    <PermissionTooltip allowed={canReact} message="You don't have permission to react to this work order.">
      <Popover
        modal={false}
        open={open}
        onOpenChange={(nextOpen) => {
          if (!canReact) return;
          setOpen(nextOpen);
        }}
      >
        <PopoverTrigger asChild>
          <button
            type="button"
            disabled={!canReact}
            aria-label="Add reaction"
            data-testid="work-order-reaction-add-button"
            className={cn(
              "inline-flex h-7 items-center justify-center gap-1 rounded-full border border-dashed border-border px-2 text-muted-foreground transition-colors",
              canReact ? "hover:bg-accent hover:text-foreground" : "cursor-not-allowed",
            )}
          >
            <SmilePlus className="size-3.5" aria-hidden />
          </button>
        </PopoverTrigger>
        <PopoverContent align="start" className="w-auto p-1.5" data-testid="work-order-reaction-picker">
          <div className="flex items-center gap-1">
            {WORK_ORDER_REACTION_EMOJIS.map((emoji) => (
              <button
                key={emoji}
                type="button"
                data-testid={`work-order-reaction-picker-option-${emojiSlug(emoji)}`}
                aria-label={`React with ${emoji}`}
                onClick={() => {
                  onPick(emoji);
                  setOpen(false);
                }}
                className="flex size-8 items-center justify-center rounded-md text-base transition-colors hover:bg-accent"
              >
                {emoji}
              </button>
            ))}
          </div>
        </PopoverContent>
      </Popover>
    </PermissionTooltip>
  );
}
