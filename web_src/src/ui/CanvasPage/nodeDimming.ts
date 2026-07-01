export function isCanvasNodeHighlighted({
  nodeId,
  edgeHoverActive,
  highlightedNodeIds,
  runDimActive,
  runParticipantSet,
}: {
  nodeId: string;
  edgeHoverActive: boolean;
  highlightedNodeIds: Set<string>;
  runDimActive: boolean;
  runParticipantSet: Set<string> | null;
}) {
  if (edgeHoverActive) {
    return highlightedNodeIds.has(nodeId);
  }

  return runDimActive && (runParticipantSet?.has(nodeId) ?? false);
}

export function shouldBlankCanvasNodeBody({
  nodeId,
  edgeHoverActive,
  runDimActive,
  runParticipantSet,
}: {
  nodeId: string;
  edgeHoverActive: boolean;
  runDimActive: boolean;
  runParticipantSet: Set<string> | null;
}) {
  if (edgeHoverActive || !runDimActive) {
    return false;
  }

  return !(runParticipantSet?.has(nodeId) ?? false);
}
