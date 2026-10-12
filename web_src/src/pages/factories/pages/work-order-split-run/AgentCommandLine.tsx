import { useEffect, useRef, useState, type Ref } from "react";

import { agentToolCommandHeadline, agentToolDisplayText, agentToolScriptText } from "@/lib/agentToolLabels";
import { formatDuration } from "@/lib/duration";
import { normalizeTerminalOutput, shellScriptLineCount } from "@/lib/shellScript";
import { cn } from "@/lib/utils";
import { ChevronRight, CircleX, SquareTerminal } from "lucide-react";

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
  const canExpand = overflows || Boolean(output || stdout || exitLabel);
  if (!canExpand) {
    return (
      <CommandRow
        headline={headline}
        failed={failed}
        exitLabel={exitLabel}
        testId={testId}
        status={status}
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
        exitLabel={exitLabel}
        status={status}
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
    return null;
  }
  return (
    <div className="mt-1 border-l border-border/60 py-1 pl-2">
      {output ? <FailedCommandOutput text={output} /> : null}
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

const COMMAND_OUTPUT_CLASSNAME = "font-mono text-[12px] leading-5 whitespace-pre-wrap break-words [tab-size:2]";

function FailedCommandOutput({ text }: { text: string }) {
  return <pre className={cn(COMMAND_OUTPUT_CLASSNAME, "text-destructive")}>{text}</pre>;
}

function CommandStdout({ text, className }: { text: string; className?: string }) {
  return <pre className={cn(COMMAND_OUTPUT_CLASSNAME, "text-muted-foreground", className)}>{text}</pre>;
}

type CommandRowProps = {
  headline: string;
  failed: boolean;
  exitLabel?: string;
  testId?: string;
  status: string;
  expandable?: boolean;
  open?: boolean;
  lineCount?: number;
  duration?: string;
  lineRef?: Ref<HTMLElement>;
  onToggle?: () => void;
};

function CommandRow({ headline, expandable = false, open = false, onToggle, ...row }: CommandRowProps) {
  if (expandable) {
    return (
      <button type="button" className="w-full" aria-expanded={open} aria-label={headline} onClick={onToggle}>
        <CommandRowBody headline={headline} expandable open={open} {...row} />
      </button>
    );
  }
  return <CommandRowBody headline={headline} {...row} />;
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
}: Omit<CommandRowProps, "onToggle">) {
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
