import type { ClaudeStepGroup } from "./work-order-split-run/PhaseLogCard";
import type { SplitRunStreamLine } from "./work-order-split-run/splitRunMocks";

const PLANNING_SESSION_NOISE_PREFIXES = [
  "Planning session tools enabled",
  "permission mode:",
  "allowed tools:",
  "Claude Code started",
  "planning tools:",
  "mcp errors:",
  "Retrying API",
];

const COLLAPSED_TOOL_TYPES = new Set(["bash", "read", "edit", "write", "tool"]);

const PREAMBLE_ID = "planning-session-preamble";

export function isPlanningSessionNoise(text: string): boolean {
  const line = text.trim();
  if (!line) {
    return false;
  }
  return PLANNING_SESSION_NOISE_PREFIXES.some((prefix) => line === prefix || line.startsWith(prefix));
}

export function isPlanningSessionToolPayload(text: string): boolean {
  const line = text.trim();
  if (!line.startsWith("{")) {
    return false;
  }
  if (/"title"\s*:/.test(line) && /"description"\s*:/.test(line)) {
    return true;
  }
  if (/"message"\s*:/.test(line) || /"text"\s*:/.test(line)) {
    return true;
  }
  try {
    const value = JSON.parse(line) as unknown;
    if (!value || typeof value !== "object" || Array.isArray(value)) {
      return false;
    }
    const record = value as Record<string, unknown>;
    return ["message", "text", "title", "description"].some((key) => typeof record[key] === "string");
  } catch {
    return false;
  }
}

export function groupPlanningSessionLog(notes: SplitRunStreamLine[]): ClaudeStepGroup[] {
  const steps: ClaudeStepGroup[] = [];
  let pendingTools: SplitRunStreamLine[] = [];
  let toolGroup = 0;

  const flushTools = (parent: ClaudeStepGroup) => {
    if (pendingTools.length === 0) {
      return;
    }
    parent.events.push({
      kind: "tools",
      id: `${parent.line.id}-tools-${toolGroup}`,
      tools: pendingTools,
    });
    toolGroup += 1;
    pendingTools = [];
  };

  const ensureStep = (): ClaudeStepGroup => {
    const existing = steps.at(-1);
    if (existing) {
      return existing;
    }
    const preamble: ClaudeStepGroup = {
      line: {
        id: PREAMBLE_ID,
        nodeId: notes[0]?.nodeId,
        at: "",
        note: true,
        componentName: "",
        status: "passed",
      },
      events: [],
    };
    steps.push(preamble);
    return preamble;
  };

  const openHiddenPrompt = (line: SplitRunStreamLine) => {
    const current = steps.at(-1);
    if (current) {
      flushTools(current);
    } else if (pendingTools.length > 0) {
      flushTools(ensureStep());
    }
    toolGroup = 0;
    steps.push({
      line: {
        ...line,
        componentName: "",
        componentType: undefined,
        detail: undefined,
      },
      events: [],
    });
    if (isPlanningSessionUserTalk(line.componentName)) {
      attachNote({
        ...line,
        id: `${line.id}-talk`,
        componentType: "note",
        detail: undefined,
      });
    }
  };

  const attachNote = (line: SplitRunStreamLine) => {
    if (!line.componentName.trim()) {
      return;
    }
    const parent = parentStep(steps, line) ?? ensureStep();
    flushTools(parent);
    parent.events.push({
      kind: "note",
      line: { ...line, componentType: "note", detail: undefined },
    });
  };

  const attachTool = (line: SplitRunStreamLine) => {
    const parent = parentStep(steps, line) ?? ensureStep();
    pendingTools.push({ ...line, componentType: line.componentType || "tool" });
  };

  for (const line of notes) {
    if (isPlanningSessionNoise(line.componentName)) {
      continue;
    }

    if (isPlanningSessionToolPayload(line.componentName) || isCollapsedTool(line)) {
      attachTool(line);
      continue;
    }

    if (!line.noteParentId && line.componentType === "prompt") {
      if (isPlanningSessionPromptStep(line)) {
        openHiddenPrompt(line);
        continue;
      }
      attachNote(line);
      continue;
    }

    const parent = parentStep(steps, line);
    if (parent) {
      attachNote(line);
      continue;
    }

    if (!line.noteParentId) {
      attachNote(line);
    }
  }

  const last = steps.at(-1);
  if (last) {
    flushTools(last);
  } else if (pendingTools.length > 0) {
    flushTools(ensureStep());
  }
  return steps;
}

function isCollapsedTool(line: SplitRunStreamLine): boolean {
  const type = (line.componentType ?? "").toLowerCase();
  return (
    COLLAPSED_TOOL_TYPES.has(type) ||
    type.includes("mcp__") ||
    type.endsWith("propose_draft") ||
    type.endsWith("survey") ||
    type.endsWith("say")
  );
}

function isPlanningSessionPromptStep(line: SplitRunStreamLine): boolean {
  if (line.componentType !== "prompt") {
    return false;
  }
  if (/step-\d+$/.test(line.id)) {
    return true;
  }
  return isPlanningSessionSystemPrompt(line.componentName);
}

function isPlanningSessionUserTalk(text: string): boolean {
  const name = text.trim();
  return Boolean(name) && !isPlanningSessionSystemPrompt(name);
}

function isPlanningSessionSystemPrompt(text: string): boolean {
  const name = text.trim();
  return (
    name === "Plan with the user" ||
    name === "Wait for the next user message" ||
    name.startsWith("You are in a SuperPlane planning session") ||
    name.startsWith("This is a SuperPlane planning session") ||
    name.startsWith("Greet the user") ||
    name.startsWith("The user created the draft task") ||
    name.startsWith("The user skipped that draft") ||
    name.startsWith("The user started refining")
  );
}

function parentStep(steps: ClaudeStepGroup[], line: SplitRunStreamLine): ClaudeStepGroup | undefined {
  if (line.noteParentId) {
    return steps.find((step) => step.line.id === line.noteParentId);
  }
  return steps.at(-1);
}
