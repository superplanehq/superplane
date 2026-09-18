import { commandText, isCommandTool, toolFilePaths } from "@/lib/agentToolLabels";
import type { AgentActivityItem, AgentToolItem } from "./agentActivity";

export { commandDisplayText, commandText, isCommandTool, toolFilePaths } from "@/lib/agentToolLabels";

export type ToolActivityGroup = {
  type: "tool_activity_group";
  id: string;
  tools: AgentToolItem[];
};

export type ActivityEntry = Exclude<AgentActivityItem, AgentToolItem> | ToolActivityGroup;

type ActivityCount = {
  count: number;
  running: boolean;
};

type ActivityCounts = {
  fileChanges: ActivityCount;
  fileReads: ActivityCount;
  searches: ActivityCount;
  repositoryExplorations: ActivityCount;
  sources: ActivityCount;
  gitInspections: ActivityCount;
  checks: ActivityCount;
  specifications: ActivityCount;
  taskScores: ActivityCount;
  questions: ActivityCount;
  tasksCreated: ActivityCount;
  toolCalls: ActivityCount;
  terminalUses: ActivityCount;
};

export function activitySummaryLabel(tools: AgentToolItem[]): string {
  const counts = countActivities(tools);
  const parts = [
    actionCountLabel(counts.fileChanges, "editing", "edited", "file"),
    actionCountLabel(counts.fileReads, "exploring", "explored", "file"),
    repeatedActionLabel(counts.searches, "searching code", "searched code"),
    repeatedActionLabel(counts.repositoryExplorations, "exploring repository", "explored repository"),
    actionCountLabel(counts.sources, "researching", "researched", "source"),
    repeatedActionLabel(counts.gitInspections, "inspecting Git", "inspected Git"),
    actionCountLabel(counts.checks, "running", "ran", "check"),
    repeatedActionLabel(counts.specifications, "preparing specification", "prepared specification"),
    repeatedActionLabel(counts.taskScores, "scoring task", "scored task"),
    repeatedActionLabel(counts.questions, "preparing questions", "prepared questions"),
    actionCountLabel(counts.tasksCreated, "creating", "created", "task"),
    actionCountLabel(counts.toolCalls, "using", "used", "tool"),
    repeatedActionLabel(counts.terminalUses, "using terminal", "used terminal"),
  ].filter((part): part is string => Boolean(part));

  const label = parts.join(", ") || "Activity";
  return label.charAt(0).toUpperCase() + label.slice(1);
}

export function completedActivitySummaryLabel(tools: AgentToolItem[]): string {
  const counts = countActivities(tools);
  const parts = [
    groupedActivityLabel([counts.fileChanges], "editing files", "edited files"),
    groupedActivityLabel(
      [counts.fileReads, counts.searches, counts.repositoryExplorations, counts.gitInspections],
      "exploring codebase",
      "explored codebase",
    ),
    groupedActivityLabel([counts.sources], "researching sources", "researched sources"),
    groupedActivityLabel([counts.checks], "running checks", "ran checks"),
    groupedActivityLabel(
      [counts.specifications, counts.taskScores, counts.questions],
      "preparing task",
      "prepared task",
    ),
    groupedActivityLabel([counts.tasksCreated], "creating tasks", "created tasks"),
    groupedActivityLabel([counts.toolCalls], "using tools", "used tools"),
    groupedActivityLabel([counts.terminalUses], "using terminal", "used terminal"),
  ].filter((part): part is string => Boolean(part));

  const label = parts.join(", ") || "Activity";
  return label.charAt(0).toUpperCase() + label.slice(1);
}

export function groupToolRuns(entries: AgentActivityItem[]): ActivityEntry[] {
  const grouped: ActivityEntry[] = [];
  let index = 0;

  while (index < entries.length) {
    const entry = entries[index];
    if (entry.type !== "tool") {
      grouped.push(entry);
      index += 1;
      continue;
    }

    const tools: AgentToolItem[] = [entry];
    let nextIndex = index + 1;
    while (nextIndex < entries.length && entries[nextIndex].type === "tool") {
      tools.push(entries[nextIndex] as AgentToolItem);
      nextIndex += 1;
    }

    grouped.push({ type: "tool_activity_group", id: `tool-group-${entry.id}`, tools });
    index = nextIndex;
  }

  return grouped;
}

function countActivities(tools: AgentToolItem[]): ActivityCounts {
  const counts = emptyActivityCounts();
  for (const tool of tools) {
    const kind = tool.kind.toLowerCase();
    if (isCommandTool(tool)) countShellActivity(counts, tool);
    else if (isMCPTool(tool)) countMCPActivity(counts, tool);
    else if (kind === "read") increment(counts.fileReads, tool, fileCount(tool));
    else if (["search", "grep", "glob"].includes(kind)) increment(counts.searches, tool);
    else if (["web_search", "web_fetch"].includes(kind)) increment(counts.sources, tool);
    else if (["edit", "write"].includes(kind)) increment(counts.fileChanges, tool, fileCount(tool));
    else increment(counts.toolCalls, tool);
  }
  return counts;
}

function emptyActivityCounts(): ActivityCounts {
  const count = (): ActivityCount => ({ count: 0, running: false });
  return {
    fileChanges: count(),
    fileReads: count(),
    searches: count(),
    repositoryExplorations: count(),
    sources: count(),
    gitInspections: count(),
    checks: count(),
    specifications: count(),
    taskScores: count(),
    questions: count(),
    tasksCreated: count(),
    toolCalls: count(),
    terminalUses: count(),
  };
}

function countShellActivity(counts: ActivityCounts, tool: AgentToolItem): void {
  const command = commandText(tool.input);
  let classified = false;

  const fileReadCount = shellFileReadCount(command);
  if (fileReadCount > 0) {
    increment(counts.fileReads, tool, fileReadCount);
    classified = true;
  }
  classified = countShellExecutables(command, ["grep", "rg"], counts.searches, tool) || classified;
  classified =
    countShellExecutables(command, ["fd", "find", "ls", "tree"], counts.repositoryExplorations, tool) || classified;
  classified = countShellExecutables(command, ["curl", "wget"], counts.sources, tool) || classified;
  classified = countShellExecutables(command, ["git"], counts.gitInspections, tool) || classified;
  if (isCheckCommand(command)) {
    increment(counts.checks, tool);
    classified = true;
  }
  if (!classified) increment(counts.terminalUses, tool);
}

function shellFileReadCount(command: string): number {
  const pattern = new RegExp(
    `${shellCommandBoundary()}${shellCommandWrappers()}(?:\\S*/)?cat(?=\\s|$)([^\\n;&|]*)`,
    "gi",
  );
  return [...command.matchAll(pattern)].reduce((total, match) => total + shellFileArgumentCount(match[1]), 0);
}

function shellFileArgumentCount(argumentsText: string): number {
  const argumentsList = argumentsText.match(/"(?:\\.|[^"])*"|'[^']*'|\S+/g) ?? [];
  return argumentsList.filter((argument) => {
    const value = argument.replace(/^['"]|['"]$/g, "");
    return value && !value.startsWith("-") && !value.includes(">") && value !== "/dev/null";
  }).length;
}

function countMCPActivity(counts: ActivityCounts, tool: AgentToolItem): void {
  const identifier = `${tool.kind} ${tool.name}`.toLowerCase();
  if (identifier.includes("propose_spec")) {
    increment(counts.specifications, tool);
    return;
  }
  if (identifier.includes("confidence") || identifier.includes("clarity")) {
    increment(counts.taskScores, tool);
    return;
  }
  if (identifier.includes("survey")) {
    increment(counts.questions, tool);
    return;
  }
  if (identifier.includes("create_task")) {
    increment(counts.tasksCreated, tool);
    return;
  }
  increment(counts.toolCalls, tool);
}

function countShellExecutables(
  command: string,
  executables: string[],
  count: ActivityCount,
  tool: AgentToolItem,
): boolean {
  const names = executables.join("|");
  const pattern = new RegExp(`${shellCommandBoundary()}${shellCommandWrappers()}(?:\\S*/)?(?:${names})(?=\\s|$)`, "i");
  if (!pattern.test(command)) return false;
  increment(count, tool);
  return true;
}

function shellCommandBoundary(): string {
  return String.raw`(?:^|[\n;&|()]|\b(?:do|then)\b)\s*`;
}

function shellCommandWrappers(): string {
  return String.raw`(?:sudo\s+)?(?:env\s+)?(?:[\w.-]+=\S+\s+)*`;
}

function isCheckCommand(command: string): boolean {
  return /\b(?:make\s+\S*(?:test|lint|check|build)|go\s+test|pytest|cargo\s+test|(?:npm|pnpm|yarn|bun)\s+(?:run\s+)?(?:test|lint|check|build))\b/i.test(
    command,
  );
}

function increment(count: ActivityCount, tool: AgentToolItem, amount = 1): void {
  count.count += amount;
  count.running ||= tool.status === "running";
}

function fileCount(tool: AgentToolItem): number {
  return Math.max(1, toolFilePaths(tool.input).length);
}

function actionCountLabel(
  activity: ActivityCount,
  runningVerb: string,
  completedVerb: string,
  noun: string,
): string | undefined {
  if (activity.count === 0) return undefined;
  const verb = activity.running ? runningVerb : completedVerb;
  return `${verb} ${activity.count} ${activity.count === 1 ? noun : `${noun}s`}`;
}

function repeatedActionLabel(
  activity: ActivityCount,
  runningPhrase: string,
  completedPhrase: string,
): string | undefined {
  if (activity.count === 0) return undefined;
  const phrase = activity.running ? runningPhrase : completedPhrase;
  return activity.count === 1 ? phrase : `${phrase} ${activity.count} times`;
}

function groupedActivityLabel(
  activities: ActivityCount[],
  runningPhrase: string,
  completedPhrase: string,
): string | undefined {
  if (!activities.some((activity) => activity.count > 0)) return undefined;
  return activities.some((activity) => activity.running) ? runningPhrase : completedPhrase;
}

function isMCPTool(tool: AgentToolItem): boolean {
  return [tool.kind, tool.name].some((value) => {
    const normalized = value.toLowerCase();
    return normalized === "mcp" || normalized === "mcp_tool_call" || normalized.startsWith("mcp__");
  });
}
