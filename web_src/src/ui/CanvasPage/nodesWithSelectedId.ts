export function nodesWithSelectedId<T extends { id: string; selected?: boolean }>(
  nodes: T[],
  selectedNodeId: string | null,
): T[] {
  if (selectedNodeId && !nodes.some((node) => node.id === selectedNodeId)) {
    return nodes;
  }

  const alreadyCorrect = nodes.every((node) => Boolean(node.selected) === (node.id === selectedNodeId));
  if (alreadyCorrect) {
    return nodes;
  }

  return nodes.map((node) => ({
    ...node,
    selected: node.id === selectedNodeId,
  }));
}

/** Keep an already-open sidebar node selected without clearing other selection. */
export function selectOpenSidebarNode<T extends { id: string; selected?: boolean }>(
  nodes: T[],
  selectedNodeId: string | null,
): T[] {
  if (!selectedNodeId) {
    return nodes;
  }
  return nodesWithSelectedId(nodes, selectedNodeId);
}
