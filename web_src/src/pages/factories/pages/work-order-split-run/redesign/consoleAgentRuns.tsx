import { cn } from "@/lib/utils";
import { MarkdownContent } from "@/pages/app/Markdown";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/ui/collapsible";
import { ChevronRight } from "lucide-react";
import { useLayoutEffect, useRef, useState } from "react";

import type { SplitRunPhase } from "../splitRunMocks";
import type { AutomationStage } from "./automationsViewModel";
import { LiveAgentSteps } from "./LiveAgentSteps";
import { formatClock, META_TEXT_CLASSNAME } from "./redesignFormat";
import { StaticStatusGlyph } from "./redesignShared";

const RUN_TITLE_MARKDOWN = "min-w-0 truncate text-left text-[12px] font-medium text-foreground [&_p]:m-0 [&_p]:inline";
const RUN_LINK = "font-medium text-current !underline !decoration-current underline-offset-2";

/**
 * Every run of this automation, oldest first. One flat list: runs are
 * divider rows, not nested boxes — the card border is the only frame.
 * A lone run skips the row chrome; its title stays when it says more
 * than the card title (each Address PR feedback run has its own title).
 */
export function AgentRunsPage({
  runs,
  phases,
  automationName,
  organizationId,
}: {
  runs: AutomationStage[];
  phases: SplitRunPhase[];
  automationName: string;
  organizationId?: string;
}) {
  const single = runs.length === 1 ? runs[0] : undefined;
  if (single) {
    return (
      <SingleRun
        run={single}
        phase={phases.find((phase) => phase.id === single.id)}
        automationName={automationName}
        organizationId={organizationId}
      />
    );
  }
  return (
    <div className="flex flex-col divide-y divide-border">
      {runs.map((run, index) => (
        <AgentRunRow
          key={run.id}
          run={run}
          phase={phases.find((phase) => phase.id === run.id)}
          organizationId={organizationId}
          defaultOpen={index === runs.length - 1}
        />
      ))}
    </div>
  );
}

/** The only run: no row to open, just the title (when it adds one), then the log. */
function SingleRun({
  run,
  phase,
  automationName,
  organizationId,
}: {
  run: AutomationStage;
  phase?: SplitRunPhase;
  automationName: string;
  organizationId?: string;
}) {
  const title = plainRunTitle(run.name);
  return (
    <div className="space-y-2">
      {title && title !== automationName ? (
        <MarkdownContent
          content={run.name}
          variant="workspace"
          openLinksInNewTab
          linkClassName={RUN_LINK}
          className={RUN_TITLE_MARKDOWN}
        />
      ) : null}
      <RunDescription description={run.description} />
      <LiveAgentSteps stage={run} phase={phase} organizationId={organizationId} emptyNote="No steps for this run." />
    </div>
  );
}

function AgentRunRow({
  run,
  phase,
  organizationId,
  defaultOpen,
}: {
  run: AutomationStage;
  phase?: SplitRunPhase;
  organizationId?: string;
  defaultOpen: boolean;
}) {
  const title = plainRunTitle(run.name);
  const clock = formatClock(run.startedAt);
  return (
    <Collapsible defaultOpen={defaultOpen} className="group py-1.5 first:pt-0 last:pb-0">
      <div className="relative flex items-center gap-1.5 py-1">
        <CollapsibleTrigger className="absolute inset-0 z-0 cursor-pointer" aria-label={`Toggle ${title}`} />
        <div className="pointer-events-none relative z-10 flex min-w-0 flex-1 items-center gap-1.5">
          <ChevronRight
            className="size-3.5 shrink-0 text-muted-foreground transition-transform group-data-[state=open]:rotate-90"
            aria-hidden
          />
          {clock ? <span className={cn(META_TEXT_CLASSNAME, "shrink-0 tabular-nums")}>{clock}</span> : null}
          <StaticStatusGlyph status={run.status} />
          <MarkdownContent
            content={run.name}
            variant="workspace"
            openLinksInNewTab
            linkClassName={RUN_LINK}
            className={cn(RUN_TITLE_MARKDOWN, "flex-1 [&_a]:pointer-events-auto")}
          />
        </div>
        {run.duration ? (
          <span className={cn(META_TEXT_CLASSNAME, "pointer-events-none relative z-10 shrink-0 tabular-nums")}>
            {run.duration}
          </span>
        ) : null}
      </div>
      <CollapsibleContent className="space-y-3 py-2 pl-5">
        <RunDescription description={run.description} />
        <LiveAgentSteps stage={run} phase={phase} organizationId={organizationId} emptyNote="No steps for this run." />
      </CollapsibleContent>
    </Collapsible>
  );
}

/** Review comments stay at five lines until the person asks for the rest. */
function RunDescription({ description }: { description?: string }) {
  const [expanded, setExpanded] = useState(false);
  const [overflows, setOverflows] = useState(false);
  const textRef = useRef<HTMLDivElement>(null);

  useLayoutEffect(() => {
    if (expanded) {
      return;
    }
    const el = textRef.current;
    if (!el) {
      return;
    }
    const measure = () => setOverflows(el.scrollHeight - el.clientHeight > 1);
    measure();
    if (typeof ResizeObserver === "undefined") {
      return;
    }
    const observer = new ResizeObserver(measure);
    observer.observe(el);
    return () => observer.disconnect();
  }, [description, expanded]);

  if (!description?.trim()) {
    return null;
  }
  return (
    <div data-testid="redesign-run-description">
      <div
        ref={textRef}
        data-testid="redesign-run-description-body"
        className={cn("text-[12.5px] leading-5 text-muted-foreground", !expanded && "line-clamp-5")}
      >
        <MarkdownContent content={description} variant="workspace" />
      </div>
      {overflows ? (
        <button
          type="button"
          className="mt-1 text-[12px] font-medium text-muted-foreground hover:text-foreground"
          aria-expanded={expanded}
          onClick={(event) => {
            event.stopPropagation();
            setExpanded((open) => !open);
          }}
        >
          {expanded ? "Show less" : "Show more"}
        </button>
      ) : null}
    </div>
  );
}

/** Markdown links reduced to their text, for aria labels. */
function plainRunTitle(name: string): string {
  return name.replace(/\[([^\]]*)\]\([^)]*\)/g, "$1").trim() || "run";
}
