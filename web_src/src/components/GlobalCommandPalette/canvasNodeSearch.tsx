import type { BuiltInEdge, Node } from "@xyflow/react";
import { useReactFlow } from "@xyflow/react";
import { useEffect, useMemo } from "react";
import { registerCanvasNodeSearchProvider } from "./canvasNodeSearchStore";
import type { CanvasNodeSearchProvider, CanvasNodeSearchResult } from "./canvasNodeSearchStore";

type CanvasNodeSearchBridgeProps = {
  onSearch?: (searchString: string) => Node[];
  onSelectNode?: (node: Node) => void;
};

export function GlobalCommandPaletteCanvasNodeSearch({ onSearch, onSelectNode }: CanvasNodeSearchBridgeProps) {
  const { getNodes, fitView, setNodes } = useReactFlow<Node, BuiltInEdge>();

  const provider = useMemo<CanvasNodeSearchProvider>(
    () => ({
      searchNodes: (query) => {
        const nodes = onSearch ? onSearch(query) : defaultSearchNodes(getNodes(), query);
        return nodes.map(toSearchResult);
      },
      selectNode: (nodeId) => {
        const node = getNodes().find((current) => current.id === nodeId);
        if (!node) return;
        setNodes((nodes) =>
          nodes.map((current) => (current.id === node.id ? { ...current, selected: true } : current)),
        );
        fitView({ nodes: [node], duration: 500 });
        onSelectNode?.(node);
      },
    }),
    [fitView, getNodes, onSearch, onSelectNode, setNodes],
  );

  useEffect(() => {
    return registerCanvasNodeSearchProvider(provider);
  }, [provider]);

  return null;
}

function defaultSearchNodes(nodes: Node[], query: string) {
  const normalizedQuery = query.trim().toLowerCase();
  if (!normalizedQuery) return nodes;

  return nodes.filter((node) => {
    const label = nodeLabel(node).toLowerCase();
    const id = node.id.toLowerCase();
    return label.includes(normalizedQuery) || id.includes(normalizedQuery);
  });
}

function toSearchResult(node: Node): CanvasNodeSearchResult {
  return {
    id: node.id,
    label: displayNodeLabel(node),
    iconSlug: nodeIconSlug(node),
    keywords: [nodeLabel(node), node.id],
  };
}

function displayNodeLabel(node: Node) {
  if ((node.data as { type?: string })?.type === "annotation") return "Note";
  return nodeLabel(node) || node.id;
}

function nodeLabel(node: Node) {
  const data = node.data as { label?: string; nodeName?: string };
  return data.nodeName || data.label || "";
}

function nodeIconSlug(node: Node): string {
  const nodeType = (node.data as { type?: string })?.type;

  if (nodeType === "annotation") return "sticky-note";
  if (nodeType === "component") return dataIconSlug(node, "component", "box");
  if (nodeType === "trigger") return dataIconSlug(node, "trigger", "play");
  if (nodeType === "composite") return dataIconSlug(node, "composite", "boxes");

  return "box";
}

function dataIconSlug(node: Node, key: "component" | "trigger" | "composite", fallback: string) {
  const data = node.data as Record<string, { iconSlug?: string } | undefined>;
  return data[key]?.iconSlug || fallback;
}
