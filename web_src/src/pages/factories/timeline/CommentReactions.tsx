import { PermissionTooltip } from "@/components/PermissionGate";
import { cn } from "@/lib/utils";
import { Popover, PopoverContent, PopoverTrigger } from "@/ui/popover";
import { SmilePlus } from "lucide-react";
import { useState } from "react";
import type { WorkOrderTimelineCommentReaction } from "../lib/workOrderTimelineEvents";

// The fixed reaction vocabulary, reusing GitHub's own reaction content
// values (see pkg/models.AllowedCommentReactionEmoji /
// pkg/integrations/github/components/pulls/add_reaction.go) so the value
// stored on the backend, the value GitHub automations send, and what's
// rendered here all agree on one vocabulary. Order here is also the
// picker's display order.
const REACTION_EMOJI: Array<{ value: string; glyph: string; label: string }> = [
  { value: "+1", glyph: "👍", label: "thumbs up" },
  { value: "-1", glyph: "👎", label: "thumbs down" },
  { value: "laugh", glyph: "😄", label: "laugh" },
  { value: "hooray", glyph: "🎉", label: "hooray" },
  { value: "confused", glyph: "😕", label: "confused" },
  { value: "heart", glyph: "❤️", label: "heart" },
  { value: "rocket", glyph: "🚀", label: "rocket" },
  { value: "eyes", glyph: "👀", label: "eyes" },
];

const REACTION_GLYPH_BY_EMOJI = new Map(REACTION_EMOJI.map(({ value, glyph }) => [value, glyph]));

function glyphForEmoji(emoji: string): string {
  return REACTION_GLYPH_BY_EMOJI.get(emoji) ?? emoji;
}

interface CommentReactionsProps {
  reactions: WorkOrderTimelineCommentReaction[];
  /** Same permission tier as commenting — see WorkOrderCommentComposer. */
  canReact: boolean;
  onAddReaction: (emoji: string) => void;
  onRemoveReaction: (emoji: string) => void;
}

export function CommentReactions({ reactions, canReact, onAddReaction, onRemoveReaction }: CommentReactionsProps) {
  const [pickerOpen, setPickerOpen] = useState(false);
  const visibleReactions = reactions.filter((reaction) => reaction.count > 0);

  // Nothing to show and nothing the viewer could do about it — don't
  // render an empty, disabled "add reaction" affordance for read-only
  // viewers.
  if (visibleReactions.length === 0 && !canReact) {
    return null;
  }

  const handleToggle = (emoji: string, reactedByMe: boolean) => {
    if (!canReact) return;
    if (reactedByMe) {
      onRemoveReaction(emoji);
    } else {
      onAddReaction(emoji);
    }
  };

  const handlePickerSelect = (emoji: string) => {
    const existing = reactions.find((reaction) => reaction.emoji === emoji);
    handleToggle(emoji, existing?.reactedByMe ?? false);
    setPickerOpen(false);
  };

  return (
    <div className="mt-2 flex flex-wrap items-center gap-1" data-testid="work-order-comment-reactions">
      {visibleReactions.map((reaction) => (
        <button
          key={reaction.emoji}
          type="button"
          disabled={!canReact}
          onClick={() => handleToggle(reaction.emoji, reaction.reactedByMe)}
          data-testid={`work-order-comment-reaction-${reaction.emoji}`}
          aria-pressed={reaction.reactedByMe}
          className={cn(
            "inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-xs leading-5 transition-colors",
            reaction.reactedByMe
              ? "border-primary/40 bg-primary/10 text-foreground"
              : "border-border bg-background text-muted-foreground hover:bg-accent",
            !canReact && "cursor-not-allowed opacity-60",
          )}
        >
          <span aria-hidden>{glyphForEmoji(reaction.emoji)}</span>
          <span>{reaction.count}</span>
        </button>
      ))}

      <PermissionTooltip allowed={canReact} message="You don't have permission to react to comments.">
        <Popover open={pickerOpen} onOpenChange={setPickerOpen}>
          <PopoverTrigger asChild>
            <button
              type="button"
              disabled={!canReact}
              aria-label="Add reaction"
              data-testid="work-order-comment-add-reaction"
              className={cn(
                "inline-flex size-6 items-center justify-center rounded-full border border-border text-muted-foreground transition-colors hover:bg-accent",
                !canReact && "cursor-not-allowed opacity-60",
              )}
            >
              <SmilePlus className="size-3.5" aria-hidden />
            </button>
          </PopoverTrigger>
          <PopoverContent align="start" className="w-auto p-1.5" sideOffset={6}>
            <div className="flex gap-0.5">
              {REACTION_EMOJI.map(({ value, glyph, label }) => (
                <button
                  key={value}
                  type="button"
                  aria-label={label}
                  data-testid={`work-order-comment-reaction-picker-${value}`}
                  onClick={() => handlePickerSelect(value)}
                  className="flex size-7 items-center justify-center rounded-md text-base hover:bg-accent"
                >
                  {glyph}
                </button>
              ))}
            </div>
          </PopoverContent>
        </Popover>
      </PermissionTooltip>
    </div>
  );
}
