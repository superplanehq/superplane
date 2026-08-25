import { CircleX, ExternalLink, Hourglass } from "lucide-react";

import { Link } from "@/components/Link/link";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { MarkdownContent } from "@/pages/app/Markdown";

import type { SplitRunFooterNote } from "./splitRunFooter";

const TONE = {
  waiting: {
    strip: "border-[color:var(--status-waiting-border)] bg-[color:var(--status-waiting-bg)]",
    iconWrap: "bg-[color:var(--status-waiting-dot)]/15",
    icon: "text-[color:var(--status-waiting-fg)]",
    Icon: Hourglass,
  },
  failed: {
    strip: "border-[color:var(--status-failed-border)] bg-[color:var(--status-failed-bg)]",
    iconWrap: "bg-[color:var(--status-failed-dot)]/15",
    icon: "text-[color:var(--status-failed-fg)]",
    Icon: CircleX,
  },
} as const;

/**
 * Sticky waiting or failed note above Stop. No Update manually, no
 * source time. The CTA sits on the right.
 */
export function SplitRunAttentionNote({
  note,
  tone = "waiting",
  runHref,
}: {
  note: SplitRunFooterNote;
  tone?: "waiting" | "failed";
  runHref?: string | null;
}) {
  const visual = TONE[tone];
  const Icon = visual.Icon;
  const href = note.cta?.href ?? runHref ?? undefined;
  const external = Boolean(href?.startsWith("http"));

  return (
    <div className={cn("border-t px-4 py-3", visual.strip)} data-testid="split-run-attention-note">
      <div className="flex items-start gap-3">
        <span
          className={cn("flex size-8 shrink-0 items-center justify-center rounded-full", visual.iconWrap)}
          aria-hidden
        >
          <Icon className={cn("size-4", visual.icon)} />
        </span>

        <div className="min-w-0 flex-1">
          <h3 className="text-[13px] font-semibold tracking-[-0.01em] text-foreground">{note.headline}</h3>
          {note.text ? (
            <div className="mt-1 text-[13px] text-foreground/80">
              <MarkdownContent content={note.text} variant="workspace" />
            </div>
          ) : null}
        </div>

        {note.cta && href ? (
          <Button asChild size="sm" className="shrink-0">
            {external ? (
              <a href={href} target="_blank" rel="noreferrer">
                {note.cta.label}
                <ExternalLink className="size-3.5" aria-hidden />
              </a>
            ) : (
              <Link href={href}>{note.cta.label}</Link>
            )}
          </Button>
        ) : null}
      </div>
    </div>
  );
}
