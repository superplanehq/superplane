import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Popover, PopoverContent, PopoverTrigger } from "@/ui/popover";
import { cn } from "@/lib/utils";
import { Search, SmilePlus } from "lucide-react";
import { useState } from "react";

/** A single participant behind a reaction pill. */
export interface WorkOrderReactionUser {
  id: string;
  name: string;
}

/** All the reactors that picked the same emoji on a work order. */
export interface WorkOrderReactionGroup {
  emoji: string;
  users: WorkOrderReactionUser[];
}

/** GitHub's curated default reaction set — kept small and familiar on purpose. */
export const WORK_ORDER_REACTION_EMOJI_SET = ["👍", "👎", "😄", "🎉", "😕", "❤️", "🚀", "👀"] as const;

interface WorkOrderReactionBarProps {
  reactions: WorkOrderReactionGroup[];
  currentUserId: string;
  /** Whether the viewer is allowed to add/remove a reaction. */
  canReact: boolean;
  /** Message shown in the disabled-state tooltip, matching the comment composer pattern. */
  permissionMessage?: string;
  /**
   * Emoji currently in flight (optimistic toggle awaiting confirmation).
   * The affected pill shows a subdued/pending look while set.
   */
  pendingEmoji?: string | null;
  /** Renders the picker popover pre-opened. Storybook/demo use only. */
  defaultPickerOpen?: boolean;
  onToggleReaction: (emoji: string) => void;
}

export function WorkOrderReactionBar({
  reactions,
  currentUserId,
  canReact,
  permissionMessage = "You don't have permission to react to this work order.",
  pendingEmoji = null,
  defaultPickerOpen = false,
  onToggleReaction,
}: WorkOrderReactionBarProps) {
  const [pickerOpen, setPickerOpen] = useState(defaultPickerOpen);
  const visibleReactions = reactions.filter((group) => group.users.length > 0);

  const handlePick = (emoji: string) => {
    onToggleReaction(emoji);
    setPickerOpen(false);
  };

  return (
    <div className="flex flex-wrap items-center gap-1.5" data-testid="work-order-reaction-bar">
      {visibleReactions.map((group) => (
        <ReactionPill
          key={group.emoji}
          group={group}
          currentUserId={currentUserId}
          canReact={canReact}
          pending={pendingEmoji === group.emoji}
          onToggle={() => onToggleReaction(group.emoji)}
        />
      ))}

      <Popover open={pickerOpen} onOpenChange={(open) => canReact && setPickerOpen(open)} modal={false}>
        <PermissionTooltip allowed={canReact} message={permissionMessage}>
          <PopoverTrigger asChild>
            <Button
              type="button"
              variant="outline"
              size={visibleReactions.length === 0 ? "sm" : "icon-xs"}
              disabled={!canReact}
              aria-label="Add reaction"
              data-testid="work-order-add-reaction-trigger"
              className="font-normal text-muted-foreground"
            >
              <SmilePlus className="size-3.5" aria-hidden />
              {visibleReactions.length === 0 ? "Add reaction" : null}
            </Button>
          </PopoverTrigger>
        </PermissionTooltip>
        <PopoverContent align="start" className="w-64 p-2" sideOffset={8}>
          <EmojiPicker onPick={handlePick} />
        </PopoverContent>
      </Popover>
    </div>
  );
}

function ReactionPill({
  group,
  currentUserId,
  canReact,
  pending,
  onToggle,
}: {
  group: WorkOrderReactionGroup;
  currentUserId: string;
  canReact: boolean;
  pending: boolean;
  onToggle: () => void;
}) {
  const isMine = group.users.some((user) => user.id === currentUserId);

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          type="button"
          disabled={!canReact}
          onClick={onToggle}
          aria-pressed={isMine}
          data-testid={`work-order-reaction-pill-${group.emoji}`}
          className={cn(
            "flex h-7 items-center gap-1 rounded-full border px-2 text-[12px] transition-colors",
            isMine
              ? "border-primary/40 bg-primary/10 text-primary"
              : "border-border bg-muted/40 text-foreground hover:bg-muted",
            pending && "opacity-60",
            !canReact && "cursor-not-allowed opacity-70",
          )}
        >
          <span aria-hidden>{group.emoji}</span>
          <span className="tabular-nums">{group.users.length}</span>
        </button>
      </TooltipTrigger>
      <TooltipContent side="top">{formatReactorNames(group.users, currentUserId)}</TooltipContent>
    </Tooltip>
  );
}

function EmojiPicker({ onPick }: { onPick: (emoji: string) => void }) {
  return (
    <div className="space-y-2">
      <div className="relative">
        <Search className="pointer-events-none absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-muted-foreground" />
        <Input
          type="text"
          placeholder="Search all emoji"
          className="h-8 pl-8 text-[12px]"
          data-testid="work-order-reaction-search"
          // Stubbed for the prototype — the curated set below covers the
          // common reactions; full-picker filtering isn't wired up yet.
          readOnly
        />
      </div>
      <div className="grid grid-cols-8 gap-0.5" role="listbox" aria-label="Emoji reactions">
        {WORK_ORDER_REACTION_EMOJI_SET.map((emoji) => (
          <button
            key={emoji}
            type="button"
            onClick={() => onPick(emoji)}
            aria-label={`React with ${emoji}`}
            data-testid={`work-order-reaction-option-${emoji}`}
            className="flex size-7 items-center justify-center rounded-md text-base hover:bg-muted"
          >
            {emoji}
          </button>
        ))}
      </div>
    </div>
  );
}

/**
 * Formats the reactor list for a pill's tooltip, e.g. "You", "Alex and
 * Priya", or "Alex, Priya, and 2 others". The current user (if present)
 * is shown first as "You".
 */
export function formatReactorNames(users: WorkOrderReactionUser[], currentUserId: string): string {
  const names = [...users]
    .sort((a, b) => (a.id === currentUserId ? -1 : b.id === currentUserId ? 1 : 0))
    .map((user) => (user.id === currentUserId ? "You" : user.name));

  const maxNamed = 2;
  if (names.length <= maxNamed) {
    if (names.length === 1) return names[0];
    return `${names[0]} and ${names[1]}`;
  }

  const shown = names.slice(0, maxNamed);
  const remaining = names.length - maxNamed;
  return `${shown.join(", ")}, and ${remaining} other${remaining === 1 ? "" : "s"}`;
}
