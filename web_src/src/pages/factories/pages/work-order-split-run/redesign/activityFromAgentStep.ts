import type { AgentActivity, AgentActivityItem, AgentActivityStatus } from "../agentActivity";
import { isThinkingPlaceholder } from "../streamNotesFromLiveLog";
import type { AgentStep } from "./automationsViewModel";

/** Turns a collapsible step into the same activity shape the live turn uses. */
export function activityFromAgentStep(step: AgentStep): AgentActivity | undefined {
  const items: AgentActivityItem[] = [];
  const status = activityStatus(step.status);

  const command = bashCommandFromStep(step);
  if (command) {
    items.push({
      type: "tool",
      id: `${step.id}-output`,
      kind: "bash",
      name: step.title,
      input: command.script,
      output: command.stdout,
      outputStreams: [],
      status,
      truncated: false,
    });
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

function activityStatus(status: string): AgentActivityStatus {
  if (status === "failed") {
    return "failed";
  }
  if (status === "cancelled") {
    return "cancelled";
  }
  if (status === "running") {
    return "running";
  }
  return "passed";
}
