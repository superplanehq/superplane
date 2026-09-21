import type { AgentActivityItem, AgentToolItem } from "@/lib/agentActivity";
import { isHiddenAgentLiveLogText } from "@/lib/agentRunTelemetry";

const OUTPUT_PREVIEW_LINE_LIMIT = 3;
const FILE_KINDS = new Set(["read", "edit", "write"]);
const SEARCH_KINDS = new Set(["search", "grep", "glob"]);
const FETCH_KINDS = new Set(["web_search", "web_fetch"]);
const COMMAND_KINDS = new Set(["bash", "command_execution"]);
const FILE_KEYS = new Set(["path", "file", "filePath", "file_path", "filename", "notebookPath", "notebook_path"]);

export type AgentToolOutputPreview = {
  lines: string[];
  hasMore: boolean;
};

type AgentToolDisplayInput = Pick<AgentToolItem, "input" | "kind" | "name">;

export function isCommandKind(kind: string): boolean {
  return COMMAND_KINDS.has(kind.toLowerCase());
}

export function isCommandTool(item: AgentActivityItem): boolean {
  return item.type === "tool" && isCommandKind(item.kind);
}

export function commandText(input: string): string {
  const trimmed = input.trim();
  if (!trimmed) return "";
  if (!startsWithJSON(trimmed)) return input;
  return stringField(parseJSONObject(trimmed), ["command"]) ?? "";
}

export function toolFilePaths(input: string): string[] {
  const trimmed = input.trim();
  if (!trimmed) return [];

  if (startsWithJSON(trimmed)) {
    const paths: string[] = [];
    collectJSONPaths(parseJSON(trimmed), paths);
    return unique(paths);
  }

  return unique(
    trimmed
      .split(/\r?\n/)
      .map((line) => line.trim())
      .filter(looksLikeFilePath),
  );
}

export function agentToolDisplayText(tool: AgentToolDisplayInput): string {
  return toolInputDisplayText(tool) ?? toolDisplayName(tool);
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

function toolInputDisplayText(tool: AgentToolDisplayInput): string | undefined {
  const input = tool.input.trim();
  if (!input) return undefined;

  const kind = normalizedKind(tool);
  if (!startsWithJSON(input)) {
    return collapseWhitespace(input);
  }

  const value = parseJSONObject(input);
  if (!value) return undefined;
  if (COMMAND_KINDS.has(kind)) return collapseWhitespace(stringField(value, ["command"]));
  if (FILE_KINDS.has(kind)) return fileInputDisplayText(value);
  if (SEARCH_KINDS.has(kind)) return collapseWhitespace(stringField(value, ["pattern", "glob_pattern", "query"]));
  if (FETCH_KINDS.has(kind)) return collapseWhitespace(stringField(value, ["url", "query"]));
  return undefined;
}

function fileInputDisplayText(value: Record<string, unknown>): string | undefined {
  const paths: string[] = [];
  collectJSONPaths(value, paths);
  const uniquePaths = unique(paths);
  if (uniquePaths.length === 1) return uniquePaths[0];
  if (uniquePaths.length > 1) return `${uniquePaths.length} files`;
  return undefined;
}

function normalizedKind(tool: Pick<AgentToolItem, "kind">): string {
  return tool.kind.toLowerCase();
}

function toolDisplayName(tool: AgentToolDisplayInput): string {
  return tool.name || tool.kind || "Tool";
}

function startsWithJSON(input: string): boolean {
  return input.startsWith("{") || input.startsWith("[");
}

function parseJSON(input: string): unknown {
  try {
    return JSON.parse(input) as unknown;
  } catch {
    return undefined;
  }
}

function parseJSONObject(input: string): Record<string, unknown> | undefined {
  const value = parseJSON(input);
  if (!value || typeof value !== "object" || Array.isArray(value)) return undefined;
  return value as Record<string, unknown>;
}

function stringField(value: Record<string, unknown> | undefined, keys: string[]): string | undefined {
  if (!value) return undefined;
  for (const key of keys) {
    const field = value[key];
    if (typeof field === "string" && field.trim()) return field;
  }
  return undefined;
}

function collectJSONPaths(value: unknown, paths: string[]): void {
  if (Array.isArray(value)) {
    value.forEach((entry) => collectJSONPaths(entry, paths));
    return;
  }
  if (!value || typeof value !== "object") return;
  for (const [key, entry] of Object.entries(value)) {
    if (FILE_KEYS.has(key) && typeof entry === "string") {
      paths.push(entry);
      continue;
    }
    if (key === "changes" || key === "files") collectJSONPaths(entry, paths);
  }
}

function looksLikeFilePath(value: string): boolean {
  if (!value || value.length > 2048 || /[|;&`]/.test(value)) return false;
  return value.includes("/") || /^\.?[\w -]+\.[a-z0-9]{1,12}$/i.test(value);
}

function unique(values: string[]): string[] {
  return [...new Set(values.filter(Boolean))];
}

function collapseWhitespace(value: string | undefined): string | undefined {
  const collapsed = value?.trim().replace(/\r?\n/g, " ");
  return collapsed || undefined;
}
