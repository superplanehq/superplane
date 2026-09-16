import { useState } from "react";

import { cn } from "@/lib/utils";
import { MarkdownContent } from "@/pages/app/Markdown";
import { ChevronRight } from "lucide-react";

import type { AgentActivity, AgentActivityItem, AgentContentItem, AgentToolItem } from "./agentActivity";
import { AnimatedThinkingState } from "./AnimatedThinkingState";
import {
  activitySummaryLabel,
  commandDisplayText,
  completedActivitySummaryLabel,
  groupToolRuns,
  isCommandTool,
  toolFilePaths,
  type ToolActivityGroup,
} from "./agentActivitySummary";

export function AgentActivityView({ activity, live = false }: { activity: AgentActivity; live?: boolean }) {
  const entries = visibleActivityItems(activity.items, live);
  if (entries.length === 0 && !activity.truncated) return null;

  const tools = activity.items.filter((item): item is AgentToolItem => item.type === "tool");
  if (!live && tools.length > 0) {
    return <CompletedActivity activity={activity} entries={entries} tools={tools} />;
  }

  return (
    <div className="space-y-0 px-2" data-testid={`agent-activity-${activity.id}`}>
      <ActivityEntries entries={entries} truncated={activity.truncated} />
    </div>
  );
}

function CompletedActivity({
  activity,
  entries,
  tools,
}: {
  activity: AgentActivity;
  entries: AgentActivityItem[];
  tools: AgentToolItem[];
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
          <ActivityEntries entries={entries} truncated={activity.truncated} />
        </div>
      ) : null}
    </div>
  );
}

function ActivityEntries({
  entries,
  truncated,
  groupTools = true,
}: {
  entries: AgentActivityItem[];
  truncated: boolean;
  groupTools?: boolean;
}) {
  const displayEntries = groupTools ? groupToolRuns(entries) : entries;
  return (
    <>
      {displayEntries.map((item) => {
        if (item.type === "tool_activity_group") {
          return <ToolGroup key={item.id} group={item} />;
        }
        if (item.type === "tool") {
          return <ToolLine key={item.id} tool={item} />;
        }
        if (item.type === "content") {
          if (item.kind === "assistant") {
            return <AssistantContent key={item.id} item={item} />;
          }
          return <ReasoningContent key={item.id} item={item} />;
        }
        return (
          <p key={item.id} className="px-2 py-1 text-[12px] leading-5 text-muted-foreground">
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

function ToolGroup({ group }: { group: ToolActivityGroup }) {
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
            <ToolLine key={tool.id} tool={tool} />
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

function ToolLine({ tool }: { tool: AgentToolItem }) {
  if (isCommandTool(tool)) {
    return <CommandLine tool={tool} />;
  }

  const label = toolLineLabel(tool);
  const input = toolInputPreview(tool);
  return (
    <div
      className={cn(
        "sp-tool-enter flex min-w-0 items-baseline gap-2 px-1 py-0.5 text-[12px] leading-5 text-muted-foreground",
        failedTool(tool) && "text-destructive",
      )}
      data-testid={`agent-tool-${tool.id}`}
      data-status={tool.status}
    >
      <span className="shrink-0">{label}</span>
      {input ? (
        <span className="min-w-0 flex-1 truncate font-mono text-[11px] opacity-80" title={input}>
          {input}
        </span>
      ) : null}
    </div>
  );
}

function CommandLine({ tool }: { tool: AgentToolItem }) {
  const command = commandDisplayText(tool.input) ?? "Command";
  return (
    <div className="flex min-w-0 items-center px-1 py-0.5">
      <code
        className={cn(
          "block min-w-0 flex-1 truncate font-mono text-[12px] leading-5 whitespace-nowrap text-muted-foreground",
          failedTool(tool) && "text-destructive",
        )}
        data-testid={`agent-tool-${tool.id}`}
        data-status={tool.status}
        title={command}
      >
        {command}
      </code>
    </div>
  );
}

function visibleActivityItems(items: AgentActivityItem[], live: boolean): AgentActivityItem[] {
  return items.filter((item, index) => isVisibleActivityItem(item, index, items, live));
}

function isVisibleActivityItem(
  item: AgentActivityItem,
  index: number,
  items: AgentActivityItem[],
  live: boolean,
): boolean {
  if (item.type === "content" && item.kind === "assistant") {
    if (!item.text.trim()) return false;
    return live || items.slice(index + 1).some((candidate) => candidate.type === "tool");
  }
  return item.type !== "content" || item.kind !== "reasoning" || item.status === "running" || Boolean(item.text.trim());
}

function ReasoningContent({ item }: { item: AgentContentItem }) {
  const [manualOpen, setManualOpen] = useState<boolean | null>(null);
  const running = item.status === "running";
  const label = reasoningLabel(item);

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

function AssistantContent({ item }: { item: AgentContentItem }) {
  if (!item.text) return null;
  return (
    <div
      className={cn("px-2 py-1 text-[14px] leading-6 text-foreground", item.status === "running" && "sp-stream-text")}
      data-testid={`agent-assistant-${item.id}`}
    >
      <MarkdownContent content={item.text} variant="workspace" className="max-w-none font-sans" />
    </div>
  );
}

function toolLineLabel(tool: AgentToolItem): string {
  const fileLabel = fileToolLabel(tool);
  if (fileLabel) return fileLabel;

  const kind = tool.kind.toLowerCase();
  const running = tool.status === "running";
  const input = toolInputPreview(tool);
  if (["search", "grep", "glob"].includes(kind)) {
    return input ? `${running ? "Searching" : "Searched"} ${input}` : running ? "Searching" : "Searched";
  }
  if (kind === "web_search") return running ? "Searching the web" : "Searched the web";
  if (kind === "web_fetch") return running ? "Fetching page" : "Fetched page";
  return tool.name || tool.kind || "Tool";
}

function fileToolLabel(tool: AgentToolItem): string | undefined {
  const kind = tool.kind.toLowerCase();
  const verbs: Record<string, [running: string, completed: string]> = {
    read: ["Exploring", "Explored"],
    edit: ["Editing", "Edited"],
    write: ["Creating", "Created"],
  };
  const action = verbs[kind];
  if (!action) return undefined;

  const files = toolFilePaths(tool.input);
  const count = files.length;
  const fileName = count === 1 ? displayFileName(files[0]) : undefined;
  const verb = action[tool.status === "running" ? 0 : 1];

  if (count > 1) return `${verb} ${count} files`;
  return fileName ? `${verb} ${fileName}` : `${verb} file`;
}

function toolInputPreview(tool: AgentToolItem): string | undefined {
  if (!tool.input || ["read", "edit", "write"].includes(tool.kind.toLowerCase())) return undefined;
  return tool.input
    .split(/\r?\n/)
    .find((line) => line.trim())
    ?.trim();
}

function displayFileName(path: string): string {
  const normalized = path.replace(/\\/g, "/").replace(/\/$/, "");
  return (normalized.split("/").at(-1) || normalized).replace(/\s+\(\d+ chars\)$/, "");
}

function failedTool(tool: AgentToolItem): boolean {
  return tool.status === "failed" || tool.status === "timed_out";
}
