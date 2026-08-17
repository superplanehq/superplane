import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { PermissionTooltip } from "@/components/PermissionGate";
import { cn } from "@/lib/utils";
import { ArrowUp, Loader2 } from "lucide-react";
import { useWorkOrderCommentComposer } from "./useWorkOrderCommentComposer";
import { WorkOrderMentionMenu } from "./WorkOrderMentionMenu";

interface WorkOrderCommentComposerProps {
  organizationId: string;
  canComment: boolean;
  isSubmitting: boolean;
  onSubmit: (body: string, mentionedUserIds: string[]) => Promise<void>;
}

export function WorkOrderCommentComposer({
  organizationId,
  canComment,
  isSubmitting,
  onSubmit,
}: WorkOrderCommentComposerProps) {
  const {
    textareaRef,
    body,
    canSubmit,
    showMenu,
    menuPosition,
    mentionOptions,
    highlightedIndex,
    setHighlightedIndex,
    selectMentionOption,
    handleChange,
    handleKeyDown,
    handleSubmit,
    syncCursor,
  } = useWorkOrderCommentComposer({ organizationId, canComment, isSubmitting, onSubmit });

  return (
    <div
      className="relative min-h-[68px] rounded-xl border border-border bg-background focus-within:border-ring focus-within:ring-1 focus-within:ring-ring/15"
      data-testid="work-order-comment-composer"
    >
      <label htmlFor="work-order-comment" className="sr-only">
        Add a comment
      </label>
      <Textarea
        id="work-order-comment"
        ref={textareaRef}
        value={body}
        role="combobox"
        aria-expanded={showMenu}
        aria-haspopup="listbox"
        aria-controls={showMenu ? "work-order-mention-menu" : undefined}
        onChange={handleChange}
        onClick={(event) => syncCursor(event.currentTarget)}
        onKeyUp={(event) => syncCursor(event.currentTarget)}
        onKeyDown={handleKeyDown}
        rows={2}
        placeholder="Add a comment… (@ to mention someone)"
        disabled={!canComment || isSubmitting}
        className="min-h-[66px] resize-none border-0 bg-transparent px-3 pt-2.5 pb-8 text-[13px] leading-5 shadow-none focus-visible:ring-0"
      />
      {showMenu && menuPosition && (
        <WorkOrderMentionMenu
          options={mentionOptions}
          position={menuPosition}
          highlightedIndex={highlightedIndex}
          onHighlightChange={setHighlightedIndex}
          onSelect={selectMentionOption}
        />
      )}
      <div className="absolute right-1.5 bottom-1.5 flex items-center gap-1">
        <PermissionTooltip allowed={canComment} message="You don't have permission to comment.">
          <Button
            type="button"
            size="icon"
            onClick={() => void handleSubmit()}
            disabled={!canSubmit}
            aria-label="Send comment"
            data-testid="work-order-comment-submit"
            className={cn(
              "size-7 rounded-full transition-colors",
              canSubmit
                ? "bg-foreground text-background hover:bg-foreground/85"
                : "cursor-not-allowed bg-accent text-muted-foreground",
            )}
          >
            {isSubmitting ? (
              <Loader2 className="size-3.5 animate-spin" aria-hidden />
            ) : (
              <ArrowUp className="size-3.5" aria-hidden />
            )}
          </Button>
        </PermissionTooltip>
      </div>
    </div>
  );
}
