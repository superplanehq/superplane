import { Frame, FrameHeader, FramePanel } from "@/components/reui/frame";
import { cn } from "@/lib/utils";
import { MarkdownContent } from "@/pages/app/Markdown";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/ui/collapsible";
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from "@/ui/sheet";
import { ChevronRight } from "lucide-react";

import type { SplitRunFixture, SplitRunPhase } from "../splitRunMocks";
import type { AutomationStage, ConsoleAutomation } from "./automationsViewModel";
import { runFooterLine } from "./consoleCardText";
import { LiveAgentSteps } from "./LiveAgentSteps";
import { formatClock, META_TEXT_CLASSNAME } from "./redesignFormat";
import { StageStatusGlyph } from "./redesignShared";

export function ConsoleRunsDrawer({
  automation,
  open,
  onOpenChange,
  fixture,
  organizationId,
}: {
  automation: ConsoleAutomation | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  fixture: SplitRunFixture;
  organizationId?: string;
}) {
  const runs = automation?.runs ?? [];
  const totalCost = runs.reduce((sum, run) => sum + parseUsd(run.cost), 0);
  const summary = [
    `${runs.length} ${runs.length === 1 ? "run" : "runs"}`,
    totalCost > 0 ? `$${totalCost.toFixed(2)} total` : "",
    automation?.latest.model,
  ]
    .filter(Boolean)
    .join(" · ");
  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent
        side="right"
        className="flex w-full flex-col gap-3 sm:max-w-3xl"
        data-testid="redesign-console-run-drawer"
      >
        <SheetHeader className="pr-8">
          <SheetTitle className="text-[15px]">{automation?.name ?? "Runs"}</SheetTitle>
          <SheetDescription>{automation ? summary : "Select an automation to read its runs."}</SheetDescription>
        </SheetHeader>
        <div className="flex min-h-0 flex-1 flex-col gap-2 overflow-auto">
          {runs.map((run, index) => {
            const phase = fixture.phases.find((entry) => entry.id === run.id);
            return (
              <DrawerRun
                key={run.id}
                run={run}
                phase={phase}
                organizationId={organizationId}
                defaultOpen={index === 0}
              />
            );
          })}
        </div>
      </SheetContent>
    </Sheet>
  );
}

function parseUsd(value?: string): number {
  const amount = Number(value?.replace(/[^0-9.]/g, ""));
  return Number.isFinite(amount) ? amount : 0;
}

/**
 * One historical run inside the drawer. The newest run starts open. The
 * row leads with the outcome glyph, then the trigger that caused the run
 * (rendered, so mentions and comment links work), then the revision sha.
 * The body reuses the card's live step list.
 */
function DrawerRun({
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
  const revision = run.pullRequestActivity?.revision;
  const plainName = plainRunTitle(run.name);
  return (
    <Frame variant="default" spacing="sm" stacked dense className="[--frame-radius:var(--radius-lg)]">
      <Collapsible defaultOpen={defaultOpen} className="group/run">
        <FrameHeader className="relative flex min-w-0 flex-row items-center gap-2 py-2">
          <CollapsibleTrigger
            className="absolute inset-0 z-10 cursor-pointer rounded-[inherit]"
            aria-label={`Toggle ${plainName}`}
          />
          <StageStatusGlyph status={run.status} />
          <MarkdownContent
            content={run.name}
            variant="workspace"
            openLinksInNewTab
            linkClassName="font-medium text-current !underline !decoration-current underline-offset-2"
            className="relative z-20 min-w-0 truncate text-left text-[13px] font-medium text-foreground [&_p]:m-0 [&_p]:inline"
          />
          {revision ? (
            <span className="shrink-0 font-mono text-[12px] text-muted-foreground">{revision.sha?.slice(0, 7)}</span>
          ) : null}
          <span className={cn(META_TEXT_CLASSNAME, "ml-auto shrink-0 tabular-nums")}>
            {[formatClock(run.startedAt), run.duration, run.cost].filter(Boolean).join(" · ")}
          </span>
          <ChevronRight
            className="size-4 shrink-0 text-muted-foreground transition-transform duration-200 group-data-[state=open]/run:rotate-90"
            aria-hidden
          />
        </FrameHeader>
        <CollapsibleContent>
          <FramePanel className="space-y-3">
            {run.description ? (
              <div className="text-[12.5px] leading-5 text-muted-foreground">
                <MarkdownContent content={run.description} variant="workspace" />
              </div>
            ) : null}
            <LiveAgentSteps stage={run} phase={phase} organizationId={organizationId} />
            <div className="flex items-center justify-between gap-2 border-t pt-3">
              <span className={META_TEXT_CLASSNAME}>{runFooterLine(run)}</span>
            </div>
          </FramePanel>
        </CollapsibleContent>
      </Collapsible>
    </Frame>
  );
}

/** Markdown links reduced to their text, for aria labels. */
function plainRunTitle(name: string): string {
  return name.replace(/\[([^\]]*)\]\([^)]*\)/g, "$1");
}
