import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { Textarea } from "@/components/ui/textarea";
import { PermissionTooltip } from "@/components/PermissionGate";
import { AtSign, Paperclip, Slash, SendHorizontal } from "lucide-react";
import { useState } from "react";

interface WorkOrderCommentComposerProps {
  canComment: boolean;
  isSubmitting: boolean;
  onSubmit: (body: string) => Promise<void>;
}

/**
 * Comment composer under the activity feed. Slash commands and attachments
 * are visual placeholders (disabled) per the redesign plan — the field
 * itself is fully functional plain-text.
 */
export function WorkOrderCommentComposer({ canComment, isSubmitting, onSubmit }: WorkOrderCommentComposerProps) {
  const [body, setBody] = useState("");

  const handleSubmit = async () => {
    const trimmed = body.trim();
    if (!trimmed) {
      return;
    }

    try {
      await onSubmit(trimmed);
      setBody("");
    } catch {
      // Toast surfaced from the action hook.
    }
  };

  return (
    <div
      className="rounded-2xl border border-gray-200 bg-white p-3 shadow-sm focus-within:ring-2 focus-within:ring-violet-100 dark:border-gray-700/70 dark:bg-gray-900/40 dark:focus-within:ring-violet-500/20"
      data-testid="work-order-comment-composer"
    >
      <label htmlFor="work-order-comment" className="sr-only">
        Add a comment
      </label>
      <Textarea
        id="work-order-comment"
        value={body}
        onChange={(event) => setBody(event.target.value)}
        rows={3}
        placeholder="Leave a comment for the team or the assistant..."
        disabled={!canComment || isSubmitting}
        className="min-h-[72px] resize-none border-0 bg-transparent p-0 shadow-none focus-visible:ring-0"
      />
      <div className="mt-2 flex items-center gap-1">
        <PlaceholderIconButton icon={Slash} label="Slash commands (coming soon)" />
        <PlaceholderIconButton icon={AtSign} label="Mention someone (coming soon)" />
        <PlaceholderIconButton icon={Paperclip} label="Attach a file (coming soon)" />

        <div className="ml-auto flex items-center gap-2">
          <Button type="button" variant="ghost" size="sm" onClick={() => setBody("")} disabled={!body || isSubmitting}>
            Clear
          </Button>
          <PermissionTooltip allowed={canComment} message="You don't have permission to comment.">
            <LoadingButton
              type="button"
              size="sm"
              disabled={!canComment || !body.trim() || isSubmitting}
              loading={isSubmitting}
              loadingText="Posting..."
              onClick={handleSubmit}
              data-testid="work-order-comment-submit"
            >
              <span className="mr-1">Comment</span>
              <SendHorizontal className="h-3.5 w-3.5" aria-hidden />
            </LoadingButton>
          </PermissionTooltip>
        </div>
      </div>
    </div>
  );
}

function PlaceholderIconButton({ icon: Icon, label }: { icon: typeof AtSign; label: string }) {
  return (
    <button
      type="button"
      disabled
      aria-label={label}
      title={label}
      className="inline-flex h-7 w-7 items-center justify-center rounded-md text-gray-400 opacity-60 hover:bg-gray-100 dark:text-gray-500 dark:hover:bg-gray-800/60"
    >
      <Icon className="h-4 w-4" aria-hidden />
    </button>
  );
}
