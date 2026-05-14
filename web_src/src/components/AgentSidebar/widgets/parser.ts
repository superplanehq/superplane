import YAML from "js-yaml";

// --- Types ---

export type MarkdownSegment = { type: "markdown"; content: string };
export type ButtonsSegment = { type: "buttons"; prompt: string; items: string[] };
export type ConfirmSegment = { type: "confirm"; message: string; yes: string; no: string };
export type ChartSegment = { type: "chart"; config: ChartConfig };
export type CollapseSegment = { type: "collapse"; title: string; content: string };
export type MermaidSegment = { type: "mermaid"; content: string };
export type StepsSegment = { type: "steps"; items: StepItem[] };
export type SuccessSegment = { type: "success"; content: string };
export type ErrorSegment = { type: "error"; content: string };

export type Segment =
  | MarkdownSegment
  | ButtonsSegment
  | ConfirmSegment
  | ChartSegment
  | CollapseSegment
  | MermaidSegment
  | StepsSegment
  | SuccessSegment
  | ErrorSegment;

export type StepItem = { done: boolean; text: string };

export type ChartConfig = {
  type: "line" | "bar" | "area" | "pie";
  title?: string;
  x?: string[];
  series?: { name: string; data: number[]; color?: string }[];
  data?: { name: string; value: number; color?: string }[];
};

// --- Parser ---

const BLOCK_RE = /^\s*:::(\w+)(?:\s+(.*))?$/;
const BLOCK_END_RE = /^\s*:::$/;
const MERMAID_FENCE_START = /^\s*```mermaid\s*$/;
const FENCE_END = /^\s*```\s*$/;

export function parseAgentContent(content: string): Segment[] {
  if (!content) return [];

  const lines = content.split("\n");
  const segments: Segment[] = [];
  let markdownBuffer: string[] = [];
  let blockType: string | null = null;
  let blockMeta = "";
  let blockLines: string[] = [];

  function flushMarkdown() {
    const text = markdownBuffer.join("\n").trim();
    if (text) {
      segments.push({ type: "markdown", content: text });
    }
    markdownBuffer = [];
  }

  function flushBlock() {
    if (!blockType) return;
    const raw = blockLines.join("\n");
    const segment = parseBlock(blockType, blockMeta, raw);
    if (segment) {
      segments.push(segment);
    }
    blockType = null;
    blockMeta = "";
    blockLines = [];
  }

  let inMermaidFence = false;
  let mermaidLines: string[] = [];

  for (const line of lines) {
    // Handle ```mermaid fenced code blocks
    if (inMermaidFence) {
      if (FENCE_END.test(line.trim())) {
        flushMarkdown();
        segments.push({ type: "mermaid", content: mermaidLines.join("\n") });
        mermaidLines = [];
        inMermaidFence = false;
      } else {
        mermaidLines.push(line);
      }
      continue;
    }

    if (blockType) {
      if (BLOCK_END_RE.test(line.trim())) {
        flushBlock();
      } else {
        blockLines.push(line);
      }
    } else if (MERMAID_FENCE_START.test(line.trim())) {
      flushMarkdown();
      inMermaidFence = true;
      mermaidLines = [];
    } else {
      const match = line.match(BLOCK_RE);
      if (match) {
        flushMarkdown();
        blockType = match[1];
        blockMeta = match[2] || "";
      } else {
        markdownBuffer.push(line);
      }
    }
  }

  // Handle unclosed blocks gracefully
  if (blockType) {
    flushBlock();
  }
  flushMarkdown();

  return segments;
}

function parseBlock(type: string, meta: string, raw: string): Segment | null {
  switch (type) {
    case "buttons":
      return parseButtons(raw);
    case "confirm":
      return parseConfirm(raw);
    case "chart":
      return parseChart(raw);
    case "collapse":
      return parseCollapse(meta, raw);
    case "steps":
      return parseSteps(raw);
    case "success":
      return { type: "success", content: raw.trim() };
    case "error":
      return { type: "error", content: raw.trim() };
    default:
      return { type: "markdown", content: `:::${type} ${meta}\n${raw}\n:::` };
  }
}

function parseButtons(raw: string): ButtonsSegment {
  const lines = raw.split("\n").filter((l) => l.trim());
  const items: string[] = [];
  const promptLines: string[] = [];

  for (const line of lines) {
    const trimmed = line.trim();
    if (/^[-*]\s/.test(trimmed)) {
      items.push(trimmed.replace(/^[-*]\s*/, "").trim());
    } else {
      promptLines.push(trimmed);
    }
  }

  return { type: "buttons", prompt: promptLines.join("\n"), items };
}

function parseConfirm(raw: string): ConfirmSegment {
  try {
    const parsed = YAML.parse(raw) as Record<string, string>;
    return {
      type: "confirm",
      message: parsed.message || raw.trim(),
      yes: parsed.yes || "Yes",
      no: parsed.no || "No",
    };
  } catch {
    return { type: "confirm", message: raw.trim(), yes: "Yes", no: "No" };
  }
}

function parseChart(raw: string): ChartSegment {
  try {
    const config = YAML.parse(raw) as ChartConfig;
    return { type: "chart", config };
  } catch {
    return { type: "chart", config: { type: "bar", title: "Parse Error" } };
  }
}

function parseCollapse(meta: string, raw: string): CollapseSegment {
  const titleMatch = meta.match(/title="([^"]+)"/);
  return {
    type: "collapse",
    title: titleMatch ? titleMatch[1] : "Details",
    content: raw,
  };
}

function parseSteps(raw: string): StepsSegment {
  const items = raw
    .split("\n")
    .filter((l) => l.trim().startsWith("- ["))
    .map((l) => {
      const done = l.includes("[x]") || l.includes("[X]");
      const text = l.replace(/^[-*]\s*\[[ xX]\]\s*/, "").trim();
      return { done, text };
    });
  return { type: "steps", items };
}
