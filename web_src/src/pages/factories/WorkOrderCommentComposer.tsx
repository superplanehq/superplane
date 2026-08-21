import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { PermissionTooltip } from "@/components/PermissionGate";
import type { SuperplaneUsersUser } from "@/api-client";
import { useOrganizationUsers } from "@/hooks/useOrganizationData";
import { useWorkOrderMentionComposer } from "@/hooks/useWorkOrderMentionComposer";
import { cn } from "@/lib/utils";
import { WorkOrderMentionText } from "@/pages/app/markdownMentions";
import { ArrowUp, Loader2 } from "lucide-react";
import { useRef } from "react";
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
      {/*
        z-[3] keeps the submit button above the mention overlay (z-[2]).
        The overlay renders `@mention` pills with `pointer-events-auto` so
        hover cards work while composing, and the overlay scrolls with the
        textarea. Without a higher z-index here, a mention pill that scrolls
        under this corner intercepts the click meant for the button, making
        it look like "Send" silently does nothing.
      */}
      <div className="absolute right-1.5 bottom-1.5 z-[3] flex items-center gap-1">
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
      <WorkOrderCommentComposerField
        composer={composer}
        canComment={canComment}
        isSubmitting={isSubmitting}
        canSubmit={canSubmit}
        onSubmit={onSubmit}
      />
    </>
  );
}

const COMPOSER_FIELD_TEXT_CLASS = "wrap-anywhere whitespace-pre-wrap px-3 pt-2.5 pb-8 text-[13px] leading-5";

function WorkOrderCommentComposerField({
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
  const overlayRef = useRef<HTMLDivElement>(null);

  return (
    <div className="relative">
      <div
        ref={overlayRef}
        aria-hidden
        data-testid="work-order-comment-mention-overlay"
        className={cn(
          "pointer-events-none absolute inset-0 z-[2] overflow-hidden text-foreground",
          COMPOSER_FIELD_TEXT_CLASS,
        )}
      >
        <WorkOrderMentionText
          text={composer.body}
          people={composer.mentionPeople}
          mentionClassName="work-order-mention-in-composer"
          capturePointer
        />
        {composer.body.endsWith("\n") ? "\n" : null}
      </div>
      <Textarea
        ref={composer.textareaRef}
        id="work-order-comment"
        value={composer.body}
        onChange={(event) => composer.handleChange(event.target.value, event.target.selectionStart)}
        onSelect={(event) => composer.handleChange(event.currentTarget.value, event.currentTarget.selectionStart)}
        onScroll={(event) => {
          const overlay = overlayRef.current;
          if (!overlay) {
            return;
          }
          overlay.scrollTop = event.currentTarget.scrollTop;
          overlay.scrollLeft = event.currentTarget.scrollLeft;
        }}
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
        className={cn(
          COMPOSER_FIELD_TEXT_CLASS,
          "relative z-[1] min-h-[66px] resize-none border-0 bg-transparent text-transparent caret-foreground shadow-none focus-visible:ring-0 dark:text-transparent",
        )}
      />
    </div>
  );
}
