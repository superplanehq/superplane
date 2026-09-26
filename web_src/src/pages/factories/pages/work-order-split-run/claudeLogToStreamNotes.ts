import { parseClaudeCodeLog } from "./parseClaudeCodeLog";
import type { SplitRunStreamLine } from "./splitRunMocks";

/**
 * Turns a runner log into the agent notes the log card renders: one step
 * note per `$ step`, with its tool calls nested under it.
 */
export function claudeLogToStreamNotes(
  nodeId: string,
  text: string,
  configured: Array<{ name: string; type: string }> = [],
): SplitRunStreamLine[] {
  const notes: SplitRunStreamLine[] = [];
  for (const [stepIndex, step] of parseClaudeCodeLog(text, configured).entries()) {
    const stepId = `${nodeId}-step-${stepIndex}`;
    notes.push({
      id: stepId,
      nodeId,
      at: "",
      note: true,
      componentType: step.type || "prompt",
      componentName: step.name,
      status: step.status,
      duration: step.duration,
      detail: step.output,
    });
    for (const [commandIndex, command] of step.commands.entries()) {
      notes.push({
        id: `${stepId}-cmd-${commandIndex}`,
        nodeId,
        at: "",
        note: true,
        noteParentId: stepId,
        noteDepth: 1,
        componentType: command.type,
        componentName: command.name,
        status: command.status,
        detail: command.output,
      });
    }
  }
  return notes;
}
