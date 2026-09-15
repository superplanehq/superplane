import type { AgentActivityItem, AgentToolItem } from "./agentActivity";

export type RunningCommandGroup = {
  type: "running_command_group";
  id: string;
  tools: AgentToolItem[];
};

export type ActivityEntry = AgentActivityItem | RunningCommandGroup;

export function completedActivityLabel(tools: AgentToolItem[]): string {
  const counts = {
    commands: 0,
    mcpCalls: 0,
    fileReads: 0,
    searches: 0,
    webRequests: 0,
    fileChanges: 0,
    toolCalls: 0,
  };

  for (const tool of tools) {
    const kind = tool.kind.toLowerCase();
    if (isCommandTool(tool)) counts.commands += 1;
    else if (isMCPTool(tool)) counts.mcpCalls += 1;
    else if (kind === "read") counts.fileReads += 1;
    else if (["search", "grep", "glob", "web_search"].includes(kind)) counts.searches += 1;
    else if (kind === "web_fetch") counts.webRequests += 1;
    else if (["edit", "write"].includes(kind)) counts.fileChanges += 1;
    else counts.toolCalls += 1;
  }

  const parts = [
    countLabel(counts.commands, "command"),
    countLabel(counts.mcpCalls, "MCP call"),
    countLabel(counts.fileReads, "file read"),
    countLabel(counts.searches, "search", "searches"),
    countLabel(counts.webRequests, "web request"),
    countLabel(counts.fileChanges, "file change"),
    countLabel(counts.toolCalls, "other tool call"),
  ].filter((part): part is string => Boolean(part));

  return `Ran ${joinSummaryParts(parts)}`;
}

export function isCommandTool(item: AgentActivityItem): item is AgentToolItem {
  if (item.type !== "tool") return false;
  return ["bash", "command_execution"].includes(item.kind.toLowerCase());
}

export function groupConcurrentCommands(entries: AgentActivityItem[]): ActivityEntry[] {
  const grouped: ActivityEntry[] = [];
  let index = 0;

  while (index < entries.length) {
    const entry = entries[index];
    if (!isRunningCommand(entry)) {
      grouped.push(entry);
      index += 1;
      continue;
    }

    const tools: AgentToolItem[] = [entry];
    let nextIndex = index + 1;
    while (nextIndex < entries.length && isRunningCommand(entries[nextIndex])) {
      tools.push(entries[nextIndex] as AgentToolItem);
      nextIndex += 1;
    }

    grouped.push(
      tools.length === 1 ? entry : { type: "running_command_group", id: `running-commands-${entry.id}`, tools },
    );
    index = nextIndex;
  }

  return grouped;
}

export function runningCommandPreview(tool: AgentToolItem): string | undefined {
  if (tool.status !== "running" || !isCommandTool(tool)) return undefined;
  return commandPreview(tool.input);
}

export function commandPreview(input: string): string | undefined {
  const command = firstNonEmptyLine(commandText(input));
  if (!command) return undefined;

  const separator = command.indexOf("&&");
  if (separator < 0 || !/^cd(?:\s|$)/.test(command.slice(0, separator).trim())) return command;
  return command.slice(separator + 2).trim() || command;
}

function countLabel(count: number, singular: string, plural = `${singular}s`): string | undefined {
  if (count === 0) return undefined;
  return `${count} ${count === 1 ? singular : plural}`;
}

function joinSummaryParts(parts: string[]): string {
  if (parts.length <= 1) return parts[0] ?? "activity";
  if (parts.length === 2) return `${parts[0]} and ${parts[1]}`;
  return `${parts.slice(0, -1).join(", ")}, and ${parts.at(-1)}`;
}

function isMCPTool(tool: AgentToolItem): boolean {
  return [tool.kind, tool.name].some((value) => {
    const normalized = value.toLowerCase();
    return normalized === "mcp" || normalized === "mcp_tool_call" || normalized.startsWith("mcp__");
  });
}

function isRunningCommand(item: AgentActivityItem | undefined): item is AgentToolItem {
  return Boolean(item && isCommandTool(item) && item.status === "running");
}

function firstNonEmptyLine(value: string): string | undefined {
  return value
    .split(/\r?\n/)
    .find((line) => line.trim())
    ?.trim();
}

export function commandText(input: string): string {
  const trimmed = input.trim();
  if (!trimmed.startsWith("{")) return input;

  try {
    const parsed = JSON.parse(trimmed) as unknown;
    if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
      const command = (parsed as Record<string, unknown>).command;
      if (typeof command === "string") return command;
    }
  } catch {
    // A live tool input is often incomplete. Read the available command below.
  }

  return commandFromPartialJSON(trimmed) ?? input;
}

function commandFromPartialJSON(input: string): string | undefined {
  const property = /"command"\s*:\s*"/g.exec(input);
  if (!property) return undefined;

  let command = "";
  for (let index = property.index + property[0].length; index < input.length; index += 1) {
    const character = input[index];
    if (character === '"') return command;
    if (character !== "\\") {
      command += character;
      continue;
    }

    const escaped = input[index + 1];
    if (escaped === undefined) return command;
    const decoded = decodeJSONEscape(escaped);
    command += decoded;
    index += 1;
  }
  return command;
}

function decodeJSONEscape(character: string): string {
  const escapes: Record<string, string> = {
    '"': '"',
    "\\": "\\",
    "/": "/",
    b: "\b",
    f: "\f",
    n: "\n",
    r: "\r",
    t: "\t",
  };
  return escapes[character] ?? character;
}
