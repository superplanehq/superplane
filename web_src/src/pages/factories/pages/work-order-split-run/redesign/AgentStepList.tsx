import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/ui/collapsible";
import { SegmentedNav } from "@/ui/SegmentedNav";
import { ChevronRight } from "lucide-react";
import { useState } from "react";

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
  className,
}: {
  stage: AutomationStage;
  steps?: AgentStep[];
  view?: AgentStepView;
  onViewChange?: (view: AgentStepView) => void;
  showToggle?: boolean;
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
            <AgentStepRow key={step.id} step={step} detailed={view === "detailed"} />
          ))}
        </ol>
      )}
      <AgentStepFooter stage={stage} />
    </div>
  );
}

/** In the detailed view every expandable step starts open. */
function AgentStepRow({ step, detailed }: { step: AgentStep; detailed: boolean }) {
  const expandable = detailed && (step.events.length > 0 || Boolean(step.output));
  const [open, setOpen] = useState(expandable);
  // An open step already shows its output, so do not repeat its first line.
  const reason = open && !step.summary ? "" : stepReason(step);
  const row = (
    <div className="flex min-w-0 items-start gap-2.5 px-3 py-2">
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
          {step.status !== "passed" ? <StepStatusBadge status={step.status} /> : null}
          {step.duration ? (
            <span className={cn(META_TEXT_CLASSNAME, "ml-auto shrink-0 tabular-nums")}>{step.duration}</span>
          ) : null}
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
            {step.events.map((event) => (
              <AgentEventBlock key={event.id} event={event} />
            ))}
            {step.output ? <pre className={cn(MONO_LOG_CLASSNAME, "max-h-48 overflow-auto")}>{step.output}</pre> : null}
          </div>
        </CollapsibleContent>
      </Collapsible>
    </li>
  );
}

function StepIcon({ step }: { step: AgentStep }) {
  if (step.type === "node") {
    return <NodeIcon iconSlug={step.iconSlug} className="mt-1" />;
  }
  return <ToolKindIcon type={step.type} className="mt-1" />;
}

function stepReason(step: AgentStep): string {
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

function StepStatusBadge({ status }: { status: AgentStep["status"] }) {
  if (status === "failed" || status === "cancelled") {
    return (
      <Badge variant="outline" className="border-destructive/40 text-destructive">
        {status === "cancelled" ? "Skipped" : "Failed"}
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
  return <Badge variant="outline">Running</Badge>;
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
                <code className="min-w-0 flex-1 truncate font-mono text-[12px] text-foreground/90" title={tool.name}>
                  {tool.name}
                </code>
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
