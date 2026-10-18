import { durationLabelMs } from "@/lib/duration";

import type { AgentActivity, AgentActivityItem, AgentActivityStatus } from "../agentActivity";
import { isThinkingPlaceholder } from "../streamNotesFromLiveLog";
import type { AgentStep } from "./automationsViewModel";

/** Turns a collapsible step into the same activity shape the live turn uses. */
export function activityFromAgentStep(step: AgentStep): AgentActivity | undefined {
  const items: AgentActivityItem[] = [];
  const status = activityStatus(step.status);
  const commandStatus = step.type === "prompt" ? activityStatus(step.promptStatus ?? step.status) : status;

  const command = bashCommandFromStep(step);
  if (command) {
    items.push(...bashToolsFromStep(step, command, commandStatus));
  }

  for (const event of step.events) {
    if (event.kind === "note") {
      if (isThinkingPlaceholder(event.text)) {
        continue;
      }
      items.push({
        type: "content",
        id: event.id,
        kind: "assistant",
        text: event.text,
        status: "passed",
        truncated: false,
      });
      continue;
    }
    for (const tool of event.tools) {
      items.push({
        type: "tool",
        id: tool.id,
        kind: tool.type,
        name: tool.type,
        input: tool.name,
        output: tool.output ?? "",
        outputStreams: [],
        status: activityStatus(tool.status),
        truncated: false,
      });
    }
  }

  if (items.length === 0) {
    return undefined;
  }
  return {
    id: step.id,
    provider: "runner",
    status,
    sequence: items.length,
    items,
    truncated: false,
  };
}

/** Scroll follow tick so a growing last paragraph still pins the log. */
export function activityFollowTick(activity?: AgentActivity): string {
  const last = activity?.items.at(-1);
  const growing =
    last?.type === "content" ? last.text.length : last?.type === "tool" ? last.input.length + last.output.length : 0;
  return `${activity?.items.length ?? 0}:${growing}`;
}

/** Joins every live turn so the open step keeps earlier activity. */
export function activityFromTranscript(activities: AgentActivity[]): AgentActivity | undefined {
  const withItems = activities.filter((activity) => activity.items.length > 0);
  if (withItems.length === 0) {
    return undefined;
  }
  const last = withItems[withItems.length - 1];
  if (withItems.length === 1) {
    return last;
  }
  return {
    ...last,
    id: "live-transcript",
    items: withItems.flatMap((activity) => activity.items),
    sequence: last.sequence,
    status: withItems.some((activity) => activity.status === "running") ? "running" : last.status,
  };
}

function bashToolsFromStep(
  step: AgentStep,
  command: { script: string; stdout: string },
  status: AgentActivityStatus,
): AgentActivityItem[] {
  const lines = step.title === "Setup" ? command.script.split(/\r?\n/).filter((line) => line.trim()) : [];
  if (lines.length <= 1) {
    return [bashTool(step, command.script, command.stdout, status, stepDurationMs(step.duration))];
  }
  return lines.map((line, index) =>
    bashTool(
      step,
      line,
      index === lines.length - 1 ? command.stdout : "",
      status,
      index === 0 ? stepDurationMs(step.duration) : undefined,
      index,
    ),
  );
}

function bashTool(
  step: AgentStep,
  script: string,
  stdout: string,
  status: AgentActivityStatus,
  durationMs: number | undefined,
  index?: number,
): AgentActivityItem {
  return {
    type: "tool",
    id: index === undefined ? `${step.id}-output` : `${step.id}-output-${index}`,
    kind: "bash",
    name: step.title,
    input: script,
    output: stdout,
    outputStreams: [],
    status,
    durationMs,
    truncated: false,
  };
}

function bashCommandFromStep(step: AgentStep): { script: string; stdout: string } | undefined {
  const script = step.commandScript?.trim() ? step.commandScript : "";
  if (script) {
    return { script, stdout: step.commandStdout ?? "" };
  }
  if (!step.output?.trim()) {
    return undefined;
  }
  return { script: step.output, stdout: "" };
}

function stepDurationMs(label?: string): number | undefined {
  if (!label?.trim()) {
    return undefined;
  }
  if (/^<\s*1s$/i.test(label.trim())) {
    return 500;
  }
  const ms = durationLabelMs(label);
  return ms > 0 ? ms : undefined;
}

function activityStatus(status: string): AgentActivityStatus {
  if (status === "failed") {
    return "failed";
  }
  if (status === "cancelled") {
    return "cancelled";
  }
  if (status === "stopped") {
    return "interrupted";
  }
  if (status === "running") {
    return "running";
  }
  return "passed";
}
