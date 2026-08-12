import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { PermissionTooltip } from "@/components/PermissionGate";
import { cn } from "@/lib/utils";
import { ArrowUp, Loader2, Paperclip } from "lucide-react";
import { useState } from "react";

interface WorkOrderCommentComposerProps {
  canComment: boolean;
  isSubmitting: boolean;
  onSubmit: (body: string) => Promise<void>;
}

export function WorkOrderCommentComposer({ canComment, isSubmitting, onSubmit }: WorkOrderCommentComposerProps) {
  const [body, setBody] = useState("");
  const canSubmit = canComment && Boolean(body.trim()) && !isSubmitting;

  const handleSubmit = async () => {
    const trimmed = body.trim();
    if (!trimmed) return;

    try {
      await onSubmit(trimmed);
      setBody("");
    } catch {
      // Toast surfaced from the action hook.
    }
  };

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
        value={body}
        onChange={(event) => setBody(event.target.value)}
        onKeyDown={(event) => {
          if ((event.metaKey || event.ctrlKey) && event.key === "Enter" && canSubmit) {
            event.preventDefault();
            void handleSubmit();
          }
        }}
        rows={2}
        placeholder="Add a comment…"
        disabled={!canComment || isSubmitting}
        className="min-h-[66px] resize-none border-0 bg-transparent px-3 pt-2.5 pb-8 text-[13px] leading-5 shadow-none focus-visible:ring-0"
      />
      <div className="absolute right-1.5 bottom-1.5 flex items-center gap-1">
        <PermissionTooltip allowed={canComment} message="Attachments coming soon.">
          <Button
            type="button"
            variant="ghost"
            size="icon"
            disabled
            aria-label="Attach files (coming soon)"
            title="Attach files (coming soon)"
            className="size-7 text-muted-foreground opacity-60"
          >
            <Paperclip className="size-3.5" aria-hidden />
          </Button>
        </PermissionTooltip>
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
