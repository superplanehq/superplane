export type ClaudeCodeLogCommand = {
  type: string;
  name: string;
};

export type ClaudeCodeLogStep = {
  name: string;
  type: string;
  commands: ClaudeCodeLogCommand[];
};

const HIDDEN_STEP_NAMES = new Set(["Prepare Claude Code"]);
const COMMAND_DETAIL_MAX = 72;
const NOTE_DETAIL_MAX = 96;
const STEP_LINE = /^\$ (.+)$/;
const TOOL_LINE = /^-> \[([^\]]+)\]\s*(.*)$/;
const RUNNER_NOISE = /^(Claude Code (ready|started)\b|claude=|node=v|cwd=|[✓✗] |Thinking$)/;

type OpenStep = ClaudeCodeLogStep & { agentStream: boolean };

export function parseClaudeCodeLog(
  text: string,
  configured: Array<{ name: string; type: string }> = [],
): ClaudeCodeLogStep[] {
  const steps: ClaudeCodeLogStep[] = [];
  let current: OpenStep | undefined;

  for (const rawLine of text.split(/\r?\n/)) {
    const stepName = rawLine.match(STEP_LINE)?.[1]?.trim();
    if (stepName) {
      current = startStep(steps, stepName, configured);
      continue;
    }
    if (!current) {
      continue;
    }

    const tool = rawLine.match(TOOL_LINE);
    if (tool) {
      current.agentStream = true;
      current.commands.push({
        type: tool[1].trim().toLowerCase(),
        name: cleanCommandDetail(tool[2] ?? "", COMMAND_DETAIL_MAX),
      });
      continue;
    }

    if (!rawLine.trim() || /^\s/.test(rawLine)) {
      continue;
    }
    if (RUNNER_NOISE.test(rawLine)) {
      if (/^Claude Code started\b/.test(rawLine)) {
        current.agentStream = true;
      }
      continue;
    }
    if (!current.agentStream) {
      continue;
    }
    current.commands.push({
      type: "note",
      name: cleanCommandDetail(rawLine.trim(), NOTE_DETAIL_MAX),
    });
  }

  return steps
    .filter((step) => !HIDDEN_STEP_NAMES.has(step.name))
    .map((step) => ({ name: step.name, type: typeForStep(step, configured), commands: step.commands }));
}

function startStep(
  steps: ClaudeCodeLogStep[],
  name: string,
  configured: Array<{ name: string; type: string }>,
): OpenStep | undefined {
  if (HIDDEN_STEP_NAMES.has(name)) {
    return undefined;
  }
  const step: OpenStep = {
    name,
    type: "",
    commands: [],
    agentStream: configured.some((entry) => entry.name === name && entry.type === "prompt"),
  };
  steps.push(step);
  return step;
}

function typeForStep(step: ClaudeCodeLogStep, configured: Array<{ name: string; type: string }>): string {
  const match = configured.find((entry) => entry.name === step.name);
  if (match?.type) {
    return match.type;
  }
  return step.commands.some((command) => command.type !== "note") ? "prompt" : "bash";
}

function cleanCommandDetail(text: string, max = COMMAND_DETAIL_MAX): string {
  const trimmed = text
    .trim()
    .replace(/\s+/g, " ")
    .replace(/^\/home\/ubuntu\/superplane\//, "")
    .replace(/^\/home\/ubuntu\//, "");
  if (trimmed.length <= max) {
    return trimmed;
  }
  const slice = trimmed.slice(0, max - 1);
  const at = slice.lastIndexOf(" ");
  const kept = at > 24 ? slice.slice(0, at) : slice;
  return `${kept}…`;
}
