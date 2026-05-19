import type { AgentMessage } from "./types";

export type MessageGroup =
  | { type: "message"; message: AgentMessage }
  | { type: "tool-group"; messages: AgentMessage[] }
  | { type: "subagent-group"; messages: AgentMessage[] };

export function groupMessages(messages: AgentMessage[]): MessageGroup[] {
  const groups: MessageGroup[] = [];
  let toolBuffer: AgentMessage[] = [];
  let subagentBuffer: AgentMessage[] = [];

  const flushTools = () => {
    if (toolBuffer.length > 0) {
      groups.push({ type: "tool-group", messages: [...toolBuffer] });
      toolBuffer = [];
    }
  };
  const flushSubagents = () => {
    if (subagentBuffer.length > 0) {
      groups.push({ type: "subagent-group", messages: [...subagentBuffer] });
      subagentBuffer = [];
    }
  };

  for (const message of messages) {
    if (message.role === "tool" && message.toolName?.startsWith("subagent:")) {
      flushTools();
      if (shouldStartNewSubagentGroup(subagentBuffer, message)) {
        flushSubagents();
      }
      subagentBuffer.push(message);
      continue;
    }

    if (message.role === "tool") {
      flushSubagents();
      toolBuffer.push(message);
      continue;
    }

    flushTools();
    flushSubagents();
    groups.push({ type: "message", message });
  }

  flushTools();
  flushSubagents();
  return groups;
}

function shouldStartNewSubagentGroup(buffer: AgentMessage[], message: AgentMessage): boolean {
  if (buffer.length === 0) return false;
  if (buffer[0]?.toolName !== message.toolName) return true;

  const hasStarted = buffer.some((entry) => entry.toolStatus === "started");
  const hasFinished = buffer.some((entry) => entry.toolStatus === "finished");
  return (
    (hasStarted && hasFinished) ||
    (message.toolStatus === "started" && hasStarted) ||
    (message.toolStatus === "finished" && hasFinished)
  );
}
