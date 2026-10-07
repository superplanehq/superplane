import { Frame, FrameHeader, FramePanel } from "@/components/reui/frame";
import { cn } from "@/lib/utils";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/ui/collapsible";
import { ChevronRight } from "lucide-react";
import { useEffect, useRef, useState } from "react";

import { type SplitRunPhase, type SplitRunPhaseStatus } from "../splitRunMocks";
import { AutomationCardBody } from "./AutomationCardBody";
import { runMetaLine } from "./consoleCardText";
import { StepOutputCounts } from "./consoleOutputChips";
import type { ConsoleAutomation } from "./automationsViewModel";
import { META_TEXT_CLASSNAME } from "./redesignFormat";
import { StageStatusGlyph } from "./redesignShared";

function isLiveAutomationStatus(status: SplitRunPhaseStatus): boolean {
  return status === "running" || status === "waiting";
}

export function ConsoleAutomationCard({
  automation,
  phase,
  phases,
  organizationId,
  factoryKey,
  orderNumber,
  expandIdle,
  canStopRun,
  actionBusy,
  onStopRun,
  onRerunStep,
  onRerunAutomation,
}: {
  automation: ConsoleAutomation;
  phase?: SplitRunPhase;
  phases: SplitRunPhase[];
  organizationId?: string;
  factoryKey?: string;
  orderNumber?: string;
  /** True for the current-column card on a draft that has not started a run. */
  expandIdle: boolean;
  canStopRun: boolean;
  actionBusy: boolean;
  onStopRun?: (run: { appId: string; runId: string }) => void;
  onRerunStep?: (phase: SplitRunPhase) => void;
  onRerunAutomation?: (phase: SplitRunPhase) => void;
}) {
  const { latest } = automation;
  const stopping = useStopRequested(latest.status, actionBusy);
  const shownStatus: SplitRunPhaseStatus = stopping.active && latest.status === "running" ? "cancelled" : latest.status;
  const shownPhase = phase && shownStatus !== phase.status ? { ...phase, status: shownStatus } : phase;
  const live = isLiveAutomationStatus(shownStatus);
  const [open, setOpen] = useState(live || expandIdle);
  const closedByUser = useRef(false);
  useEffect(() => {
    if (live) {
      setOpen(true);
      return;
    }
    if (closedByUser.current) {
      setOpen(false);
      return;
    }
    if (expandIdle) {
      return;
    }
    setOpen(false);
  }, [expandIdle, live]);
  const handleOpenChange = (next: boolean) => {
    closedByUser.current = !next;
    setOpen(next);
  };
  const stopRun =
    canStopRun && onStopRun && shownStatus === "running" && shownPhase?.appId && shownPhase.runId
      ? () => {
          stopping.request();
          onStopRun({ appId: shownPhase.appId ?? "", runId: shownPhase.runId ?? "" });
        }
      : undefined;
  const retry = consoleAutomationRetry({
    canUpdate: canStopRun,
    status: shownStatus,
    phase: shownPhase,
    onRerunStep,
    onRerunAutomation,
  });
  return (
    <Frame
      variant="default"
      spacing="sm"
      stacked
      dense
      className="[--frame-radius:var(--radius-lg)]"
      data-testid={`redesign-console-automation-${automation.id}`}
    >
      <Collapsible open={open} onOpenChange={handleOpenChange} className="group/collapsible">
        <FrameHeader
          className="relative flex min-w-0 flex-row items-center gap-2 py-2"
          data-testid={`redesign-console-card-header-${automation.id}`}
        >
          <CollapsibleTrigger
            className="absolute inset-0 z-10 cursor-pointer rounded-[inherit]"
            aria-label={`Toggle ${automation.name} details`}
          />
          <StageStatusGlyph status={shownStatus} />
          <span className="shrink-0 text-[13px] font-medium text-foreground">{automation.name}</span>
          <StepOutputCounts stage={latest} phase={shownPhase} runs={automation.runs} />
          <span className={cn(META_TEXT_CLASSNAME, "ml-auto px-1.5 tabular-nums")}>
            {runMetaLine(latest, automation.runs)}
          </span>
          <ChevronRight
            className="size-4 shrink-0 text-muted-foreground transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90"
            aria-hidden
          />
        </FrameHeader>
        <CollapsibleContent>
          <FramePanel className="space-y-3">
            <AutomationCardBody
              automation={automation}
              phase={shownPhase}
              phases={phases.map((entry) => (entry.id === latest.id && shownPhase ? shownPhase : entry))}
              organizationId={organizationId}
              factoryKey={factoryKey}
              orderNumber={orderNumber}
              onStop={stopRun}
              onRetry={retry}
              actionBusy={actionBusy}
            />
          </FramePanel>
        </CollapsibleContent>
      </Collapsible>
    </Frame>
  );
}

function consoleAutomationRetry({
  canUpdate,
  status,
  phase,
  onRerunStep,
  onRerunAutomation,
}: {
  canUpdate: boolean;
  status: SplitRunPhaseStatus;
  phase?: SplitRunPhase;
  onRerunStep?: (phase: SplitRunPhase) => void;
  onRerunAutomation?: (phase: SplitRunPhase) => void;
}): (() => void) | undefined {
  if (!canUpdate || status !== "failed" || !phase) {
    return undefined;
  }
  if (phase.stepIndex != null) {
    return onRerunStep ? () => onRerunStep(phase) : undefined;
  }
  if (!phase.appId || !phase.runId) {
    return undefined;
  }
  return onRerunAutomation ? () => onRerunAutomation(phase) : undefined;
}

function useStopRequested(status: SplitRunPhaseStatus, actionBusy: boolean) {
  const [stopping, setStopping] = useState(false);
  const sawBusy = useRef(false);
  useEffect(() => {
    if (status !== "running") {
      sawBusy.current = false;
      setStopping(false);
      return;
    }
    if (actionBusy) {
      sawBusy.current = true;
      return;
    }
    if (sawBusy.current) {
      sawBusy.current = false;
      setStopping(false);
    }
  }, [actionBusy, status]);
  return { active: stopping, request: () => setStopping(true) };
}
