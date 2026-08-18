import { ExternalLink } from "lucide-react";

import { Link } from "@/components/Link/link";
import { Badge } from "@/components/ui/badge";
import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog";
import { cn } from "@/lib/utils";
import { MarkdownContent } from "@/pages/app/Markdown";

import {
  CheckAttribution,
  formatCheckScore,
  LEVEL_LABEL,
  type WorkOrderCheckPresentation,
} from "./WorkOrderChecksSection";

/** Expanded view of one check: score, summary, and the full markdown analysis. */
export function WorkOrderCheckDialog({
  open,
  onClose,
  check,
  runHref,
}: {
  open: boolean;
  onClose: () => void;
  check: WorkOrderCheckPresentation;
  runHref?: string | null;
}) {
  const { value, scale } = formatCheckScore(check);
  const level = LEVEL_LABEL[check.level];

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && onClose()}>
      <DialogContent
        size="large"
        className="max-h-[calc(100vh-2rem)] w-[calc(100%-2rem)] max-w-2xl overflow-y-auto text-left"
      >
        <DialogTitle>{check.name}</DialogTitle>

        <div className="mt-3 flex items-center gap-3">
          <span className="flex items-baseline gap-0.5">
            <span className="text-3xl font-semibold tabular-nums tracking-tight text-foreground">{value}</span>
            <span className="text-sm text-muted-foreground">{scale}</span>
          </span>
          <Badge variant="outline" className={cn("border", level.badgeClassName)}>
            {level.label}
          </Badge>
        </div>

        {check.summary ? <p className="mt-3 text-sm text-foreground">{check.summary}</p> : null}

        {check.analysis?.trim() ? (
          <div className="mt-4 border-t border-border pt-4">
            <MarkdownContent content={check.analysis} variant="workspace" />
          </div>
        ) : null}

        <div className="mt-4 flex flex-wrap items-center justify-between gap-2 text-[12px] text-muted-foreground">
          <span>
            <CheckAttribution check={check} />
          </span>
          {runHref ? (
            <Link
              href={runHref}
              className="inline-flex items-center gap-1 font-medium text-foreground/80 hover:text-foreground hover:underline"
            >
              View run
              <ExternalLink className="size-3 shrink-0" aria-hidden />
            </Link>
          ) : null}
        </div>
      </DialogContent>
    </Dialog>
  );
}
