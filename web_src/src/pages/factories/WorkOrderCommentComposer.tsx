import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { PermissionTooltip } from "@/components/PermissionGate";
import { useState } from "react";

interface WorkOrderCommentComposerProps {
  canComment: boolean;
  isSubmitting: boolean;
  onSubmit: (body: string) => Promise<void>;
}

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
    <div className="rounded-lg border border-gray-200 bg-white p-3 dark:border-gray-700/70 dark:bg-gray-900/40">
      <label htmlFor="work-order-comment" className="sr-only">
        Add a comment
      </label>
      <textarea
        id="work-order-comment"
        value={body}
        onChange={(event) => setBody(event.target.value)}
        rows={3}
        placeholder="Leave a comment for the team or the assistant..."
        className="block w-full resize-y rounded-md border border-transparent bg-transparent px-2 py-1.5 text-sm text-gray-900 placeholder:text-gray-400 focus:border-violet-400 focus:outline-none dark:text-gray-100"
        disabled={!canComment || isSubmitting}
      />
      <div className="mt-2 flex justify-end gap-2">
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
            Comment
          </LoadingButton>
        </PermissionTooltip>
      </div>
    </div>
  );
}
