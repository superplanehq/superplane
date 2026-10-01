import { useEffect, useRef, useState, type Ref } from "react";

import { agentToolCommandHeadline, agentToolDisplayText, agentToolScriptText } from "@/lib/agentToolLabels";
import { formatDuration } from "@/lib/duration";
import { normalizeTerminalOutput, shellScriptLineCount } from "@/lib/shellScript";
import { cn } from "@/lib/utils";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { ChevronRight, CircleX, SquareTerminal } from "lucide-react";

import { formatWorkOrderDateTime } from "../../lib/workOrderDateTime";
import type { AgentToolItem } from "./agentActivity";

export function CommandLine({ tool, expandable }: { tool: AgentToolItem; expandable: boolean }) {
  const failed = tool.status === "failed" || tool.status === "timed_out";
  const script = agentToolScriptText(tool);
  const headline = agentToolCommandHeadline(tool) || agentToolDisplayText(tool);
  const error = failedCommandError(tool, failed);
  if (!expandable) {
    return (
      <CommandRow
        headline={agentToolDisplayText(tool)}
        failed={failed}
        testId={`agent-tool-${tool.id}`}
        status={tool.status}
        meta={commandMetaLabel(tool)}
        duration={commandDurationLabel(tool)}
      />
    );
  }
  return (
    <ExpandableCommand
      script={script || headline}
      headline={headline}
      failed={failed}
      error={error}
      testId={`agent-tool-${tool.id}`}
      status={tool.status}
      meta={commandMetaLabel(tool)}
      duration={commandDurationLabel(tool)}
    />
  );
}

function ExpandableCommand({
  script,
  headline,
  failed,
  error,
  testId,
  status,
  meta,
  duration,
}: {
  script: string;
  headline: string;
  failed: boolean;
  error?: string;
  testId: string;
  status: string;
  meta?: string;
  duration?: string;
}) {
  const multiLine = script.includes("\n");
  const hidesRest = script.trim() !== headline;
  const lineCount = shellScriptLineCount(script);
  const [open, setOpen] = useState(false);
  const lineRef = useRef<HTMLElement>(null);
  const [overflows, setOverflows] = useState(multiLine || hidesRest);
  useEffect(() => {
    const node = lineRef.current;
    if (!node || overflows) {
      return;
    }
    setOverflows(node.scrollWidth > node.clientWidth + 1);
  }, [overflows, script]);
  const canExpand = overflows || Boolean(error);
  if (!canExpand) {
    return (
      <CommandRow
        headline={headline}
        failed={failed}
        testId={testId}
        status={status}
        meta={meta}
        duration={duration}
        lineRef={lineRef}
      />
    );
  }
  return (
    <div className="px-1 py-0.5">
      <CommandRow
        headline={headline}
        failed={failed}
        status={status}
        meta={meta}
        duration={duration}
        expandable
        open={open}
        lineCount={lineCount}
        onToggle={() => setOpen((current) => !current)}
      />
      {open ? (
        <div className="mt-1 border-l border-border/60 py-1 pl-2">
          {error ? <p className="text-[11px] leading-4 text-destructive">{error}</p> : null}
          <pre
            className={cn(
              "overflow-x-auto font-mono text-[12px] leading-5 whitespace-pre text-foreground/90 [tab-size:2]",
              failed && "text-destructive",
            )}
            data-testid={testId}
            data-status={status}
          >
            {script}
          </pre>
        </div>
      ) : null}
    </div>
  );
}

function CommandRow({
  headline,
  failed,
  testId,
  status,
  meta,
  expandable = false,
  open = false,
  lineCount,
  duration,
  lineRef,
  onToggle,
}: {
  headline: string;
  failed: boolean;
  testId?: string;
  status: string;
  meta?: string;
  expandable?: boolean;
  open?: boolean;
  lineCount?: number;
  duration?: string;
  lineRef?: Ref<HTMLElement>;
  onToggle?: () => void;
}) {
  const row = (
    <div
      className={cn(
        "group flex w-full min-w-0 items-center gap-1.5 rounded px-1 py-0.5 font-mono text-[12px] leading-5 text-foreground/90",
        expandable && "cursor-pointer text-left hover:bg-muted/60",
        failed && "text-destructive",
      )}
    >
      <span className="inline-flex w-3 shrink-0 items-center justify-center" aria-hidden>
        {expandable ? (
          <ChevronRight className={cn("size-3 text-muted-foreground transition-transform", open && "rotate-90")} />
        ) : null}
      </span>
      <code
        ref={lineRef}
        className={cn(
          "block min-w-0 flex-1 truncate font-mono text-[12px] leading-5 whitespace-nowrap",
          failed && "text-destructive",
        )}
        data-testid={testId}
        data-status={status}
      >
        {headline}
      </code>
      {lineCount && lineCount > 1 ? (
        <span className="shrink-0 font-sans text-[11px] text-muted-foreground" aria-hidden>
          {lineCount} lines
        </span>
      ) : null}
      {duration ? (
        <span className="shrink-0 font-sans text-[11px] text-muted-foreground opacity-0 group-hover:opacity-100">
          {duration}
        </span>
      ) : null}
      <SquareTerminal className="size-3 shrink-0 text-muted-foreground opacity-0 group-hover:opacity-100" aria-hidden />
      {failed ? <CircleX className="size-3.5 shrink-0 text-destructive" aria-label="failed" /> : null}
    </div>
  );
  const body = expandable ? (
    <button type="button" className="w-full" aria-expanded={open} aria-label={headline} onClick={onToggle}>
      {row}
    </button>
  ) : (
    row
  );
  if (!meta) {
    return body;
  }
  return (
    <Tooltip>
      <TooltipTrigger asChild>{expandable ? body : <div>{body}</div>}</TooltipTrigger>
      <TooltipContent side="top">{meta}</TooltipContent>
    </Tooltip>
  );
}

function commandDurationLabel(tool: AgentToolItem): string | undefined {
  if (tool.durationMs === undefined) {
    return undefined;
  }
  return formatDuration(tool.durationMs, { precision: "second" }) || undefined;
}

function commandMetaLabel(tool: AgentToolItem): string | undefined {
  const parts: string[] = [];
  if (tool.startedAtMs !== undefined) {
    const stamp = formatWorkOrderDateTime(new Date(tool.startedAtMs));
    if (stamp) {
      parts.push(stamp);
    }
  }
  const duration = commandDurationLabel(tool);
  if (duration) {
    parts.push(duration);
  }
  return parts.length > 0 ? parts.join(" · ") : undefined;
}

function failedCommandError(tool: AgentToolItem, failed: boolean): string | undefined {
  if (!failed) {
    return undefined;
  }
  const output = normalizeTerminalOutput(tool.output);
  if (tool.exitCode !== undefined) {
    return output ? `Exit code ${tool.exitCode}. ${output}` : `Exit code ${tool.exitCode}`;
  }
  return output || undefined;
}
