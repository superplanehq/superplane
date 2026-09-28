import { Frame, FrameHeader, FramePanel } from "@/components/reui/frame";
import { Link } from "@/components/Link/link";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { MarkdownContent } from "@/pages/app/Markdown";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/ui/collapsible";
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from "@/ui/sheet";
import { ChevronRight, Maximize2 } from "lucide-react";

import type { SplitRunFixture, SplitRunPhase } from "../splitRunMocks";
import { splitRunPhaseRunHref } from "../splitRunPopupModel";
import type { AutomationStage, ConsoleAutomation } from "./automationsViewModel";
import { runFooterLine } from "./consoleCardText";
import { LiveAgentSteps } from "./LiveAgentSteps";
import { formatClock, HEADER_ICON_BUTTON, META_TEXT_CLASSNAME } from "./redesignFormat";
import { StageStatusGlyph } from "./redesignShared";

export function ConsoleRunsDrawer({
  automation,
  open,
  onOpenChange,
  fixture,
  organizationId,
  factoryKey,
  orderNumber,
  lineId,
}: {
  automation: ConsoleAutomation | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  fixture: SplitRunFixture;
  organizationId?: string;
  factoryKey?: string;
  orderNumber?: string;
  lineId?: string;
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
            const runHref = phase
              ? splitRunPhaseRunHref({ organizationId, factoryKey, orderNumber, lineId, phase })
              : undefined;
            return (
              <DrawerRun
                key={run.id}
                run={run}
                phase={phase}
                organizationId={organizationId}
                runHref={runHref}
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
  runHref,
  defaultOpen,
}: {
  run: AutomationStage;
  phase?: SplitRunPhase;
  organizationId?: string;
  runHref?: string;
  defaultOpen: boolean;
}) {
  const revision = run.pullRequestActivity?.revision;
  const plainName = plainRunTitle(run.name);
  return (
    <Frame variant="default" spacing="sm" stacked dense className="[--frame-radius:var(--radius-lg)]">
      <Collapsible defaultOpen={defaultOpen} className="group/run">
        <FrameHeader className="flex min-w-0 flex-row items-center gap-2 py-2">
          <CollapsibleTrigger className="shrink-0" aria-label={`Toggle ${plainName}`}>
            <StageStatusGlyph status={run.status} />
          </CollapsibleTrigger>
          <MarkdownContent
            content={run.name}
            variant="workspace"
            openLinksInNewTab
            linkClassName="font-medium text-current !underline !decoration-current underline-offset-2"
            className="min-w-0 truncate text-left text-[13px] font-medium text-foreground [&_p]:m-0 [&_p]:inline"
          />
          {revision ? (
            <span className="shrink-0 font-mono text-[12px] text-muted-foreground">{revision.sha?.slice(0, 7)}</span>
          ) : null}
          <span className={cn(META_TEXT_CLASSNAME, "ml-auto shrink-0 tabular-nums")}>
            {[formatClock(run.startedAt), run.duration, run.cost].filter(Boolean).join(" · ")}
          </span>
          <CollapsibleTrigger className={HEADER_ICON_BUTTON} tabIndex={-1} aria-hidden>
            <ChevronRight
              className="size-4 shrink-0 transition-transform duration-200 group-data-[state=open]/run:rotate-90"
              aria-hidden
            />
          </CollapsibleTrigger>
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
              {runHref ? (
                <Button size="sm" variant="ghost" className="h-7 px-2 text-[12px]" asChild>
                  <Link href={runHref}>
                    <Maximize2 className="size-3.5" aria-hidden />
                    View run
                  </Link>
                </Button>
              ) : null}
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
