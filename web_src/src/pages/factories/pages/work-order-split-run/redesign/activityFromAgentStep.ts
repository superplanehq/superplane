import type { AgentActivity, AgentActivityItem, AgentActivityStatus } from "../agentActivity";
import { isThinkingPlaceholder } from "../streamNotesFromLiveLog";
import type { AgentStep } from "./automationsViewModel";

/** Turns a collapsible step into the same activity shape the live turn uses. */
export function activityFromAgentStep(step: AgentStep): AgentActivity | undefined {
  const items: AgentActivityItem[] = [];
  const status = activityStatus(step.status);

  if (step.output?.trim() && step.events.length === 0) {
    items.push({
      type: "tool",
      id: `${step.id}-output`,
      kind: "bash",
      name: step.title,
      input: step.output,
      output: "",
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
