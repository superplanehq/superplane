import type { FactoriesFactoryPullRequest, FactoriesWorkOrderArtifact } from "@/api-client";
import { isCommandKind } from "@/lib/agentToolLabels";

import type { SplitRunStreamLine } from "./splitRunMocks";

export function toolCallSummary(tools: Array<{ type?: string; componentType?: string }>): string {
  const kinds = tools.map((tool) => tool.type ?? tool.componentType ?? "");
  const files = kinds.filter((kind) => kind === "read").length;
  const commands = kinds.filter(isCommandKind).length;
  const otherTools = tools.length - files - commands;
  const parts: string[] = [];
  if (files > 0) {
    parts.push(files === 1 ? "Read 1 file" : `Read ${files} files`);
  }
  if (commands > 0) {
    const ran = commands === 1 ? "ran 1 command" : `ran ${commands} commands`;
    parts.push(parts.length === 0 ? ran.charAt(0).toUpperCase() + ran.slice(1) : ran);
  }
  if (otherTools > 0) {
    const used = otherTools === 1 ? "used 1 tool" : `used ${otherTools} tools`;
    parts.push(parts.length === 0 ? used.charAt(0).toUpperCase() + used.slice(1) : used);
  }
  return parts.join(", ");
}

export type ClaudeStepEvent =
  | { kind: "note"; line: SplitRunStreamLine }
  | { kind: "tools"; id: string; tools: SplitRunStreamLine[]; label?: string };

export type ClaudeStepGroup = {
  line: SplitRunStreamLine;
  events: ClaudeStepEvent[];
};

export function groupClaudeSteps(notes: SplitRunStreamLine[]): ClaudeStepGroup[] {
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

  for (const line of notes) {
    if (!line.noteParentId) {
      const current = steps.at(-1);
      if (current) {
        flushTools(current);
      }
      toolGroup = 0;
      steps.push({ line, events: [] });
      continue;
    }
    const parent = steps.find((step) => step.line.id === line.noteParentId);
    if (!parent) {
      continue;
    }
    if (line.componentType === "note") {
      appendStepNote(parent, line, flushTools);
      continue;
    }
    pendingTools.push(line);
  }

  const last = steps.at(-1);
  if (last) {
    flushTools(last);
  }
  return steps;
}

function appendStepNote(
  parent: ClaudeStepGroup,
  line: SplitRunStreamLine,
  flushTools: (parent: ClaudeStepGroup) => void,
) {
  if (!line.componentName.trim()) {
    return;
  }
  flushTools(parent);
  parent.events.push({ kind: "note", line });
}

export type StreamNodeGroup = {
  line: SplitRunStreamLine;
  notes: SplitRunStreamLine[];
  artifact?: FactoriesWorkOrderArtifact;
  pullRequest?: FactoriesFactoryPullRequest;
};

export function groupSplitRunStream(lines: SplitRunStreamLine[]): StreamNodeGroup[] {
  const notesByNode = new Map<string, SplitRunStreamLine[]>();
  for (const line of lines) {
    if (!line.note || !line.nodeId) {
      continue;
    }
    const notes = notesByNode.get(line.nodeId) ?? [];
    notes.push(line);
    notesByNode.set(line.nodeId, notes);
  }

  return lines
    .filter((line) => !line.note && line.action !== "did not run")
    .map((line) => ({
      line,
      notes: notesByNode.get(line.nodeId ?? "") ?? [],
      artifact: line.artifact,
      pullRequest: line.pullRequest,
    }));
}
