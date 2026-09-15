import { useState } from "react";

import { cn } from "@/lib/utils";
import { MarkdownContent } from "@/pages/app/Markdown";
import { CopyButton } from "@/ui/CopyButton";
import { Check, ChevronRight, CircleAlert, LoaderCircle, X } from "lucide-react";

import type { AgentActivity, AgentActivityItem, AgentContentItem, AgentToolItem } from "./agentActivity";
import {
  commandPreview,
  commandText,
  completedActivityLabel,
  groupConcurrentCommands,
  runningCommandPreview,
  type RunningCommandGroup,
} from "./agentActivitySummary";

export function AgentActivityView({ activity, live = false }: { activity: AgentActivity; live?: boolean }) {
  const entries = visibleActivityItems(activity.items, live);

  if (entries.length === 0 && !activity.truncated) return null;

  const tools = activity.items.filter((item): item is AgentToolItem => item.type === "tool");
  if (!live && tools.length > 0) {
    return <CompletedActivity activity={activity} entries={entries} tools={tools} />;
  }

  return (
    <div className="space-y-1 py-1.5" data-testid={`agent-activity-${activity.id}`}>
      <ActivityEntries entries={entries} truncated={activity.truncated} groupRunningCommands={live} />
    </div>
  );
}

function ActivityEntries({
  entries,
  truncated,
  groupRunningCommands = false,
}: {
  entries: AgentActivityItem[];
  truncated: boolean;
  groupRunningCommands?: boolean;
}) {
  const displayEntries = groupRunningCommands ? groupConcurrentCommands(entries) : entries;

  return (
    <>
      {displayEntries.map((item) => {
        if (item.type === "running_command_group") {
          return <RunningCommands key={item.id} group={item} />;
        }
        if (item.type === "content") {
          if (item.kind === "assistant") {
            return <AssistantContent key={item.id} item={item} />;
          }
          return <ReasoningContent key={item.id} item={item} />;
        }
        if (item.type === "tool") {
          return <ToolActivity key={item.id} tool={item} />;
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

function RunningCommands({ group }: { group: RunningCommandGroup }) {
  const [open, setOpen] = useState(true);
  const label = `Running ${group.tools.length} commands`;

  return (
    <div className="sp-tool-enter px-1" data-testid={group.id}>
      <button
        type="button"
        aria-expanded={open}
        aria-label={label}
        onClick={() => setOpen((current) => !current)}
        className="flex w-full items-center gap-1.5 rounded-md px-1 py-1 text-left text-[13px] leading-5 text-muted-foreground hover:text-foreground"
      >
        <ChevronRight className={cn("size-3.5 shrink-0 transition-transform", open && "rotate-90")} aria-hidden />
        <span className="flex shrink-0">
          <ToolStatusIcon status="running" />
        </span>
        <span className="sp-ai-thinking min-w-0 flex-1" data-text={label}>
          {label}
        </span>
      </button>
      {open ? (
        <div className="ml-1 space-y-1 border-l border-border/70 pl-1.5">
          {group.tools.map((tool) => (
            <ToolActivity key={tool.id} tool={tool} />
          ))}
        </div>
      ) : null}
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
  const label = completedActivityLabel(tools);

  return (
    <div className="space-y-1 py-1.5" data-testid={`agent-activity-${activity.id}`}>
      <div>
        <button
          type="button"
          aria-expanded={open}
          aria-label={label}
          onClick={() => setOpen((current) => !current)}
          data-testid={`agent-activity-summary-${activity.id}`}
          className="sp-tool-enter flex w-full items-center gap-1.5 rounded-md py-1 text-left text-[13px] leading-5 text-muted-foreground hover:text-foreground"
        >
          <ChevronRight className={cn("size-3.5 shrink-0 transition-transform", open && "rotate-90")} aria-hidden />
          <span className="flex shrink-0">
            <ToolStatusIcon status="passed" />
          </span>
          <span className="min-w-0 flex-1">{label}</span>
        </button>
      </div>
      {open ? (
        <div
          className="ml-1 space-y-1 border-l border-border/70 pl-1.5"
          data-testid={`agent-activity-details-${activity.id}`}
        >
          <ActivityEntries entries={entries} truncated={activity.truncated} />
        </div>
      ) : null}
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

function ToolActivity({ tool }: { tool: AgentToolItem }) {
  const [open, setOpen] = useState(false);
  const label = toolLabel(tool);
  const summary = toolSummary(tool);
  const runningCommand = runningCommandPreview(tool);

  return (
    <div className="sp-tool-enter px-1" data-testid={`agent-tool-${tool.id}`}>
      <button
        type="button"
        aria-expanded={open}
        aria-label={label}
        onClick={() => setOpen((current) => !current)}
        className="flex w-full items-start gap-1.5 rounded-md px-1 py-1 text-left text-[13px] leading-5 text-muted-foreground hover:text-foreground"
      >
        <ChevronRight
          className={cn("mt-[3px] size-3.5 shrink-0 transition-transform", open && "rotate-90")}
          aria-hidden
        />
        <span className="mt-[3px] flex shrink-0">
          <ToolStatusIcon status={tool.status} />
        </span>
        <span className="min-w-0 flex-1">
          <span className="block">{runningCommand ? <RunningCommandLabel command={runningCommand} /> : label}</span>
          {summary && !open && !runningCommand ? (
            <span
              className="block truncate font-mono text-[11px] leading-4 whitespace-nowrap opacity-80"
              data-testid={`agent-tool-summary-${tool.id}`}
            >
              {summary}
            </span>
          ) : null}
        </span>
        {tool.durationMs !== undefined && tool.status !== "running" ? (
          <span className="mt-[1px] ml-auto shrink-0 text-[11px] tabular-nums">{formatDuration(tool.durationMs)}</span>
        ) : null}
      </button>
      {open ? <ToolDetails tool={tool} /> : null}
    </div>
  );
}

function RunningCommandLabel({ command }: { command: string }) {
  return (
    <span className="flex min-w-0 items-baseline gap-1.5 whitespace-nowrap">
      <span className="sp-ai-thinking shrink-0" data-text="Running">
        Running
      </span>
      <span className="sp-running-command truncate font-mono text-[11px]">{command}</span>
    </span>
  );
}

function ToolDetails({ tool }: { tool: AgentToolItem }) {
  const outputs = groupedOutputs(tool);
  const input = ["bash", "command_execution"].includes(tool.kind.toLowerCase()) ? commandText(tool.input) : tool.input;
  return (
    <div className="ml-6 space-y-3 py-1 pr-1" data-testid={`agent-tool-details-${tool.id}`}>
      {input ? <CodeBlock id={`${tool.id}-command`} label={toolInputLabel(tool.kind)} value={input} /> : null}
      {outputs.length > 0 ? (
        outputs.map((output, index) => (
          <CodeBlock
            key={`${output.stream}-${index}`}
            id={`${tool.id}-${output.stream}-${index}`}
            label={output.label}
            value={output.text}
          />
        ))
      ) : tool.output ? (
        <CodeBlock id={`${tool.id}-output`} label="Output" value={tool.output} />
      ) : null}
      {!tool.output && tool.status !== "running" ? (
        <p className="text-[11px] leading-4 text-muted-foreground">No output.</p>
      ) : null}
      {tool.exitCode !== undefined || tool.signal ? (
        <p className="text-[11px] leading-4 text-muted-foreground">
          {tool.exitCode !== undefined ? `Exit code ${tool.exitCode}` : ""}
          {tool.exitCode !== undefined && tool.signal ? " · " : ""}
          {tool.signal ? `Signal ${tool.signal}` : ""}
        </p>
      ) : null}
      {terminalStateText(tool.status) ? (
        <p className="text-[11px] leading-4 text-muted-foreground">{terminalStateText(tool.status)}</p>
      ) : null}
      {tool.truncated ? <p className="text-[11px] leading-4 text-muted-foreground">Output was truncated.</p> : null}
    </div>
  );
}

function terminalStateText(status: AgentToolItem["status"]): string | undefined {
  if (status === "failed") return "Tool failed.";
  if (status === "cancelled") return "Tool was cancelled.";
  if (status === "timed_out") return "Tool timed out.";
  if (status === "interrupted") return "Tool was interrupted.";
  return undefined;
}

function groupedOutputs(tool: AgentToolItem): Array<{ stream: string; label: string; text: string }> {
  const grouped: Array<{ stream: string; label: string; text: string }> = [];
  for (const output of tool.outputStreams) {
    const last = grouped.at(-1);
    if (last?.stream === output.stream) {
      last.text += output.text;
      continue;
    }
    grouped.push({
      stream: output.stream,
      label: output.stream === "stderr" ? "Error output" : output.stream === "stdout" ? "Standard output" : "Output",
      text: output.text,
    });
  }
  return grouped;
}

function CodeBlock({ id, label, value }: { id: string; label: string; value: string }) {
  const output = label === "Output" || label === "Standard output" || label === "Error output";
  const errorOutput = label === "Error output";
  return (
    <div
      className={cn(
        "group relative py-0.5",
        output && "border-l border-border pl-3 text-muted-foreground",
        errorOutput && "border-destructive/60 text-destructive",
        !output && "text-foreground",
      )}
      data-testid={`agent-detail-${id}`}
    >
      <span className="sr-only">{label}</span>
      <pre className="max-h-52 overflow-auto pr-8 font-mono text-[11px] leading-4 whitespace-pre-wrap [overflow-wrap:anywhere]">
        {value}
      </pre>
      <CopyButton
        text={value}
        ariaLabel={`Copy ${label.toLowerCase()}`}
        copiedAriaLabel={`${label} copied`}
        className="absolute top-0 right-0 size-6 text-muted-foreground opacity-0 transition-opacity group-hover:opacity-100 hover:bg-transparent focus-visible:opacity-100 data-[copied=true]:opacity-100 dark:hover:bg-transparent"
      />
    </div>
  );
}

function ToolStatusIcon({ status }: { status: AgentToolItem["status"] }) {
  if (status === "running") return <LoaderCircle className="size-3.5 shrink-0 animate-spin" aria-hidden />;
  if (status === "passed") return <Check className="size-3.5 shrink-0" aria-hidden />;
  if (status === "failed" || status === "timed_out") {
    return <CircleAlert className="size-3.5 shrink-0 text-destructive" aria-hidden />;
  }
  return <X className="size-3.5 shrink-0" aria-hidden />;
}

function toolLabel(tool: AgentToolItem): string {
  const command = runningCommandPreview(tool);
  if (command) return `Running ${command}`;

  const fileLabel = fileToolLabel(tool);
  if (fileLabel) return fileLabel;

  const running = tool.status === "running";
  const labels: Record<string, [running: string, completed: string]> = {
    bash: ["Running command", "Ran command"],
    command_execution: ["Running command", "Ran command"],
    read: ["Reading file", "Read file"],
    search: ["Searching files", "Searched files"],
    grep: ["Searching files", "Searched files"],
    glob: ["Searching files", "Searched files"],
    web_search: ["Searching the web", "Searched the web"],
    web_fetch: ["Fetching page", "Fetched page"],
    edit: ["Editing file", "Edited file"],
    write: ["Creating file", "Created file"],
  };
  const knownLabel = labels[tool.kind.toLowerCase()];
  if (knownLabel) return knownLabel[running ? 0 : 1];
  return `${running ? "Running" : "Ran"} ${tool.name || "tool"}`;
}

function fileToolLabel(tool: AgentToolItem): string | undefined {
  const kind = tool.kind.toLowerCase();
  if (!["read", "edit", "write"].includes(kind)) return undefined;

  const files = toolFilePaths(tool.input);
  const fileCount = files.length;
  const fileName = fileCount === 1 ? displayFileName(files[0]) : undefined;
  const running = tool.status === "running";

  if (kind === "read") {
    return readToolLabel(running, fileCount, fileName);
  }
  if (kind === "edit") {
    if (fileCount > 1) return `${running ? "Editing" : "Edited"} ${fileCount} files`;
    if (fileName) return `${running ? "Editing" : "Edited"} ${fileName}`;
  }
  if (kind === "write" && fileName) {
    return `${running ? "Creating" : "Created"} ${fileName}`;
  }
  return undefined;
}

function readToolLabel(running: boolean, fileCount: number, fileName?: string): string | undefined {
  if (fileCount > 1) return `${running ? "Reading" : "Read"} ${fileCount} files`;
  if (fileName) return `${running ? "Reading" : "Read"} ${fileName}`;
  return undefined;
}

function toolSummary(tool: AgentToolItem): string | undefined {
  if (!tool.input) return undefined;
  const kind = tool.kind.toLowerCase();
  if (["read", "edit", "write"].includes(kind)) return undefined;
  if (["bash", "command_execution"].includes(kind)) return commandPreview(tool.input);
  return firstNonEmptyLine(tool.input);
}

function firstNonEmptyLine(value: string): string | undefined {
  return value
    .split(/\r?\n/)
    .find((line) => line.trim())
    ?.trim();
}

function toolFilePaths(input: string): string[] {
  const trimmed = input.trim();
  if (!trimmed) return [];

  const jsonPaths = pathsFromJSON(trimmed);
  if (jsonPaths.length > 0) return unique(jsonPaths);
  if (trimmed.startsWith("{") || trimmed.startsWith("[")) return [];

  const lines = trimmed
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(looksLikeFilePath);
  return unique(lines);
}

function pathsFromJSON(input: string): string[] {
  if (!input.startsWith("{") && !input.startsWith("[")) return [];
  try {
    const paths: string[] = [];
    collectJSONPaths(JSON.parse(input) as unknown, paths);
    return paths;
  } catch {
    return [];
  }
}

function collectJSONPaths(value: unknown, paths: string[]): void {
  if (Array.isArray(value)) {
    value.forEach((entry) => collectJSONPaths(entry, paths));
    return;
  }
  if (!value || typeof value !== "object") return;
  for (const [key, entry] of Object.entries(value)) {
    if (["path", "file", "file_path", "filename"].includes(key) && typeof entry === "string") {
      paths.push(entry);
      continue;
    }
    if (["changes", "files"].includes(key)) collectJSONPaths(entry, paths);
  }
}

function looksLikeFilePath(value: string): boolean {
  if (!value || value.length > 2048 || /[|;&`]/.test(value)) return false;
  return value.includes("/") || /^\.?[\w -]+\.[a-z0-9]{1,12}$/i.test(value);
}

function unique(values: string[]): string[] {
  return [...new Set(values.filter(Boolean))];
}

function displayFileName(path: string): string {
  const normalized = path.replace(/\\/g, "/").replace(/\/$/, "");
  return (normalized.split("/").at(-1) || normalized).replace(/\s+\(\d+ chars\)$/, "");
}

function toolInputLabel(kind: string): string {
  const normalized = kind.toLowerCase();
  if (normalized === "bash" || normalized === "command_execution") return "Command";
  if (["search", "grep", "glob", "web_search"].includes(normalized)) return "Query";
  if (normalized === "web_fetch") return "URL";
  if (["read", "edit", "write"].includes(normalized)) return "File";
  return "Input";
}

function formatDuration(durationMs: number): string {
  return durationMs < 1000 ? `${Math.round(durationMs)} ms` : `${(durationMs / 1000).toFixed(1)} s`;
}
