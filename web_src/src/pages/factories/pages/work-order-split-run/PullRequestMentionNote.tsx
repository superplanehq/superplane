import { cn } from "@/lib/utils";

/**
 * How to ask for changes, shown under whichever note rendered above (the
 * pull-request review strip or the standard note). Kept separate from
 * `note` because the pull-request strip already carries its own fixed
 * copy and does not render `note.text`.
 */
export function PullRequestMentionNote({ text, compact }: { text: string; compact: boolean }) {
  return (
    <p
      className={cn(
        "text-foreground/70",
        compact
          ? "mt-1 text-[12px] leading-4"
          : "border-t border-[color:var(--status-waiting-border)] px-5 py-3 text-[13px] leading-5",
      )}
      data-testid="split-run-pull-request-mention-note"
    >
      {text}
    </p>
  );
}
