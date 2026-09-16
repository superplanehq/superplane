import { cn } from "@/lib/utils";
import { Loader2 } from "lucide-react";
import { useEffect, useState, type ReactNode } from "react";

import {
  discussionMentionPreviewComment,
  discussionPreviewBotAuthors,
  discussionPreviewBotInitials,
  discussionPreviewBotIsSkipped,
  discussionSetupPreviewCaption,
} from "./discussionPRFeedbackPreview";
import { PR_FEEDBACK_SETTINGS_COPY } from "./prFeedbackSettingsCopy";
import type { ReviewBotOption } from "./reviewBotPickerModel";
import { PRFeedbackSetupPreviewPane, PRFeedbackSetupPreviewPrLabel } from "./PRFeedbackSetupWizardChrome";
import { DISCUSSION_MENTION, type DiscussionBotMode } from "./useDiscussionPRFeedbackSetup";

const TYPE_INTERVAL_MS = 22;
const BOT_CYCLE_INTERVAL_MS = 2800;

function prefersReducedMotion(): boolean {
  return window.matchMedia?.("(prefers-reduced-motion: reduce)").matches === true;
}

function useTypedPreviewText(text: string): { visible: string; isComplete: boolean } {
  const [visibleCount, setVisibleCount] = useState(() => (prefersReducedMotion() ? text.length : 0));

  useEffect(() => {
    if (prefersReducedMotion()) {
      setVisibleCount(text.length);
      return;
    }

    setVisibleCount(0);
    let count = 0;
    const timer = window.setInterval(() => {
      count += 1;
      setVisibleCount(count);
      if (count >= text.length) {
        window.clearInterval(timer);
      }
    }, TYPE_INTERVAL_MS);

    return () => window.clearInterval(timer);
  }, [text]);

  return {
    visible: text.slice(0, visibleCount),
    isComplete: visibleCount >= text.length,
  };
}

function useCycledIndex(length: number, enabled: boolean): number {
  const [index, setIndex] = useState(0);

  useEffect(() => {
    setIndex(0);
    if (!enabled || length <= 1 || prefersReducedMotion()) {
      return;
    }
    const timer = window.setInterval(() => {
      setIndex((current) => (current + 1) % length);
    }, BOT_CYCLE_INTERVAL_MS);
    return () => window.clearInterval(timer);
  }, [enabled, length]);

  if (length <= 0) {
    return 0;
  }
  return index % length;
}

export function DiscussionPRFeedbackSetupPreview({
  step,
  mentionRequired,
  botMode,
  selectedBots,
  catalog,
}: {
  step: "mention" | "bots";
  mentionRequired: boolean;
  botMode: DiscussionBotMode;
  selectedBots: string[];
  catalog: ReviewBotOption[];
}) {
  const caption = discussionSetupPreviewCaption(step, mentionRequired, botMode, selectedBots.length);

  return (
    <PRFeedbackSetupPreviewPane
      label={PR_FEEDBACK_SETTINGS_COPY.wizardPreviewLabel}
      caption={caption}
      testId="discussion-setup-preview"
      captionTestId="discussion-setup-preview-caption"
    >
      {step === "mention" ? (
        <MentionCommentPreview mentionRequired={mentionRequired} />
      ) : (
        <BotCommentPreview botMode={botMode} selectedBots={selectedBots} catalog={catalog} />
      )}
    </PRFeedbackSetupPreviewPane>
  );
}

function MentionCommentPreview({ mentionRequired }: { mentionRequired: boolean }) {
  const comment = discussionMentionPreviewComment(mentionRequired);
  const typed = useTypedPreviewText(comment);

  return (
    <PreviewCommentCard author={PR_FEEDBACK_SETTINGS_COPY.wizardPreviewCommentAuthor} initials="A" comment={comment}>
      <MentionPreviewBody mentionRequired={mentionRequired} visible={typed.visible} />
      <PreviewCaret visible={!typed.isComplete} />
    </PreviewCommentCard>
  );
}

function MentionPreviewBody({ mentionRequired, visible }: { mentionRequired: boolean; visible: string }) {
  if (!mentionRequired) {
    return visible;
  }
  if (visible.length <= DISCUSSION_MENTION.length) {
    return <PreviewMention>{visible}</PreviewMention>;
  }
  return (
    <>
      <PreviewMention>{DISCUSSION_MENTION}</PreviewMention>
      {visible.slice(DISCUSSION_MENTION.length)}
    </>
  );
}

function PreviewMention({ children }: { children: string }) {
  if (!children) {
    return null;
  }
  return (
    <span className="rounded-[4px] bg-[#5b33ad]/10 px-1 py-0.5 font-medium text-[#4c2a94] dark:bg-[#f6a821]/15 dark:text-[#f6a821]">
      {children}
    </span>
  );
}

function BotCommentPreview({
  botMode,
  selectedBots,
  catalog,
}: {
  botMode: DiscussionBotMode;
  selectedBots: string[];
  catalog: ReviewBotOption[];
}) {
  const authors = discussionPreviewBotAuthors(botMode, selectedBots, catalog);
  const skipped = discussionPreviewBotIsSkipped(botMode, selectedBots);
  const authorIndex = useCycledIndex(authors.length, !skipped);
  const author = authors[authorIndex] ?? authors[0];
  if (!author) {
    return null;
  }

  return (
    <div className="w-full max-w-sm space-y-3">
      <PreviewCommentCard
        key={`${botMode}-${skipped}-${author.login}`}
        author={author.displayName}
        initials={discussionPreviewBotInitials(author.login)}
        comment={PR_FEEDBACK_SETTINGS_COPY.wizardPreviewBotBody}
        muted={skipped}
        badge={PR_FEEDBACK_SETTINGS_COPY.wizardPreviewBotBadge}
      >
        {PR_FEEDBACK_SETTINGS_COPY.wizardPreviewBotBody}
      </PreviewCommentCard>
      <BotPreviewOutcome skipped={skipped} />
    </div>
  );
}

function BotPreviewOutcome({ skipped }: { skipped: boolean }) {
  if (skipped) {
    return (
      <p
        className="animate-in fade-in slide-in-from-bottom-1 text-center text-[12px] font-medium text-muted-foreground duration-300"
        data-testid="discussion-setup-preview-bot-outcome"
      >
        {PR_FEEDBACK_SETTINGS_COPY.wizardPreviewBotIgnored}
      </p>
    );
  }

  return (
    <p
      className="animate-in fade-in slide-in-from-bottom-1 inline-flex w-full items-center justify-center gap-2 text-[12px] font-medium text-[#4c2a94] duration-300 dark:text-[#f6a821]"
      data-testid="discussion-setup-preview-bot-outcome"
      role="status"
    >
      <Loader2 className="size-3.5 motion-safe:animate-spin" aria-hidden />
      {PR_FEEDBACK_SETTINGS_COPY.wizardPreviewBotFixing}
    </p>
  );
}

function PreviewCommentCard({
  author,
  initials,
  comment,
  children,
  muted = false,
  badge,
}: {
  author: string;
  initials: string;
  comment: string;
  children: ReactNode;
  muted?: boolean;
  badge?: string;
}) {
  return (
    <div className="w-full max-w-sm">
      <PRFeedbackSetupPreviewPrLabel />
      <article
        className={cn(
          "rounded-lg border border-border bg-card p-4 shadow-sm transition-opacity duration-300",
          muted ? "opacity-50" : "opacity-100",
        )}
        data-testid="discussion-setup-preview-comment"
        data-comment={comment}
        data-author={author}
      >
        <header className="mb-3 flex items-center gap-2.5">
          <span
            className="inline-flex size-7 shrink-0 items-center justify-center rounded-full bg-[#5b33ad]/10 text-[11px] font-medium text-[#4c2a94] dark:bg-[#f6a821]/15 dark:text-[#f6a821]"
            aria-hidden
          >
            {initials}
          </span>
          <div className="min-w-0">
            <p className="flex items-center gap-1.5 text-[13px] font-medium text-foreground">
              <span>{author}</span>
              {badge ? (
                <span className="rounded border border-border px-1 py-px text-[10px] font-medium uppercase tracking-[0.06em] text-muted-foreground">
                  {badge}
                </span>
              ) : null}
            </p>
            <p className="text-[11px] text-muted-foreground">{PR_FEEDBACK_SETTINGS_COPY.wizardPreviewCommentMeta}</p>
          </div>
        </header>
        <p className="text-[13px] leading-6 text-foreground">{children}</p>
      </article>
    </div>
  );
}

function PreviewCaret({ visible }: { visible: boolean }) {
  if (!visible) {
    return null;
  }
  return (
    <span
      className="ml-0.5 inline-block h-[1em] w-[2px] translate-y-[2px] bg-foreground motion-safe:animate-pulse"
      aria-hidden
    />
  );
}
