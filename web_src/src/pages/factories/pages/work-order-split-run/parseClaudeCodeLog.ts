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
  duration?: string;
  commands: ClaudeCodeLogCommand[];
};

const HIDDEN_STEP_NAMES = new Set(["Prepare Claude Code", "Fetch task attachments", "Set up GitHub"]);
const COMMAND_DETAIL_MAX = 72;
const STEP_LINE = /^\$ (.+)$/;
const DURATION_LINE = /^~ (.+)$/;
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
    if (current) {
      consumeStepLine(current, rawLine);
    }
  }

  return steps
    .filter((step) => !HIDDEN_STEP_NAMES.has(step.name))
    .map((step) => ({
      name: step.name,
      type: typeForStep(step, configured),
      status: step.status,
      output: step.output,
      duration: step.duration,
      commands: step.commands,
    }));
}

/** Tool calls, `~ duration` lines, and blank lines. Returns true when consumed. */
function consumeStepMetaLine(current: OpenStep, rawLine: string): boolean {
  const tool = rawLine.match(TOOL_LINE);
  if (tool) {
    current.agentStream = true;
    current.commands.push({
      type: tool[1].trim().toLowerCase(),
      name: cleanCommandDetail(tool[2] ?? "", COMMAND_DETAIL_MAX),
      status: "passed",
    });
    return true;
  }
  const duration = rawLine.match(DURATION_LINE)?.[1]?.trim();
  if (duration) {
    current.duration = duration;
    return true;
  }
  return !rawLine.trim();
}

function consumeStepLine(current: OpenStep, rawLine: string) {
  if (consumeStepMetaLine(current, rawLine)) {
    return;
  }
  if (rawLine === "✗ tool failed") {
    const command = lastCommand(current);
    if (command && command.type !== "note") {
      command.status = "failed";
    }
    return;
  }
  if (STEP_FAILED.test(rawLine)) {
    markFailed(current);
    return;
  }
  if (STEP_PASSED.test(rawLine) || RUNNER_NOISE.test(rawLine)) {
    if (/^Claude Code started\b/.test(rawLine)) {
      current.agentStream = true;
    }
    return;
  }

  if (/^\s/.test(rawLine)) {
    appendOutput(lastCommand(current), stripToolIndent(rawLine));
    return;
  }
  if (current.agentStream) {
    current.commands.push({
      type: "note",
      name: cleanCommandDetail(rawLine.trim()),
      status: "passed",
    });
    return;
  }
  appendOutput(current, rawLine.trim());
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

/** Runner checkout roots. Paths under these read as repo-relative. */
const WORKSPACE_ROOT = /^\/home\/ubuntu\/(?:repo|superplane)\//;
const HOME_ROOT = /^\/home\/ubuntu\//;
/** Size the runner appends to a write: `path (2264 chars)`. */
const WRITE_SIZE_SUFFIX = /\s*\(\d+ (?:chars|bytes)\)$/;

function cleanCommandDetail(text: string, max?: number): string {
  const trimmed = text
    .trim()
    .replace(/\s+/g, " ")
    .replace(WRITE_SIZE_SUFFIX, "")
    .replace(WORKSPACE_ROOT, "")
    .replace(HOME_ROOT, "");
  if (max === undefined || trimmed.length <= max) {
    return trimmed;
  }
  const slice = trimmed.slice(0, max - 1);
  const at = slice.lastIndexOf(" ");
  const kept = at > 24 ? slice.slice(0, at) : slice;
  return `${kept}…`;
}
