import { Badge } from "@/components/ui/badge";
import { Marker, MarkerContent, MarkerIcon } from "@/components/ui/marker";
import { cn } from "@/lib/utils";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/ui/collapsible";
import { SegmentedNav } from "@/ui/SegmentedNav";
import { ChevronRight } from "lucide-react";
import { useEffect, useRef, useState } from "react";

import type { AgentActivity } from "../agentActivity";
import { activityFollowTick, activityFromAgentStep } from "./activityFromAgentStep";
import { StepMarkerDetail } from "./AgentStepMarkerBody";
import { useFollowLogScroll } from "../useFollowLogScroll";
import type { AgentStep, AgentStepEvent, AgentToolRow, AutomationStage } from "./automationsViewModel";
import { RawLogPre } from "./RawLogSheet";
import { META_TEXT_CLASSNAME, MONO_LOG_CLASSNAME } from "./redesignFormat";
import { NodeIcon, StageStatusGlyph, ToolKindIcon } from "./redesignShared";

export type AgentStepView = "summary" | "detailed" | "raw";

const VIEW_OPTIONS = [
  { value: "summary", label: "Summary" },
  { value: "detailed", label: "Detailed" },
  { value: "raw", label: "Raw" },
];

/**
 * Agent transcript with three densities (agent-activity-2 pattern):
 * Summary is one row per step, Detailed opens tool groups, Raw is the
 * plain log. The view toggle can live inside or be controlled by a parent.
 * Pass `steps` to list the whole run (canvas nodes and transcript) instead
 * of the transcript alone.
 */
export function AgentStepList({
  stage,
  steps = stage.agentSteps,
  view: controlledView,
  onViewChange,
  showToggle = true,
  defaultOpen = true,
  className,
}: {
  stage: AutomationStage;
  steps?: AgentStep[];
  view?: AgentStepView;
  onViewChange?: (view: AgentStepView) => void;
  showToggle?: boolean;
  /** Detailed steps start open. The card body passes false. */
  defaultOpen?: boolean;
  className?: string;
}) {
  const [internalView, setInternalView] = useState<AgentStepView>("summary");
  const view = controlledView ?? internalView;
  const setView = (next: string) => {
    const value = next as AgentStepView;
    setInternalView(value);
    onViewChange?.(value);
  };

  if (steps.length === 0) {
    return null;
  }

  return (
    <div className={cn("flex flex-col gap-2", className)} data-testid={`redesign-agent-steps-${stage.id}`}>
      {showToggle ? (
        <div className="flex items-center justify-between gap-3">
          <span className="text-[12px] font-medium text-muted-foreground">Agent transcript</span>
          <SegmentedNav
            ariaLabel="Transcript detail"
            value={view}
            onValueChange={setView}
            options={VIEW_OPTIONS}
            size="xs"
          />
        </div>
      ) : null}
      {view === "raw" ? (
        <RawLogPre log={stage.rawLog} className="max-h-[480px]" />
      ) : (
        <ol key={view} className="divide-y divide-border/70 rounded-md border border-border/80 bg-card">
          {steps.map((step) => (
            <AgentStepRow key={step.id} step={step} detailed={view === "detailed"} defaultOpen={defaultOpen} />
          ))}
        </ol>
      )}
      <AgentStepFooter stage={stage} />
    </div>
  );
}

/** In the detailed view a step can start open or collapsed. */
function AgentStepRow({ step, detailed, defaultOpen }: { step: AgentStep; detailed: boolean; defaultOpen: boolean }) {
  const expandable = detailed && (step.events.length > 0 || Boolean(step.output));
  const [open, setOpen] = useState(expandable && defaultOpen);
  // An open step already shows its output, so do not repeat its first line.
  const reason = open && !step.summary ? "" : stepReason(step);
  const row = (
    <div className="group/step flex min-w-0 items-start gap-2.5 px-3 py-2">
      {expandable ? (
        <ChevronRight
          className={cn("mt-1 size-3.5 shrink-0 text-muted-foreground transition-transform", open && "rotate-90")}
          aria-hidden
        />
      ) : (
        <StepIcon step={step} />
      )}
      <div className="min-w-0 flex-1">
        <div className="flex min-w-0 items-center gap-2">
          <span className="truncate text-[13px] font-medium text-foreground">{step.title}</span>
          <StepStatusBadge status={step.status} />
          <StepDuration duration={step.duration} />
        </div>
        {reason ? <p className="truncate text-[12.5px] text-muted-foreground">{reason}</p> : null}
      </div>
    </div>
  );

  if (!expandable) {
    return <li>{row}</li>;
  }

  return (
    <li>
      <Collapsible open={open} onOpenChange={setOpen}>
        <CollapsibleTrigger asChild>
          <button type="button" className="block w-full text-left hover:bg-muted/40" aria-expanded={open}>
            {row}
          </button>
        </CollapsibleTrigger>
        <CollapsibleContent>
          <div className="flex flex-col gap-2 border-t border-border/60 bg-muted/20 px-3 py-2 pl-9">
            <StepDetail step={step} />
          </div>
        </CollapsibleContent>
      </Collapsible>
    </li>
  );
}

export function AgentStepMarkers({
  stage,
  expandSteps = false,
  liveActivity,
  liveActive = false,
}: {
  stage: AutomationStage;
  expandSteps?: boolean;
  /** Live turn for the open running step. Finished steps keep the collapsible body. */
  liveActivity?: AgentActivity;
  liveActive?: boolean;
}) {
  if (stage.agentSteps.length === 0) {
    return null;
  }
  const liveStepId = liveActive
    ? [...stage.agentSteps].reverse().find((step) => step.status === "running")?.id
    : undefined;
  return (
    <div className="flex flex-col" data-testid={`redesign-agent-steps-${stage.id}`}>
      {stage.agentSteps.map((step) => (
        <AgentStepMarker
          key={step.id}
          step={step}
          defaultOpen={expandSteps}
          liveActivity={step.id === liveStepId ? liveActivity : undefined}
          liveActive={step.id === liveStepId}
          liveTestId={`redesign-live-activity-${stage.id}`}
        />
      ))}
    </div>
  );
}

function AgentStepMarker({
  step,
  defaultOpen = false,
  liveActivity,
  liveActive = false,
  liveTestId,
}: {
  step: AgentStep;
  defaultOpen?: boolean;
  liveActivity?: AgentActivity;
  liveActive?: boolean;
  liveTestId?: string;
}) {
  const running = step.status === "running";
  const expandable = step.events.length > 0 || Boolean(step.output) || liveActive;
  const [open, setOpen] = useState(running || defaultOpen);
  const userToggled = useRef(false);
  useEffect(() => {
    if (!running && !userToggled.current && !defaultOpen) {
      setOpen(false);
    }
  }, [defaultOpen, running]);
  const reason = open && !step.summary ? "" : stepReason(step);
  const activity = liveActivity && liveActivity.items.length > 0 ? liveActivity : activityFromAgentStep(step);
  const follow = useFollowLogScroll<HTMLDivElement>(
    running ? step.id : null,
    liveActive ? activityFollowTick(activity) : step.events.length,
    { resumeOnBottom: true },
  );
  const row = (
    <Marker className={cn("group/step min-w-0 py-1", running && "bg-muted")}>
      <MarkerIcon className="text-muted-foreground">
        {expandable ? (
          <ChevronRight className={cn("size-3.5 transition-transform", open && "rotate-90")} aria-hidden />
        ) : step.type === "node" ? (
          <NodeIcon iconSlug={step.iconSlug} />
        ) : (
          <ToolKindIcon type={step.type} />
        )}
      </MarkerIcon>
      <MarkerContent className="flex min-w-0 flex-1 items-center gap-2 overflow-hidden text-[13px] leading-5 text-foreground">
        <span className="max-w-[50%] shrink-0 truncate">{step.title}</span>
        {step.type === "node" ? <NodeIcon iconSlug={step.iconSlug} /> : <ToolKindIcon type={step.type} />}
        <StepStatusBadge status={step.status} />
        {reason ? (
          <span className="min-w-0 flex-1 truncate font-mono text-xs text-muted-foreground">{reason}</span>
        ) : null}
        <StepDuration duration={step.duration} />
      </MarkerContent>
    </Marker>
  );
  if (!expandable) {
    return row;
  }
  return (
    <Collapsible
      open={open}
      onOpenChange={(next) => {
        userToggled.current = true;
        setOpen(next);
      }}
    >
      <CollapsibleTrigger asChild>
        <button type="button" className="block w-full min-w-0 text-left" aria-expanded={open}>
          {row}
        </button>
      </CollapsibleTrigger>
      <CollapsibleContent className="min-w-0">
        <StepMarkerDetail
          step={step}
          activity={activity}
          running={running}
          liveActive={liveActive}
          liveActivity={liveActivity}
          liveTestId={liveTestId}
          defaultOpen={defaultOpen}
          follow={follow}
        />
      </CollapsibleContent>
    </Collapsible>
  );
}

function StepDetail({ step }: { step: AgentStep }) {
  return (
    <>
      {step.output ? <StepCommand text={step.output} /> : null}
      {step.events.map((event) => (
        <AgentEventBlock key={event.id} event={event} />
      ))}
    </>
  );
}

function StepCommand({ text }: { text: string }) {
  const [open, setOpen] = useState(false);
  const lineRef = useRef<HTMLPreElement>(null);
  const [overflows, setOverflows] = useState(text.includes("\n"));
  useEffect(() => {
    const node = lineRef.current;
    if (!node || overflows) {
      return;
    }
    setOverflows(text.includes("\n") || node.scrollWidth > node.clientWidth + 1);
  }, [overflows, text]);
  if (!overflows) {
    return (
      <pre className={cn(MONO_LOG_CLASSNAME, "truncate")} ref={lineRef}>
        {text}
      </pre>
    );
  }
  return (
    <Collapsible open={open} onOpenChange={setOpen}>
      <CollapsibleTrigger asChild>
        <button
          type="button"
          className="inline-flex max-w-full items-center gap-1 rounded px-1 py-0.5 text-left text-[12px] font-medium text-muted-foreground hover:bg-muted/60 hover:text-foreground"
          aria-expanded={open}
        >
          <ChevronRight className={cn("size-3 shrink-0 transition-transform", open && "rotate-90")} aria-hidden />
          <span ref={lineRef} className="min-w-0 truncate">
            {text.split("\n").find((line) => line.trim()) ?? text}
          </span>
        </button>
      </CollapsibleTrigger>
      <CollapsibleContent>
        <pre className={cn(MONO_LOG_CLASSNAME, "mt-1 whitespace-pre-wrap")}>{text}</pre>
      </CollapsibleContent>
    </Collapsible>
  );
}

function StepDuration({ duration }: { duration?: string }) {
  if (!duration) {
    return null;
  }
  return (
    <span
      className={cn(
        META_TEXT_CLASSNAME,
        "ml-auto shrink-0 whitespace-nowrap text-end opacity-0 transition-opacity duration-150 group-hover/step:opacity-100",
      )}
    >
      <span className="sr-only">Duration </span>
      {duration}
    </span>
  );
}

function StepIcon({ step }: { step: AgentStep }) {
  if (step.type === "node") {
    return <NodeIcon iconSlug={step.iconSlug} className="mt-1" />;
  }
  return <ToolKindIcon type={step.type} className="mt-1" />;
}

function stepReason(step: AgentStep): string {
  if (step.type === "bash") {
    return "";
  }
  if (step.summary) {
    return step.summary;
  }
  const output = step.output
    ?.split("\n")
    .find((line) => line.trim())
    ?.trim();
  if (!output || output === step.title) {
    return "";
  }
  return output;
}

/**
 * Only outcomes that interrupt reading get a badge. A running step is
 * already marked by the card header spinner and its own live rows.
 */
function StepStatusBadge({ status }: { status: AgentStep["status"] }) {
  if (status === "failed" || status === "cancelled") {
    return (
      <Badge variant="outline" className="border-destructive/40 text-destructive">
        {status === "cancelled" ? "Canceled" : "Failed"}
      </Badge>
    );
  }
  if (status === "pending") {
    return (
      <Badge variant="outline" className="text-muted-foreground">
        Pending
      </Badge>
    );
  }
  return null;
}

function AgentEventBlock({ event }: { event: AgentStepEvent }) {
  if (event.kind === "note") {
    return <p className="text-[13px] leading-5 text-foreground/90">{event.text}</p>;
  }
  return <ToolGroup label={event.label} tools={event.tools} />;
}

export function ToolGroup({
  label,
  tools,
  defaultOpen = false,
}: {
  label: string;
  tools: AgentToolRow[];
  defaultOpen?: boolean;
}) {
  const [open, setOpen] = useState(defaultOpen);
  return (
    <Collapsible open={open} onOpenChange={setOpen}>
      <CollapsibleTrigger asChild>
        <button
          type="button"
          className="inline-flex items-center gap-1 rounded px-1 py-0.5 text-[12px] font-medium text-muted-foreground hover:bg-muted/60 hover:text-foreground"
          aria-expanded={open}
        >
          <ChevronRight className={cn("size-3 transition-transform", open && "rotate-90")} aria-hidden />
          {label}
        </button>
      </CollapsibleTrigger>
      <CollapsibleContent>
        <ul className="mt-1 flex flex-col gap-1 border-l border-border/70 pl-3">
          {tools.map((tool) => (
            <li key={tool.id} className="min-w-0">
              <div className="flex min-w-0 items-start gap-2">
                <ToolKindIcon type={tool.type} className="mt-1" />
                <code className="min-w-0 flex-1 break-words font-mono text-[12px] text-foreground/90">{tool.name}</code>
                <StageStatusGlyph status={tool.status} className="mt-0.5 size-3" />
              </div>
              {tool.output ? (
                <pre className={cn(MONO_LOG_CLASSNAME, "mt-1 max-h-32 overflow-auto pl-5.5 text-muted-foreground")}>
                  {tool.output}
                </pre>
              ) : null}
            </li>
          ))}
        </ul>
      </CollapsibleContent>
    </Collapsible>
  );
}

function AgentStepFooter({ stage }: { stage: AutomationStage }) {
  const tools = stage.agentSteps.flatMap((step) =>
    step.events.flatMap((event) => (event.kind === "tools" ? event.tools : [])),
  );
  if (tools.length === 0) {
    return null;
  }
  const count = (types: string[]) => tools.filter((tool) => types.includes(tool.type)).length;
  const parts = [
    plural(count(["edit", "write"]), "file edited", "files edited"),
    plural(count(["bash", "command_execution"]), "command", "commands"),
    plural(count(["read"]), "file read", "files read"),
  ].filter(Boolean);
  return (
    <p className={META_TEXT_CLASSNAME} data-testid={`redesign-agent-footer-${stage.id}`}>
      {parts.join(" · ")}
    </p>
  );
}

function plural(count: number, one: string, many: string): string {
  if (count === 0) return "";
  return `${count} ${count === 1 ? one : many}`;
}
