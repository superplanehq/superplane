import type { FactoriesWorkOrderReaction } from "@/api-client";
import { Button } from "@/components/ui/button";
import { PermissionTooltip } from "@/components/PermissionGate";
import { cn } from "@/lib/utils";
import { Popover, PopoverContent, PopoverTrigger } from "@/ui/popover";
import { SmilePlus } from "lucide-react";
import { useState } from "react";
import { REACTION_CONTENTS, reactionEmoji, reactionLabel } from "./lib/workOrderReactions";

interface WorkOrderReactionBarProps {
  reactions: FactoriesWorkOrderReaction[];
  canReact: boolean;
  isSubmitting?: boolean;
  onToggleReaction: (content: string, reactedByMe: boolean) => void;
  className?: string;
}

/**
 * GitHub-style reaction row for a work order: one pill per emoji that has
 * at least one reaction (showing its count, highlighted when the caller is
 * a reactor), plus a trailing "add reaction" picker for the fixed emoji
 * vocabulary. Clicking a pill toggles the caller's own reaction.
 */
export function WorkOrderReactionBar({
  reactions,
  canReact,
  isSubmitting = false,
  onToggleReaction,
  className,
}: WorkOrderReactionBarProps) {
  const [pickerOpen, setPickerOpen] = useState(false);
  const disabled = !canReact || isSubmitting;

  const activeContents = new Set(reactions.map((reaction) => reaction.content));
  const availableContents = REACTION_CONTENTS.filter((content) => !activeContents.has(content));

  return (
    <div className={cn("flex flex-wrap items-center gap-1.5", className)} data-testid="work-order-reaction-bar">
      {reactions.map((reaction) => {
        if (!reaction.content) {
          return null;
        }

        const reactedByMe = Boolean(reaction.reactedByMe);
        return (
          <PermissionTooltip
            key={reaction.content}
            allowed={canReact}
            message="You don't have permission to react to this work order."
          >
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={disabled}
              onClick={() => onToggleReaction(reaction.content!, reactedByMe)}
              className={cn(
                "h-7 gap-1.5 rounded-full px-2.5 text-[13px] font-medium",
                reactedByMe && "border-primary/50 bg-primary/10 text-primary hover:bg-primary/15",
              )}
              aria-pressed={reactedByMe}
              data-testid={`work-order-reaction-${reaction.content}`}
              title={reactionLabel(reaction.content)}
            >
              <span aria-hidden>{reactionEmoji(reaction.content)}</span>
              <span>{reaction.count ?? 0}</span>
            </Button>
          </PermissionTooltip>
        );
      })}

      {availableContents.length > 0 ? (
        <Popover open={pickerOpen} onOpenChange={setPickerOpen}>
          <PermissionTooltip allowed={canReact} message="You don't have permission to react to this work order.">
            <PopoverTrigger asChild>
              <Button
                type="button"
                variant="outline"
                size="icon-xs"
                disabled={disabled}
                className="size-7 rounded-full text-muted-foreground hover:text-foreground"
                aria-label="Add reaction"
                data-testid="work-order-add-reaction-button"
              >
                <SmilePlus className="size-3.5" aria-hidden />
              </Button>
            </PopoverTrigger>
          </PermissionTooltip>

          <PopoverContent align="start" className="w-auto p-2">
            <div className="grid grid-cols-4 gap-1">
              {availableContents.map((content) => (
                <button
                  key={content}
                  type="button"
                  className="flex size-8 items-center justify-center rounded-md text-lg hover:bg-accent"
                  title={reactionLabel(content)}
                  data-testid={`work-order-add-reaction-option-${content}`}
                  onClick={() => {
                    onToggleReaction(content, false);
                    setPickerOpen(false);
                  }}
                >
                  <span aria-hidden>{reactionEmoji(content)}</span>
                </button>
              ))}
            </div>
          </PopoverContent>
        </Popover>
      ) : null}
    </div>
  );
}
