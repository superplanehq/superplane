import { safeExternalUrl } from "@/lib/safeExternalUrl";
import { MarkdownContent } from "@/pages/app/Markdown";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/ui/collapsible";
import { ChevronRight, ExternalLink } from "lucide-react";

const BROADCAST_MARKDOWN =
  "max-w-none text-[13px] leading-5 text-foreground [&_p:first-child]:mt-0 [&_p:last-child]:mb-0";

export function BroadcastActivityBlock({
  title,
  body,
  url,
  defaultOpen = false,
}: {
  title: string;
  body?: string;
  url?: string;
  defaultOpen?: boolean;
}) {
  const href = safeExternalUrl(url);
  const detail = broadcastBody(body, href);
  if (!detail && !href) {
    return <span className="font-medium text-foreground">{title}</span>;
  }

  return (
    <Collapsible defaultOpen={defaultOpen} className="min-w-0" data-testid="task-activity-broadcast">
      <CollapsibleTrigger
        className="group/broadcast flex w-full min-w-0 items-center gap-1 text-left"
        data-testid="task-activity-broadcast-toggle"
      >
        <ChevronRight
          className="size-3.5 shrink-0 text-muted-foreground transition-transform group-data-[state=open]/broadcast:rotate-90"
          aria-hidden
        />
        <span className="min-w-0 truncate font-medium text-foreground">{title}</span>
      </CollapsibleTrigger>
      <CollapsibleContent>
        <div className="mt-1.5 space-y-2 rounded-md border bg-muted/40 px-3 py-2">
          {detail ? <MarkdownContent content={detail} variant="workspace" className={BROADCAST_MARKDOWN} /> : null}
          {href ? (
            <a
              href={href}
              target="_blank"
              rel="noreferrer"
              className="inline-flex max-w-full items-center gap-1.5 text-[13px] text-foreground underline decoration-border underline-offset-2 hover:decoration-foreground"
            >
              <ExternalLink className="size-3.5 shrink-0" aria-hidden />
              <span className="truncate">{href}</span>
            </a>
          ) : null}
        </div>
      </CollapsibleContent>
    </Collapsible>
  );
}

function broadcastBody(body: string | undefined, href: string | null): string | undefined {
  const text = body?.trim();
  if (!text || text === href) {
    return undefined;
  }
  return text;
}
