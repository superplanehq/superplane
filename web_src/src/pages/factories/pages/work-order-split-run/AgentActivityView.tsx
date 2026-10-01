import { useEffect, useRef, useState } from "react";

import {
  agentToolCommandHeadline,
  agentToolDisplayText,
  agentToolScriptText,
  isCommandTool,
} from "@/lib/agentToolLabels";
import { normalizeTerminalOutput, shellScriptLineCount } from "@/lib/shellScript";
import { cn } from "@/lib/utils";
import { MarkdownContent } from "@/pages/app/Markdown";
import { ChevronRight } from "lucide-react";

import type { AgentActivity, AgentActivityItem, AgentContentItem, AgentToolItem } from "./agentActivity";
import { AnimatedThinkingState } from "./AnimatedThinkingState";
import {
  activitySummaryLabel,
  completedActivitySummaryLabel,
  groupToolRuns,
  type ToolActivityGroup,
} from "./agentActivitySummary";
import { isThinkingPlaceholder } from "./streamNotesFromLiveLog";

const COMMAND_CLASSNAME =
  "whitespace-pre-wrap break-words font-mono text-[12px] leading-5 text-foreground/90 [tab-size:2]";

export function AgentActivityView({
  activity,
  live = false,
  collapseCompleted = true,
  collapseReasoning = true,
  expandableCommands = false,
  tone = "chat",
}: {
  activity: AgentActivity;
  live?: boolean;
  /** Refinement chat folds a finished turn. The run console keeps the full transcript. */
  collapseCompleted?: boolean;
  /** Analysis folds a finished thought. The run console keeps the thought text open. */
  collapseReasoning?: boolean;
  /** Finished console command rows. The run console uses this for live and settled steps. */
  expandableCommands?: boolean;
  /** `log` matches finished console type color. `chat` is the refinement thread. */
  tone?: "chat" | "log";
}) {
  const entries = visibleActivityItems(activity.items, live, !collapseCompleted);
  if (entries.length === 0 && !activity.truncated) return null;

  const tools = activity.items.filter((item): item is AgentToolItem => item.type === "tool");
  if (!live && collapseCompleted && tools.length > 0) {
    return (
      <CompletedActivity
        activity={activity}
        entries={entries}
        tools={tools}
        tone={tone}
        collapseReasoning={collapseReasoning}
        expandableCommands={expandableCommands}
      />
    );
  }

  return (
    <div className="space-y-0 px-2" data-testid={`agent-activity-${activity.id}`}>
      <ActivityEntries
        entries={entries}
        truncated={activity.truncated}
        tone={tone}
        collapseReasoning={collapseReasoning}
        expandableCommands={expandableCommands}
      />
    </div>
  );
}

function CompletedActivity({
  activity,
  entries,
  tools,
  tone,
  collapseReasoning,
  expandableCommands,
}: {
  activity: AgentActivity;
  entries: AgentActivityItem[];
  tools: AgentToolItem[];
  tone: "chat" | "log";
  collapseReasoning: boolean;
  expandableCommands: boolean;
}) {
  const [open, setOpen] = useState(false);
  const label = completedActivitySummaryLabel(tools);

  return (
    <div className="space-y-0 px-2" data-testid={`agent-activity-${activity.id}`}>
      <ActivitySummaryButton
        label={label}
        open={open}
        onToggle={() => setOpen((current) => !current)}
        testId={`agent-activity-summary-${activity.id}`}
      />
      {open ? (
        <div className="space-y-0.5" data-testid={`agent-activity-details-${activity.id}`}>
          <ActivityEntries
            entries={entries}
            truncated={activity.truncated}
            tone={tone}
            collapseReasoning={collapseReasoning}
            expandableCommands={expandableCommands}
          />
        </div>
      ) : null}
    </div>
  );
}

function ActivityEntries({
  entries,
  truncated,
  groupTools = true,
  tone = "chat",
  collapseReasoning = true,
  expandableCommands = false,
}: {
  entries: AgentActivityItem[];
  truncated: boolean;
  groupTools?: boolean;
  tone?: "chat" | "log";
  collapseReasoning?: boolean;
  expandableCommands?: boolean;
}) {
  const displayEntries = groupTools ? groupToolRuns(entries) : entries;
  return (
    <>
      {displayEntries.map((item) => {
        if (item.type === "tool_activity_group") {
          return <ToolGroup key={item.id} group={item} tone={tone} expandableCommands={expandableCommands} />;
        }
        if (item.type === "tool") {
          return <ToolLine key={item.id} tool={item} tone={tone} expandableCommands={expandableCommands} />;
        }
        if (item.type === "content") {
          if (item.kind === "assistant") {
            return <AssistantContent key={item.id} item={item} tone={tone} />;
          }
          return <ReasoningContent key={item.id} item={item} collapseReasoning={collapseReasoning} />;
        }
        return (
          <p
            key={item.id}
            className={cn(
              "px-2 py-1 text-[13px] leading-5",
              tone === "log" ? "text-foreground" : "text-muted-foreground",
            )}
          >
            {item.text}
          </p>
        );
      })}
      {truncated ? (
        <p className="px-2 text-[11px] leading-4 text-muted-foreground">Some activity details were omitted.</p>
      ) : null}
    </>
  );
}

function ToolGroup({
  group,
  tone,
  expandableCommands,
}: {
  group: ToolActivityGroup;
  tone: "chat" | "log";
  expandableCommands: boolean;
}) {
  const [open, setOpen] = useState(false);
  const label = activitySummaryLabel(group.tools);
  const running = group.tools.some((tool) => tool.status === "running");

  return (
    <div className="sp-tool-enter px-1" data-testid={group.id}>
      <ActivitySummaryButton
        label={label}
        open={open}
        running={running}
        onToggle={() => setOpen((current) => !current)}
      />
      {open ? (
        <div className="space-y-0.5" data-testid={`${group.id}-details`}>
          {group.tools.map((tool) => (
            <ToolLine key={tool.id} tool={tool} tone={tone} expandableCommands={expandableCommands} />
          ))}
        </div>
      ) : null}
    </div>
  );
}

function ActivitySummaryButton({
  label,
  open,
  running = false,
  onToggle,
  testId,
}: {
  label: string;
  open: boolean;
  running?: boolean;
  onToggle: () => void;
  testId?: string;
}) {
  return (
    <button
      type="button"
      aria-expanded={open}
      aria-label={label}
      onClick={onToggle}
      data-testid={testId}
      className="sp-tool-enter flex w-full items-center gap-1 rounded-md px-1 py-0.5 text-left text-[13px] leading-5 text-muted-foreground hover:text-foreground"
    >
      <span className="min-w-0 whitespace-normal break-words">
        {running ? <AnimatedThinkingState text={label} /> : label}
      </span>
      <ChevronRight className={cn("size-3.5 shrink-0 transition-transform", open && "rotate-90")} aria-hidden />
    </button>
  );
}

function ToolLine({
  tool,
  tone,
  expandableCommands,
}: {
  tool: AgentToolItem;
  tone: "chat" | "log";
  expandableCommands: boolean;
}) {
  const output = isCommandTool(tool) ? normalizeTerminalOutput(tool.output) : tool.output.trim();
  const exitCode = failedTool(tool) ? tool.exitCode : undefined;
  if (isCommandTool(tool)) {
    return (
      <CommandLine
        tool={tool}
        expandable={expandableCommands}
        output={tone === "log" ? output : ""}
        exitCode={tone === "log" ? exitCode : undefined}
      />
    );
  }
  const line = <ToolLabel tool={tool} tone={tone} />;
  if (tone !== "log" || (!output && exitCode === undefined)) {
    return line;
  }
  return (
    <div>
      {line}
      {exitCode !== undefined ? (
        <p className="px-1 text-[11px] leading-4 text-destructive">Exit code {exitCode}</p>
      ) : null}
      {output ? (
        <pre className={cn(COMMAND_CLASSNAME, "mt-1 max-h-32 overflow-auto px-1 text-muted-foreground")}>{output}</pre>
      ) : null}
    </div>
  );
}

function ToolLabel({ tool, tone }: { tool: AgentToolItem; tone: "chat" | "log" }) {
  const label = agentToolDisplayText(tool);
  return (
    <div
      className={cn(
        "sp-tool-enter flex min-w-0 items-baseline gap-2 px-1 py-0.5 text-[12px] leading-5",
        tone === "log" ? "text-foreground/90" : "text-muted-foreground",
        failedTool(tool) && "text-destructive",
      )}
      data-testid={`agent-tool-${tool.id}`}
      data-status={tool.status}
    >
      <span className="min-w-0 flex-1 truncate" title={label}>
        {label}
      </span>
    </div>
  );
}

function CommandLine({
  tool,
  expandable,
  output = "",
  exitCode,
}: {
  tool: AgentToolItem;
  expandable: boolean;
  output?: string;
  exitCode?: number;
}) {
  const failed = failedTool(tool);
  if (!expandable) {
    const command = agentToolDisplayText(tool);
    return (
      <div className="min-w-0 px-1 py-0.5">
        <code
          className={cn(
            "block min-w-0 flex-1 truncate font-mono text-[12px] leading-5 whitespace-nowrap text-muted-foreground",
            failed && "text-destructive",
          )}
          data-testid={`agent-tool-${tool.id}`}
          data-status={tool.status}
          title={command}
        >
          {command}
        </code>
        <CommandTrail exitCode={exitCode} output={output} />
      </div>
    );
  }
  const script = agentToolScriptText(tool);
  const headline = agentToolCommandHeadline(tool);
  return (
    <ExpandableCommand
      script={script || headline}
      headline={headline}
      failed={failed}
      output={output}
      exitCode={exitCode}
      testId={`agent-tool-${tool.id}`}
      status={tool.status}
    />
  );
}

function ExpandableCommand({
  script,
  headline,
  failed,
  output,
  exitCode,
  testId,
  status,
}: {
  script: string;
  headline: string;
  failed: boolean;
  output: string;
  exitCode?: number;
  testId: string;
  status: string;
}) {
  const multiLine = script.includes("\n");
  const hidesRest = script.trim() !== headline;
  const lineCount = shellScriptLineCount(script);
  const [open, setOpen] = useState(false);
  const lineRef = useRef<HTMLPreElement>(null);
  const [overflows, setOverflows] = useState(multiLine || hidesRest);
  useEffect(() => {
    const node = lineRef.current;
    if (!node || overflows) {
      return;
    }
    setOverflows(node.scrollWidth > node.clientWidth + 1);
  }, [overflows, script]);
  const trail = <CommandTrail exitCode={exitCode} output={output} />;
  if (!overflows) {
    return (
      <div className="px-1 py-0.5">
        <pre
          className={cn(COMMAND_CLASSNAME, "truncate", failed && "text-destructive")}
          ref={lineRef}
          data-testid={testId}
          data-status={status}
        >
          {headline}
        </pre>
        {trail}
      </div>
    );
  }
  return (
    <div className="px-1 py-0.5">
      <button
        type="button"
        className={cn(
          "flex w-full min-w-0 items-center gap-1.5 rounded px-1 py-0.5 text-left font-mono text-[12px] leading-5 text-foreground/90 hover:bg-muted/60",
          failed && "text-destructive",
        )}
        aria-expanded={open}
        aria-label={headline}
        onClick={() => setOpen((current) => !current)}
      >
        <ChevronRight
          className={cn("size-3 shrink-0 text-muted-foreground transition-transform", open && "rotate-90")}
          aria-hidden
        />
        <span className="w-3 shrink-0 text-center text-muted-foreground" aria-hidden>
          $
        </span>
        <span className="min-w-0 flex-1 truncate">{headline}</span>
        {lineCount > 1 ? (
          <span className="shrink-0 font-sans text-[11px] text-muted-foreground" aria-hidden>
            {lineCount} lines
          </span>
        ) : null}
      </button>
      {open ? (
        <div className="mt-1 overflow-x-auto border-l border-border/60 py-1 pl-2">
          <pre
            className={cn(
              "font-mono text-[12px] leading-5 whitespace-pre text-foreground/90 [tab-size:2]",
              failed && "text-destructive",
            )}
            data-testid={testId}
            data-status={status}
          >
            {script}
          </pre>
          {trail}
        </div>
      ) : (
        trail
      )}
    </div>
  );
}

function CommandTrail({ exitCode, output }: { exitCode?: number; output: string }) {
  if (exitCode === undefined && !output) {
    return null;
  }
  return (
    <>
      {exitCode !== undefined ? <p className="text-[11px] leading-4 text-destructive">Exit code {exitCode}</p> : null}
      {output ? (
        <>
          <p className="mt-1 text-[11px] leading-4 text-muted-foreground">Output</p>
          <pre className="mt-0.5 max-h-32 overflow-auto font-mono text-[12px] leading-5 whitespace-pre-wrap break-words text-muted-foreground [tab-size:2]">
            {output}
          </pre>
        </>
      ) : null}
    </>
  );
}

function visibleActivityItems(items: AgentActivityItem[], live: boolean, keepAssistant: boolean): AgentActivityItem[] {
  return items.filter((item, index) => isVisibleActivityItem(item, index, items, live, keepAssistant));
}

function isVisibleActivityItem(
  item: AgentActivityItem,
  index: number,
  items: AgentActivityItem[],
  live: boolean,
  keepAssistant: boolean,
): boolean {
  if (item.type === "content" && item.kind === "assistant") {
    if (!item.text.trim() || isThinkingPlaceholder(item.text)) return false;
    return live || keepAssistant || items.slice(index + 1).some((candidate) => candidate.type === "tool");
  }
  return item.type !== "content" || item.kind !== "reasoning" || item.status === "running" || Boolean(item.text.trim());
}

function ReasoningContent({ item, collapseReasoning = true }: { item: AgentContentItem; collapseReasoning?: boolean }) {
  const [manualOpen, setManualOpen] = useState<boolean | null>(null);
  const running = item.status === "running";
  const label = reasoningLabel(item);

  if (running && !item.text.trim() && !collapseReasoning) {
    return null;
  }

  if (running) {
    return (
      <div className="sp-tool-enter px-2 py-1">
        <span
          role="status"
          aria-label={label}
          className="sp-ai-thinking inline-block text-[13px] leading-5 text-muted-foreground"
          data-text={label}
        >
          {label}
        </span>
        {item.text ? (
          <div className="sp-reasoning-stream mt-1 border-l border-border/70 py-1 pl-3 whitespace-pre-wrap [overflow-wrap:anywhere]">
            <p className="sp-reasoning-stream-line">{item.text}</p>
            {item.truncated ? <p className="text-[11px] text-muted-foreground">Reasoning was truncated.</p> : null}
          </div>
        ) : null}
      </div>
    );
  }

  if (!collapseReasoning) {
    if (!item.text.trim()) return null;
    return (
      <div className="px-2 py-1 text-[13px] leading-5 whitespace-pre-wrap text-muted-foreground [overflow-wrap:anywhere]">
        {item.text}
        {item.truncated ? <p className="mt-1 text-[11px]">Reasoning was truncated.</p> : null}
      </div>
    );
  }

  const open = manualOpen ?? false;
  return (
    <div className="px-1">
      <button
        type="button"
        aria-expanded={open}
        aria-label={label}
        onClick={() => setManualOpen(!open)}
        className="sp-tool-enter flex w-full items-center gap-1.5 rounded-md py-1 text-left text-[13px] leading-5 text-muted-foreground hover:text-foreground"
      >
        <ChevronRight className={cn("size-3.5 shrink-0 transition-transform", open && "rotate-90")} aria-hidden />
        <span>{label}</span>
      </button>
      {open ? (
        <div className="ml-1.5 border-l border-border/70 py-1 pl-1.5 text-[13px] leading-5 whitespace-pre-wrap text-muted-foreground [overflow-wrap:anywhere]">
          {item.text}
          {item.truncated ? <p className="mt-1 text-[11px]">Reasoning was truncated.</p> : null}
        </div>
      ) : null}
    </div>
  );
}

function reasoningLabel(item: AgentContentItem): string {
  if (item.status === "running") return "Thinking";
  if (item.durationMs === undefined) return "Thought";
  if (item.durationMs < 10_000) return "Thought briefly";
  return `Thought for ${Math.max(10, Math.round(item.durationMs / 1000))} seconds`;
}

function AssistantContent({ item, tone }: { item: AgentContentItem; tone: "chat" | "log" }) {
  if (!item.text || isThinkingPlaceholder(item.text)) return null;
  return (
    <div
      className={cn(
        "px-2 text-foreground",
        tone === "log"
          ? "py-0.5 text-[13px] leading-5 [&_.workspace-markdown]:text-[13px] [&_.workspace-markdown]:leading-5 [&_.workspace-markdown]:text-foreground [&_.workspace-markdown_p]:my-0"
          : "py-1 text-[14px] leading-6",
        item.status === "running" && "sp-stream-text",
      )}
      data-testid={`agent-assistant-${item.id}`}
    >
      <MarkdownContent content={item.text} variant="workspace" className="max-w-none font-sans" />
    </div>
  );
}

function failedTool(tool: AgentToolItem): boolean {
  return tool.status === "failed" || tool.status === "timed_out";
}
