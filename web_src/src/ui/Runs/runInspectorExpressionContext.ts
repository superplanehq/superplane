import type { RunInspectorNodeSection } from "./runNodeDetailModel";

export function buildRuntimeExpressionContext(section: RunInspectorNodeSection): Record<string, unknown> | null {
  if (section.upstreamSections.length === 0) return null;

  const context: Record<string, unknown> = {};
  const nodeNames: Record<string, unknown> = {};

  section.upstreamSections.forEach((upstreamSection) => {
    const output = upstreamSection.output ?? null;
    context[upstreamSection.nodeName] = output;
    context[upstreamSection.nodeId] = output;

    const metadata = {
      nodeName: upstreamSection.nodeName,
      componentType: upstreamSection.workflowNode?.component,
    };
    nodeNames[upstreamSection.nodeName] = metadata;
    nodeNames[upstreamSection.nodeId] = metadata;
  });

  context.__root = section.upstreamSections[0]?.output ?? null;
  context.__previousByDepth = buildPreviousByDepth(section);
  context.__nodeNames = nodeNames;

  return context;
}

function buildPreviousByDepth(section: RunInspectorNodeSection): Record<string, unknown> {
  const previousByDepth: Record<string, unknown> = {};
  const reversedUpstreamSections = [...section.upstreamSections].reverse();

  reversedUpstreamSections.forEach((upstreamSection, index) => {
    previousByDepth[String(index + 1)] = upstreamSection.output ?? null;
  });

  return previousByDepth;
}
