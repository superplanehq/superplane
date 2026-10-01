import { useEffect, useRef, useState, type Ref } from "react";

import { agentToolCommandHeadline, agentToolDisplayText, agentToolScriptText } from "@/lib/agentToolLabels";
import { formatDuration } from "@/lib/duration";
import { normalizeTerminalOutput, shellScriptLineCount } from "@/lib/shellScript";
import { cn } from "@/lib/utils";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { ChevronRight, CircleX, SquareTerminal } from "lucide-react";

import { formatWorkOrderDateTime } from "../../lib/workOrderDateTime";
import type { AgentToolItem } from "./agentActivity";

export function CommandLine({
  tool,
  expandable,
  showOutput = false,
}: {
  tool: AgentToolItem;
  expandable: boolean;
  showOutput?: boolean;
}) {
  const failed = tool.status === "failed" || tool.status === "timed_out";
  const script = agentToolScriptText(tool);
  const headline = agentToolCommandHeadline(tool) || agentToolDisplayText(tool);
  const output = failedCommandOutput(tool, failed);
  const stdout = showOutput && !failed ? normalizeTerminalOutput(tool.output) || undefined : undefined;
  const exitLabel = failedExitLabel(tool, failed);
  if (!expandable) {
    return (
      <div className="min-w-0">
        <CommandRow
          headline={agentToolDisplayText(tool)}
          failed={failed}
          testId={`agent-tool-${tool.id}`}
          status={tool.status}
          meta={commandMetaLabel(tool)}
          duration={commandDurationLabel(tool)}
        />
        {stdout ? <CommandStdout text={stdout} /> : null}
      </div>
    );
  }
  return (
    <ExpandableCommand
      script={script || headline}
      headline={headline}
      failed={failed}
      output={output}
      stdout={stdout}
      exitLabel={exitLabel}
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
  output,
  stdout,
  exitLabel,
  testId,
  status,
  meta,
  duration,
}: {
  script: string;
  headline: string;
  failed: boolean;
  output?: string;
  stdout?: string;
  exitLabel?: string;
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
  const canExpand = overflows || Boolean(output || exitLabel);
  if (!canExpand) {
    const row = (
      <CommandRow
        headline={headline}
        failed={failed}
        exitLabel={exitLabel}
        testId={testId}
        status={status}
        meta={meta}
        duration={duration}
        lineRef={lineRef}
      />
    );
    if (!stdout) {
      return row;
    }
    return (
      <div className="px-1 py-0.5">
        {row}
        <CommandStdout text={stdout} />
      </div>
    );
  }
  return (
    <div className="px-1 py-0.5">
      <CommandRow
        headline={headline}
        failed={failed}
        exitLabel={exitLabel}
        status={status}
        meta={meta}
        duration={duration}
        expandable
        open={open}
        lineCount={lineCount}
        onToggle={() => setOpen((current) => !current)}
      />
      <CommandDetails
        open={open}
        output={output}
        stdout={stdout}
        script={script}
        failed={failed}
        testId={testId}
        status={status}
      />
    </div>
  );
}

function CommandDetails({
  open,
  output,
  stdout,
  script,
  failed,
  testId,
  status,
}: {
  open: boolean;
  output?: string;
  stdout?: string;
  script: string;
  failed: boolean;
  testId: string;
  status: string;
}) {
  if (!open) {
    return stdout ? <CommandStdout text={stdout} /> : null;
  }
  return (
    <div className="mt-1 border-l border-border/60 py-1 pl-2">
      {output ? (
        <pre className="overflow-x-auto font-mono text-[12px] leading-5 whitespace-pre text-destructive [tab-size:2]">
          {output}
        </pre>
      ) : null}
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
      {stdout ? <CommandStdout text={stdout} className="mt-1" /> : null}
    </div>
  );
}

function CommandStdout({ text, className }: { text: string; className?: string }) {
  return (
    <pre
      className={cn(
        "max-h-32 overflow-auto font-mono text-[12px] leading-5 whitespace-pre-wrap break-words text-muted-foreground [tab-size:2]",
        className,
      )}
    >
      {text}
    </pre>
  );
}

type CommandRowProps = {
  headline: string;
  failed: boolean;
  exitLabel?: string;
  testId?: string;
  status: string;
  meta?: string;
  expandable?: boolean;
  open?: boolean;
  lineCount?: number;
  duration?: string;
  lineRef?: Ref<HTMLElement>;
  onToggle?: () => void;
};

function CommandRow({ headline, meta, expandable = false, open = false, onToggle, ...row }: CommandRowProps) {
  const body = expandable ? (
    <button type="button" className="w-full" aria-expanded={open} aria-label={headline} onClick={onToggle}>
      <CommandRowBody headline={headline} expandable open={open} {...row} />
    </button>
  ) : (
    <CommandRowBody headline={headline} {...row} />
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

function CommandRowBody({
  headline,
  failed,
  exitLabel,
  testId,
  status,
  expandable = false,
  open = false,
  lineCount,
  duration,
  lineRef,
}: Omit<CommandRowProps, "meta" | "onToggle">) {
  return (
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
        title={headline}
        className={cn(
          "block min-w-0 flex-1 truncate font-mono text-[12px] leading-5 whitespace-nowrap",
          failed && "text-destructive",
        )}
        data-testid={testId}
        data-status={status}
      >
        {headline}
      </code>
      {exitLabel ? <span className="shrink-0 font-sans text-[11px] text-destructive">{exitLabel}</span> : null}
      {lineCount && lineCount > 1 ? (
        <span className="shrink-0 font-sans text-[11px] text-muted-foreground opacity-0 group-hover:opacity-100">
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

function failedCommandOutput(tool: AgentToolItem, failed: boolean): string | undefined {
  if (!failed) {
    return undefined;
  }
  return normalizeTerminalOutput(tool.output) || undefined;
}

function failedExitLabel(tool: AgentToolItem, failed: boolean): string | undefined {
  if (!failed || tool.exitCode === undefined) {
    return undefined;
  }
  return `Exit code ${tool.exitCode}`;
}
