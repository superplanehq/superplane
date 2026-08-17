import { useState } from "react";
import { SmilePlus } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { Popover, PopoverContent, PopoverTrigger } from "@/ui/popover";

/**
 * A short, curated set surfaced as one-tap buttons in the picker — mirrors
 * the reactions engineers reach for most on a work order (ack / done /
 * celebrate / watching / love / shipped).
 */
const CURATED_REACTION_EMOJIS = ["👍", "✅", "🎉", "👀", "❤️", "🚀"] as const;

/**
 * Slightly wider set behind the picker's search box, standing in for a
 * "full" emoji picker in this prototype (no third-party emoji library).
 * `keywords` back the search filter.
 */
const EXTENDED_REACTION_EMOJIS: { emoji: string; keywords: string }[] = [
  { emoji: "👍", keywords: "thumbs up like approve yes" },
  { emoji: "👎", keywords: "thumbs down dislike no" },
  { emoji: "✅", keywords: "check done complete approve" },
  { emoji: "🎉", keywords: "party celebrate tada" },
  { emoji: "👀", keywords: "eyes looking watching" },
  { emoji: "❤️", keywords: "heart love" },
  { emoji: "🚀", keywords: "rocket ship launch fast shipped" },
  { emoji: "🔥", keywords: "fire hot lit" },
  { emoji: "🙌", keywords: "hands raised praise yay" },
  { emoji: "🙏", keywords: "pray thanks please" },
  { emoji: "😄", keywords: "smile happy laugh" },
  { emoji: "😂", keywords: "laugh joy lol" },
  { emoji: "😮", keywords: "wow surprised" },
  { emoji: "😢", keywords: "sad cry" },
  { emoji: "🤔", keywords: "thinking hmm" },
  { emoji: "💯", keywords: "hundred perfect" },
  { emoji: "💪", keywords: "muscle strong" },
  { emoji: "🤝", keywords: "handshake deal agree" },
  { emoji: "🎯", keywords: "target bullseye goal" },
  { emoji: "🐛", keywords: "bug issue defect" },
  { emoji: "🚧", keywords: "construction wip blocked" },
  { emoji: "⭐", keywords: "star favorite" },
  { emoji: "⚡", keywords: "zap fast lightning" },
  { emoji: "💡", keywords: "idea lightbulb" },
];

export interface WorkOrderReactionSummary {
  /** The reaction emoji, e.g. `"👍"`. */
  emoji: string;
  /** Total number of people who reacted with this emoji. */
  count: number;
  /** Whether the current user is one of the reactors. */
  reactedByMe: boolean;
  /**
   * Display names of everyone who reacted, for the hover tooltip. Callers
   * should list `"You"` first when `reactedByMe` is true, e.g.
   * `["You", "Alice", "Bob"]`.
   */
  reactorNames: string[];
}

interface WorkOrderReactionsProps {
  reactions: WorkOrderReactionSummary[];
  /** Whether the current user has permission to react. */
  canReact: boolean;
  /** Matches the `canComment`/`isSubmitting` loading pattern used elsewhere on the page. */
  permissionsLoading?: boolean;
  /** Toggles the current user's reaction on an existing emoji (add or remove). */
  onToggle: (emoji: string) => void;
  /** Adds the current user's reaction using a brand-new emoji picked from the popover. */
  onPickNew: (emoji: string) => void;
  className?: string;
  /** Renders the picker popover pre-opened. Storybook-only affordance for locking the "picker open" state. */
  defaultPickerOpen?: boolean;
}

/**
 * Reaction "pills" strip for the work order detail page. Purely
 * presentational — all state (who reacted, counts, permissions) is owned by
 * the caller. Renders below the work order header, above the description.
 */
export function WorkOrderReactions({
  reactions,
  canReact,
  permissionsLoading = false,
  onToggle,
  onPickNew,
  className,
  defaultPickerOpen = false,
}: WorkOrderReactionsProps) {
  const [pickerOpen, setPickerOpen] = useState(defaultPickerOpen);

  const isReadOnly = !canReact && !permissionsLoading;
  const controlsDisabled = permissionsLoading;

  if (isReadOnly && reactions.length === 0) {
    return null;
  }

  const handlePickerOpenChange = (open: boolean) => {
    if (controlsDisabled) {
      return;
    }
    setPickerOpen(open);
  };

  const handlePick = (emoji: string) => {
    onPickNew(emoji);
    setPickerOpen(false);
  };

  return (
    <div className={cn("flex flex-wrap items-center gap-1.5", className)} data-testid="work-order-reactions">
      {reactions.map((reaction) => (
        <ReactionPill
          key={reaction.emoji}
          reaction={reaction}
          interactive={!isReadOnly}
          disabled={controlsDisabled}
          onToggle={() => onToggle(reaction.emoji)}
        />
      ))}

      {isReadOnly ? null : (
        <Popover open={pickerOpen} onOpenChange={handlePickerOpenChange}>
          <PopoverTrigger asChild>
            <Button
              type="button"
              variant="outline"
              size="icon-xs"
              disabled={controlsDisabled}
              aria-label="Add reaction"
              data-testid="work-order-add-reaction-button"
              className="text-muted-foreground hover:text-foreground"
            >
              <SmilePlus className="size-3.5" aria-hidden />
            </Button>
          </PopoverTrigger>
          <PopoverContent align="start" className="w-64 p-2" sideOffset={8}>
            <EmojiPicker onPick={handlePick} />
          </PopoverContent>
        </Popover>
      )}
    </div>
  );
}

function reactionPillClassName(reaction: WorkOrderReactionSummary, interactive: boolean, disabled: boolean) {
  const canHover = interactive && !disabled;
  return cn(
    "inline-flex h-7 items-center gap-1 rounded-full border px-2.5 text-[13px] font-medium tabular-nums transition-colors",
    reaction.reactedByMe
      ? "border-primary/40 bg-primary/10 text-primary"
      : "border-border bg-background text-foreground",
    canHover && (reaction.reactedByMe ? "hover:bg-primary/15" : "hover:bg-accent"),
    !canHover && "cursor-default",
    disabled && "opacity-60",
  );
}

function reactionPillLabel(reaction: WorkOrderReactionSummary) {
  const peopleLabel = `${reaction.count} ${reaction.count === 1 ? "person" : "people"}`;
  const mineSuffix = reaction.reactedByMe ? ", reacted by you" : "";
  return `${reaction.emoji} reaction, ${peopleLabel}${mineSuffix}`;
}

function ReactionPill({
  reaction,
  interactive,
  disabled,
  onToggle,
}: {
  reaction: WorkOrderReactionSummary;
  interactive: boolean;
  disabled: boolean;
  onToggle: () => void;
}) {
  const label = formatReactorNames(reaction.reactorNames);
  const sharedProps = {
    className: reactionPillClassName(reaction, interactive, disabled),
    "aria-label": reactionPillLabel(reaction),
    "data-testid": `work-order-reaction-pill-${reaction.emoji}`,
  };
  const content = (
    <>
      <span aria-hidden>{reaction.emoji}</span>
      <span>{reaction.count}</span>
    </>
  );

  const pill = interactive ? (
    <button type="button" onClick={onToggle} disabled={disabled} aria-pressed={reaction.reactedByMe} {...sharedProps}>
      {content}
    </button>
  ) : (
    <div {...sharedProps}>{content}</div>
  );

  if (!label) {
    return pill;
  }

  return (
    <Tooltip>
      <TooltipTrigger asChild>{pill}</TooltipTrigger>
      <TooltipContent side="top">{label}</TooltipContent>
    </Tooltip>
  );
}

/** `["You", "Alice", "Bob"]` -> `"You, Alice, Bob"`, truncated for long lists. */
function formatReactorNames(names: string[]): string {
  if (names.length === 0) {
    return "";
  }
  if (names.length <= 4) {
    return names.join(", ");
  }
  const shown = names.slice(0, 3);
  return `${shown.join(", ")} +${names.length - shown.length} more`;
}

function EmojiPicker({ onPick }: { onPick: (emoji: string) => void }) {
  const [query, setQuery] = useState("");
  const trimmedQuery = query.trim().toLowerCase();
  const results = trimmedQuery
    ? EXTENDED_REACTION_EMOJIS.filter((entry) => entry.keywords.includes(trimmedQuery))
    : EXTENDED_REACTION_EMOJIS;

  return (
    <div data-testid="work-order-reaction-picker">
      <div className="flex flex-wrap gap-1 pb-2">
        {CURATED_REACTION_EMOJIS.map((emoji) => (
          <button
            key={emoji}
            type="button"
            onClick={() => onPick(emoji)}
            className="flex size-8 items-center justify-center rounded-md text-lg hover:bg-accent"
            aria-label={`React with ${emoji}`}
            data-testid={`work-order-reaction-picker-curated-${emoji}`}
          >
            {emoji}
          </button>
        ))}
      </div>

      <div className="border-t border-border pt-2">
        <Input
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          placeholder="Search all emoji…"
          className="h-7 text-[13px]"
          aria-label="Search emoji"
          data-testid="work-order-reaction-picker-search"
        />
        <div className="mt-2 grid grid-cols-8 gap-1">
          {results.map((entry) => (
            <button
              key={entry.emoji}
              type="button"
              onClick={() => onPick(entry.emoji)}
              className="flex size-7 items-center justify-center rounded-md text-base hover:bg-accent"
              aria-label={`React with ${entry.emoji}`}
              data-testid={`work-order-reaction-picker-option-${entry.emoji}`}
            >
              {entry.emoji}
            </button>
          ))}
          {results.length === 0 ? (
            <p className="col-span-8 py-2 text-center text-xs text-muted-foreground">No matching emoji</p>
          ) : null}
        </div>
      </div>
    </div>
  );
}
