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
export type DraftActionsSegment = { type: "draft-actions"; versionId: string; message?: string };
export type SurveySegment = { type: "survey"; questions: { prompt: string; options: string[]; hasInput?: boolean }[] };
export type RubricCategory = { heading: string; criteria: { text: string }[] };
export type RubricSegment = { type: "rubric"; title: string; criteria: { text: string }[]; categories?: RubricCategory[] };

export type Segment =
  | MarkdownSegment
  | ButtonsSegment
  | ConfirmSegment
  | ChartSegment
  | CollapseSegment
  | MermaidSegment
  | StepsSegment
  | SuccessSegment
  | ErrorSegment
  | DraftActionsSegment
  | SurveySegment
  | RubricSegment;

export type StepItem = { done: boolean; text: string };

export type ChartConfig = {
  type: "line" | "bar" | "area" | "pie";
  title?: string;
  x?: string[];
  series?: { name: string; data: number[]; color?: string }[];
  data?: { name: string; value: number; color?: string }[];
};

// --- Regex patterns ---

const BLOCK_RE = /^\s*:::([\w-]+)(?:\s+(.*))?$/;
const BLOCK_END_RE = /^\s*:::$/;
const MERMAID_FENCE_START = /^\s*```mermaid\s*$/;
const FENCE_END = /^\s*```\s*$/;

// --- Parser state ---

interface ParserState {
  segments: Segment[];
  markdownBuffer: string[];
  blockType: string | null;
  blockMeta: string;
  blockLines: string[];
  inMermaidFence: boolean;
  mermaidLines: string[];
}

function createState(): ParserState {
  return {
    segments: [],
    markdownBuffer: [],
    blockType: null,
    blockMeta: "",
    blockLines: [],
    inMermaidFence: false,
    mermaidLines: [],
  };
}

function flushMarkdown(state: ParserState) {
  const text = state.markdownBuffer.join("\n").trim();
  if (text) {
    state.segments.push({ type: "markdown", content: text });
  }
  state.markdownBuffer = [];
}

function flushBlock(state: ParserState) {
  if (!state.blockType) return;
  const raw = state.blockLines.join("\n");
  const segment = parseBlock(state.blockType, state.blockMeta, raw);
  if (segment) {
    state.segments.push(segment);
  }
  state.blockType = null;
  state.blockMeta = "";
  state.blockLines = [];
}

function processLine(state: ParserState, line: string) {
  if (state.inMermaidFence) {
    if (FENCE_END.test(line.trim())) {
      flushMarkdown(state);
      state.segments.push({ type: "mermaid", content: state.mermaidLines.join("\n") });
      state.mermaidLines = [];
      state.inMermaidFence = false;
    } else {
      state.mermaidLines.push(line);
    }
    return;
  }

  if (state.blockType) {
    if (BLOCK_END_RE.test(line.trim())) {
      flushBlock(state);
    } else {
      state.blockLines.push(line);
    }
    return;
  }

  if (MERMAID_FENCE_START.test(line.trim())) {
    flushMarkdown(state);
    state.inMermaidFence = true;
    state.mermaidLines = [];
    return;
  }

  const match = line.match(BLOCK_RE);
  if (match) {
    flushMarkdown(state);
    state.blockType = match[1];
    state.blockMeta = match[2] || "";
  } else {
    state.markdownBuffer.push(line);
  }
}

// --- Public API ---

export function parseAgentContent(content: string): Segment[] {
  if (!content) return [];

  const state = createState();
  for (const line of content.split("\n")) {
    processLine(state, line);
  }

  if (state.blockType) flushBlock(state);
  flushMarkdown(state);

  return state.segments;
}

// --- Block parsers ---

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
    case "survey":
      return parseSurvey(raw);
    case "rubric":
      return parseRubric(meta, raw);
    case "draft-actions":
      return parseDraftActions(raw, meta);
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
    const parsed = YAML.load(raw) as Record<string, string>;
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
    const config = YAML.load(raw) as ChartConfig;
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

function parseDraftActions(raw: string, meta: string): DraftActionsSegment {
  // Try YAML body first
  try {
    const parsed = YAML.load(raw) as Record<string, unknown>;
    if (parsed && typeof parsed === "object") {
      return {
        type: "draft-actions",
        versionId: String(parsed.versionId ?? parsed.version_id ?? meta.trim()),
        message: parsed.message ? String(parsed.message) : undefined,
      };
    }
  } catch {
    // fall through
  }
  // Fallback: version ID from meta or raw content
  return { type: "draft-actions", versionId: (meta || raw).trim(), message: undefined };
}

function parseSurvey(raw: string): SurveySegment {
  const questions: { prompt: string; options: string[]; hasInput?: boolean }[] = [];
  let currentPrompt = "";
  let currentOptions: string[] = [];
  let hasInput = false;

  for (const line of raw.split("\n")) {
    const trimmed = line.trim();
    if (!trimmed) continue;

    if (/^[-*]\s/.test(trimmed)) {
      const option = trimmed.replace(/^[-*]\s*/, "").trim();
      if (option.toLowerCase() === "[input]" || option.toLowerCase() === "[custom]") {
        hasInput = true;
      } else {
        currentOptions.push(option);
      }
    } else {
      // New question - flush previous
      if (currentPrompt && (currentOptions.length || hasInput)) {
        questions.push({ prompt: currentPrompt, options: currentOptions, hasInput: hasInput || undefined });
      }
      currentPrompt = trimmed;
      currentOptions = [];
      hasInput = false;
    }
  }
  // Flush last question
  if (currentPrompt && (currentOptions.length || hasInput)) {
    questions.push({ prompt: currentPrompt, options: currentOptions, hasInput: hasInput || undefined });
  }

  return { type: "survey", questions };
}

function parseRubric(meta: string, raw: string): RubricSegment {
  const lines = raw.split("\n").filter((l) => l.trim());
  let title = meta.trim();

  // Check if the rubric has category headings (## Heading)
  const hasCategories = lines.some((l) => /^#{2,3}\s/.test(l.trim()));

  if (hasCategories) {
    const categories: RubricCategory[] = [];
    const uncategorized: { text: string }[] = [];
    let currentCategory: RubricCategory | null = null;

    for (const line of lines) {
      const trimmed = line.trim();
      if (/^#{2,3}\s/.test(trimmed)) {
        // New category heading
        if (currentCategory) categories.push(currentCategory);
        currentCategory = { heading: trimmed.replace(/^#{2,3}\s*/, "").trim(), criteria: [] };
      } else if (/^[-*✦•]\s/.test(trimmed)) {
        const criterion = { text: trimmed.replace(/^[-*✦•]\s*/, "").trim() };
        if (currentCategory) currentCategory.criteria.push(criterion);
        else uncategorized.push(criterion);
      } else if (/^\d+[.)]\s/.test(trimmed)) {
        const criterion = { text: trimmed.replace(/^\d+[.)]\s*/, "").trim() };
        if (currentCategory) currentCategory.criteria.push(criterion);
        else uncategorized.push(criterion);
      } else if (!title) {
        title = trimmed;
      }
    }
    if (currentCategory) categories.push(currentCategory);

    // Flatten all criteria for backward compat (used by outcome grading)
    const allCriteria = [...uncategorized, ...categories.flatMap((c) => c.criteria)];
    return { type: "rubric", title: title || "Build Plan", criteria: allCriteria, categories };
  }

  // Flat rubric (no categories)
  const criteria: { text: string }[] = [];
  for (const line of lines) {
    const trimmed = line.trim();
    if (/^[-*✦•]\s/.test(trimmed)) {
      criteria.push({ text: trimmed.replace(/^[-*✦•]\s*/, "").trim() });
    } else if (/^\d+[.)]\s/.test(trimmed)) {
      criteria.push({ text: trimmed.replace(/^\d+[.)]\s*/, "").trim() });
    } else if (!title) {
      title = trimmed;
    } else {
      criteria.push({ text: trimmed });
    }
  }

  return { type: "rubric", title: title || "Build Plan", criteria };
}
