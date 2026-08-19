import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { PermissionTooltip } from "@/components/PermissionGate";
import type { SuperplaneUsersUser } from "@/api-client";
import { useOrganizationUsers } from "@/hooks/useOrganizationData";
import { useWorkOrderMentionComposer } from "@/hooks/useWorkOrderMentionComposer";
import { cn } from "@/lib/utils";
import { ArrowUp, Loader2 } from "lucide-react";
import { WorkOrderMentionMenu } from "./WorkOrderMentionMenu";

interface WorkOrderCommentComposerProps {
  organizationId: string;
  canComment: boolean;
  isSubmitting: boolean;
  onSubmit: (body: string, mentionedUserIds: string[]) => Promise<void>;
  members?: SuperplaneUsersUser[];
}

export function WorkOrderCommentComposer({
  organizationId,
  canComment,
  isSubmitting,
  onSubmit,
  members,
}: WorkOrderCommentComposerProps) {
  if (members) {
    return (
      <WorkOrderCommentComposerView
        canComment={canComment}
        isSubmitting={isSubmitting}
        onSubmit={onSubmit}
        users={members}
      />
    );
  }

  return (
    <WorkOrderCommentComposerLoaded
      organizationId={organizationId}
      canComment={canComment}
      isSubmitting={isSubmitting}
      onSubmit={onSubmit}
    />
  );
}

function WorkOrderCommentComposerLoaded({
  organizationId,
  canComment,
  isSubmitting,
  onSubmit,
}: Omit<WorkOrderCommentComposerProps, "members">) {
  const { data: users = [] } = useOrganizationUsers(organizationId);
  return (
    <WorkOrderCommentComposerView
      canComment={canComment}
      isSubmitting={isSubmitting}
      onSubmit={onSubmit}
      users={users}
    />
  );
}

function WorkOrderCommentComposerView({
  canComment,
  isSubmitting,
  onSubmit,
  users,
}: {
  canComment: boolean;
  isSubmitting: boolean;
  onSubmit: (body: string, mentionedUserIds: string[]) => Promise<void>;
  users: SuperplaneUsersUser[];
}) {
  const composer = useWorkOrderMentionComposer(users);
  const canSubmit = canComment && Boolean(composer.body.trim()) && !isSubmitting;

  const handleSubmit = async () => {
    const trimmed = composer.body.trim();
    if (!trimmed) return;

    try {
      await onSubmit(trimmed, composer.mentionedUserIds);
      composer.reset();
    } catch {
      // Toast surfaced from the action hook.
    }
  };

  return (
    <div
      className="relative min-h-[68px] rounded-xl border border-border bg-background focus-within:border-ring focus-within:ring-1 focus-within:ring-ring/15"
      data-testid="work-order-comment-composer"
    >
      <WorkOrderCommentComposerInput
        composer={composer}
        canComment={canComment}
        isSubmitting={isSubmitting}
        canSubmit={canSubmit}
        onSubmit={() => void handleSubmit()}
      />
      <div className="absolute right-1.5 bottom-1.5 flex items-center gap-1">
        <PermissionTooltip allowed={canComment} message="You do not have permission to comment.">
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

function WorkOrderCommentComposerInput({
  composer,
  canComment,
  isSubmitting,
  canSubmit,
  onSubmit,
}: {
  composer: ReturnType<typeof useWorkOrderMentionComposer>;
  canComment: boolean;
  isSubmitting: boolean;
  canSubmit: boolean;
  onSubmit: () => void;
}) {
  return (
    <>
      <WorkOrderMentionMenu
        suggestions={composer.suggestions}
        highlightIndex={composer.highlightIndex}
        onHighlight={composer.setHighlightIndex}
        onSelect={composer.handleSelectMention}
      />
      <label htmlFor="work-order-comment" className="sr-only">
        Add a comment
      </label>
      <Textarea
        ref={composer.textareaRef}
        id="work-order-comment"
        value={composer.body}
        onChange={(event) => composer.handleChange(event.target.value, event.target.selectionStart)}
        onSelect={(event) => composer.handleChange(event.currentTarget.value, event.currentTarget.selectionStart)}
        onKeyDown={(event) => {
          if (composer.handleMentionKeyDown(event)) {
            return;
          }
          if ((event.metaKey || event.ctrlKey) && event.key === "Enter" && canSubmit) {
            event.preventDefault();
            onSubmit();
          }
        }}
        rows={2}
        placeholder="Add a comment…"
        disabled={!canComment || isSubmitting}
        className="min-h-[66px] resize-none border-0 bg-transparent px-3 pt-2.5 pb-8 text-[13px] leading-5 shadow-none focus-visible:ring-0"
      />
    </>
  );
}
