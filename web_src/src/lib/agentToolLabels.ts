import type { AgentActivityItem, AgentToolItem } from "@/lib/agentActivity";
import { isHiddenAgentLiveLogText } from "@/lib/agentRunTelemetry";

const OUTPUT_PREVIEW_LINE_LIMIT = 3;
const FILE_KINDS = new Set(["read", "edit", "write"]);
const SEARCH_KINDS = new Set(["search", "grep", "glob"]);
const FETCH_KINDS = new Set(["web_search", "web_fetch"]);
const COMMAND_KINDS = new Set(["bash", "command_execution"]);
const SEARCH_FETCH_KEYS = ["pattern", "glob_pattern", "query", "url"];

export type AgentToolLabel = {
  action: string;
  detail?: string;
};

export type AgentToolOutputPreview = {
  lines: string[];
  hasMore: boolean;
};

export function isCommandTool(item: AgentActivityItem): boolean {
  return item.type === "tool" && COMMAND_KINDS.has(item.kind.toLowerCase());
}

export function commandDisplayText(input: string): string | undefined {
  return collapseWhitespace(commandText(input));
}

export function commandText(input: string): string {
  const trimmed = input.trim();
  if (!trimmed.startsWith("{")) return input;
  return stringFieldFromJSON(trimmed, ["command"]) ?? stringFromPartialJSON(trimmed, ["command"]) ?? input;
}

export function toolFilePaths(input: string): string[] {
  const trimmed = input.trim();
  if (!trimmed) return [];

  const jsonPaths = pathsFromJSON(trimmed);
  if (jsonPaths.length > 0) return unique(jsonPaths);
  if (trimmed.startsWith("{") || trimmed.startsWith("[")) return [];

  return unique(
    trimmed
      .split(/\r?\n/)
      .map((line) => line.trim())
      .filter(looksLikeFilePath),
  );
}

export function toolLineLabel(tool: AgentToolItem): string {
  const fileLabel = fileToolLabel(tool);
  if (fileLabel) return fileLabel;

  const kind = tool.kind.toLowerCase();
  const running = tool.status === "running";
  const input = toolInputPreview(tool);
  if (SEARCH_KINDS.has(kind)) {
    return input ? `${running ? "Searching" : "Searched"} ${input}` : running ? "Searching" : "Searched";
  }
  if (kind === "web_search") return running ? "Searching the web" : "Searched the web";
  if (kind === "web_fetch") return running ? "Fetching page" : "Fetched page";
  return tool.name || tool.kind || "Tool";
}

export function fileToolLabel(tool: AgentToolItem): string | undefined {
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

export function toolInputPreview(tool: AgentToolItem): string | undefined {
  if (!tool.input || FILE_KINDS.has(tool.kind.toLowerCase())) return undefined;

  const trimmed = tool.input.trim();
  if (!trimmed) return undefined;
  if (trimmed.startsWith("{") || trimmed.startsWith("[")) {
    return collapseWhitespace(
      stringFieldFromJSON(trimmed, SEARCH_FETCH_KEYS) ?? stringFromPartialJSON(trimmed, SEARCH_FETCH_KEYS),
    );
  }

  return collapseWhitespace(trimmed.split(/\r?\n/).find((line) => line.trim()));
}

export function displayFileName(path: string): string {
  const normalized = path.replace(/\\/g, "/").replace(/\/$/, "");
  return (normalized.split("/").at(-1) || normalized).replace(/\s+\(\d+ chars\)$/, "");
}

export function agentToolLabel(tool: AgentToolItem): AgentToolLabel {
  if (isCommandTool(tool)) {
    const command = commandDisplayText(tool.input);
    return command ? { action: command } : { action: toolDisplayName(tool) };
  }

  const fileLabel = fileToolLabel(tool);
  if (fileLabel) {
    return { action: fileLabel };
  }

  const kind = tool.kind.toLowerCase();
  if (SEARCH_KINDS.has(kind) || FETCH_KINDS.has(kind)) {
    const query = toolInputPreview(tool);
    return query ? { action: searchFetchAction(tool), detail: query } : { action: searchFetchAction(tool) };
  }

  return { action: toolDisplayName(tool) };
}

export function agentToolLabelText(tool: AgentToolItem): string {
  const label = agentToolLabel(tool);
  return label.detail ?? label.action;
}

export function agentToolOutputPreview(tool: AgentToolItem): AgentToolOutputPreview {
  const lines: string[] = [];
  let extraVisible = false;
  for (const raw of tool.output.split(/\r?\n/)) {
    const line = raw.trim();
    if (!line || isHiddenAgentLiveLogText(line)) {
      continue;
    }
    if (lines.length < OUTPUT_PREVIEW_LINE_LIMIT) {
      lines.push(line);
      continue;
    }
    extraVisible = true;
    break;
  }
  return { lines, hasMore: extraVisible || tool.truncated };
}

function toolDisplayName(tool: AgentToolItem): string {
  return tool.name || tool.kind || "Tool";
}

function searchFetchAction(tool: AgentToolItem): string {
  const kind = tool.kind.toLowerCase();
  const running = tool.status === "running";
  if (SEARCH_KINDS.has(kind)) {
    return running ? "Searching" : "Searched";
  }
  if (kind === "web_search") {
    return running ? "Searching the web" : "Searched the web";
  }
  return running ? "Fetching page" : "Fetched page";
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
    if (
      ["path", "file", "filePath", "file_path", "filename", "notebookPath", "notebook_path"].includes(key) &&
      typeof entry === "string"
    ) {
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

function stringFieldFromJSON(input: string, keys: string[]): string | undefined {
  try {
    const parsed = JSON.parse(input) as unknown;
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) return undefined;
    const record = parsed as Record<string, unknown>;
    for (const key of keys) {
      const value = record[key];
      if (typeof value === "string" && value.trim()) return value;
    }
  } catch {
    // A live tool input is often incomplete. Read known fields below.
  }
  return undefined;
}

function stringFromPartialJSON(input: string, keys: string[]): string | undefined {
  for (const key of keys) {
    const value = stringPropertyFromPartialJSON(input, key);
    if (value) return value;
  }
  return undefined;
}

function stringPropertyFromPartialJSON(input: string, key: string): string | undefined {
  const property = new RegExp(`"${key}"\\s*:\\s*"`).exec(input);
  if (!property) return undefined;

  let value = "";
  for (let index = property.index + property[0].length; index < input.length; index += 1) {
    const character = input[index];
    if (character === '"') return value || undefined;
    if (character !== "\\") {
      value += character;
      continue;
    }

    const escaped = input[index + 1];
    if (escaped === undefined) return value || undefined;
    value += decodeJSONEscape(escaped);
    index += 1;
  }
  return value || undefined;
}

function collapseWhitespace(value: string | undefined): string | undefined {
  const collapsed = value?.trim().replace(/\r?\n/g, " ");
  return collapsed || undefined;
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
