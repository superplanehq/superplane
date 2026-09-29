import { Button } from "@/components/ui/button";
import { safeExternalUrl } from "@/lib/safeExternalUrl";
import { cn } from "@/lib/utils";
import { MarkdownContent } from "@/pages/app/Markdown";
import { ChevronDown, ExternalLink } from "lucide-react";
import { useId, useState } from "react";

import type { WorkOrderTimelineBroadcast } from "../lib/workOrderTimelineEvents";

export function BroadcastContent({ broadcast }: { broadcast: WorkOrderTimelineBroadcast }) {
  const contentId = useId();
  const [isExpanded, setIsExpanded] = useState(false);
  const safeUrl = safeExternalUrl(broadcast.url);
  const body = broadcast.body?.trim() ?? "";
  const canExpand = body.length > 0 || Boolean(safeUrl);

  if (!canExpand) {
    return <p className="mt-1 text-[12px] leading-relaxed text-muted-foreground">{broadcast.summary}</p>;
  }

  const linkLabel = broadcast.urlLabel?.trim() || safeUrl;

  return (
    <div className="mt-1" data-testid="task-activity-broadcast">
      <Button
        type="button"
        variant="ghost"
        aria-expanded={isExpanded}
        aria-controls={contentId}
        onClick={() => setIsExpanded((value) => !value)}
        className="h-auto w-full min-w-0 justify-start gap-1 whitespace-normal p-0 text-left text-[12px] font-normal leading-relaxed text-muted-foreground hover:bg-transparent hover:text-foreground"
      >
        <span className="min-w-0 flex-1">{broadcast.summary}</span>
        <ChevronDown
          className={cn("mt-0.5 size-3 shrink-0 transition-transform", isExpanded && "rotate-180")}
          aria-hidden
        />
      </Button>
      {isExpanded ? (
        <div
          id={contentId}
          data-testid="task-activity-broadcast-content"
          className="mt-1.5 rounded-md border px-3 py-2"
        >
          {body ? <MarkdownContent content={body} variant="workspace" /> : null}
          {safeUrl ? (
            <a
              href={safeUrl}
              target="_blank"
              rel="noopener noreferrer"
              className={cn(
                "inline-flex max-w-full items-center gap-1 text-[13px] font-medium text-foreground hover:underline",
                body && "mt-2",
              )}
            >
              <span className="truncate">{linkLabel}</span>
              <ExternalLink className="size-3 shrink-0 text-muted-foreground" aria-hidden />
            </a>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}
