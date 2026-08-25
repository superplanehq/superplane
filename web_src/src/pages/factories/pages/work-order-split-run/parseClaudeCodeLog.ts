export type ClaudeCodeLogStatus = "passed" | "failed";

export type ClaudeCodeLogCommand = {
  type: string;
  name: string;
  status: ClaudeCodeLogStatus;
  output?: string;
};

export type ClaudeCodeLogStep = {
  name: string;
  type: string;
  status: ClaudeCodeLogStatus;
  output?: string;
  commands: ClaudeCodeLogCommand[];
};

const HIDDEN_STEP_NAMES = new Set(["Prepare Claude Code"]);
const COMMAND_DETAIL_MAX = 72;
const STEP_LINE = /^\$ (.+)$/;
const TOOL_LINE = /^-> \[([^\]]+)\]\s*(.*)$/;
const RUNNER_NOISE = /^(Claude Code (ready|started)\b|claude=|node=v|cwd=|Thinking$)/;
const STEP_PASSED = /^✓ /;
const STEP_FAILED = /^✗ /;

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
        status: "passed",
      });
      continue;
    }

    if (!rawLine.trim()) {
      continue;
    }
    if (STEP_FAILED.test(rawLine)) {
      markFailed(current);
      continue;
    }
    if (STEP_PASSED.test(rawLine) || RUNNER_NOISE.test(rawLine)) {
      if (/^Claude Code started\b/.test(rawLine)) {
        current.agentStream = true;
      }
      continue;
    }

    if (/^\s/.test(rawLine)) {
      appendOutput(lastCommand(current), stripToolIndent(rawLine));
      continue;
    }
    if (current.agentStream) {
      current.commands.push({
        type: "note",
        name: cleanCommandDetail(rawLine.trim()),
        status: "passed",
      });
      continue;
    }
    appendOutput(current, rawLine.trim());
  }

  return steps
    .filter((step) => !HIDDEN_STEP_NAMES.has(step.name))
    .map((step) => ({
      name: step.name,
      type: typeForStep(step, configured),
      status: step.status,
      output: step.output,
      commands: step.commands,
    }));
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
    status: "passed",
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

function lastCommand(step: OpenStep): ClaudeCodeLogCommand | undefined {
  return step.commands.at(-1);
}

function markFailed(step: OpenStep) {
  step.status = "failed";
  const command = lastCommand(step);
  if (command && command.type !== "note") {
    command.status = "failed";
  }
}

function appendOutput(target: { output?: string } | undefined, line: string) {
  if (!target || !line) {
    return;
  }
  target.output = target.output ? `${target.output}\n${line}` : line;
}

function stripToolIndent(line: string): string {
  return line.replace(/^ {5}/, "").replace(/^ {4}/, "");
}

function cleanCommandDetail(text: string, max?: number): string {
  const trimmed = text
    .trim()
    .replace(/\s+/g, " ")
    .replace(/^\/home\/ubuntu\/superplane\//, "")
    .replace(/^\/home\/ubuntu\//, "");
  if (max === undefined || trimmed.length <= max) {
    return trimmed;
  }
  const slice = trimmed.slice(0, max - 1);
  const at = slice.lastIndexOf(" ");
  const kept = at > 24 ? slice.slice(0, at) : slice;
  return `${kept}…`;
}
