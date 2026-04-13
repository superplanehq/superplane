import type { CanvasOperation } from "@/lib/ai/types";

/**
 * Diagram generation guidelines:
 *
 * Layout
 *   - Adaptive direction: LR for 1-4 nodes, TD for 5+.
 *
 * Node shapes
 *   - Triggers use stadium shape:  id(["label"])
 *   - Components use rectangles:   id["label"]
 *   - Trigger detection: blockName matches *.on*, or is exactly "start".
 *
 * Node styles (diff semantics)
 *   - Added nodes:    solid blue border          (:::added)
 *   - Modified nodes: dashed amber border         (:::updated)
 *   - Deleted nodes:  dashed red border, faded    (:::deleted)
 *
 * Edge styles (diff semantics)
 *   - New connections:      solid arrow   -->
 *   - Removed connections:  dotted arrow  -.->  with "removed" class
 *
 * Labels
 *   - Preview mode:  name only (compact for sidebar).
 *   - Expanded mode: name + blockName on second line, edge channel labels.
 */

export type DiagramMode = "preview" | "expanded";

function sanitizeLabel(text: string): string {
  return text.replace(/"/g, "#quot;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
}

function sanitizeId(key: string): string {
  return key.replace(/[^a-zA-Z0-9_]/g, "_");
}

function isTriggerBlock(blockName: string): boolean {
  if (blockName === "start") return true;
  const parts = blockName.split(".");
  return parts.length >= 2 && /^on[A-Z]/.test(parts[parts.length - 1]);
}

function resolveKey(ref?: { nodeKey?: string; nodeId?: string; nodeName?: string }): string {
  return ref?.nodeKey || ref?.nodeId || ref?.nodeName || "unknown";
}

export function proposalToMermaid(operations: CanvasOperation[], mode: DiagramMode = "preview"): string {
  const nodes = new Map<string, { name: string; block: string; isTrigger: boolean }>();
  const addedEdges: { from: string; to: string; channel: string }[] = [];
  const removedEdges: { from: string; to: string; channel: string }[] = [];
  const deletedKeys = new Set<string>();
  const updatedKeys = new Set<string>();

  for (const op of operations) {
    switch (op.type) {
      case "add_node": {
        const key = op.nodeKey || op.blockName;
        nodes.set(key, {
          name: op.nodeName || op.blockName,
          block: op.blockName,
          isTrigger: isTriggerBlock(op.blockName),
        });
        break;
      }
      case "connect_nodes": {
        addedEdges.push({
          from: resolveKey(op.source),
          to: resolveKey(op.target),
          channel: op.source.handleId?.trim() || "",
        });
        break;
      }
      case "disconnect_nodes": {
        removedEdges.push({
          from: resolveKey(op.source),
          to: resolveKey(op.target),
          channel: op.source.handleId?.trim() || "",
        });
        break;
      }
      case "delete_node": {
        const key = resolveKey(op.target);
        deletedKeys.add(key);
        break;
      }
      case "update_node_config": {
        const key = resolveKey(op.target);
        updatedKeys.add(key);
        if (op.nodeName && !nodes.has(key)) {
          nodes.set(key, { name: op.nodeName, block: "", isTrigger: false });
        }
        break;
      }
    }
  }

  const direction = nodes.size > 4 ? "TD" : "LR";
  const lines: string[] = [`graph ${direction}`];

  for (const [key, { name, block, isTrigger }] of nodes) {
    const id = sanitizeId(key);
    let label: string;
    if (mode === "expanded" && block) {
      label = `${sanitizeLabel(name)}<br/><small>${sanitizeLabel(block)}</small>`;
    } else {
      label = sanitizeLabel(name);
    }

    const shape = isTrigger ? `(["${label}"])` : `["${label}"]`;
    let nodeClass: string;
    if (deletedKeys.has(key)) {
      nodeClass = ":::deleted";
    } else if (updatedKeys.has(key)) {
      nodeClass = ":::updated";
    } else {
      nodeClass = ":::added";
    }
    lines.push(`  ${id}${shape}${nodeClass}`);
  }

  for (const { from, to, channel } of addedEdges) {
    const fromId = sanitizeId(from);
    const toId = sanitizeId(to);
    if (mode === "expanded" && channel) {
      lines.push(`  ${fromId} -->|"${sanitizeLabel(channel)}"| ${toId}`);
    } else {
      lines.push(`  ${fromId} --> ${toId}`);
    }
  }

  for (const { from, to, channel } of removedEdges) {
    const fromId = sanitizeId(from);
    const toId = sanitizeId(to);
    if (mode === "expanded" && channel) {
      lines.push(`  ${fromId} -.->|"${sanitizeLabel(channel)}"| ${toId}:::removedEdge`);
    } else {
      lines.push(`  ${fromId} -.-> ${toId}`);
    }
  }

  lines.push("  classDef added stroke:#2563eb,stroke-width:2px");
  lines.push("  classDef updated stroke:#d97706,stroke-width:2px,stroke-dasharray:5 5");
  lines.push("  classDef deleted stroke:#dc2626,stroke-width:2px,stroke-dasharray:3 3,opacity:0.6");

  return lines.join("\n");
}

export type DiffItem = {
  label: string;
  blockName: string;
};

export type ProposalDiffSummary = {
  addedNodes: DiffItem[];
  modifiedNodes: DiffItem[];
  removedNodes: DiffItem[];
  addedConnections: DiffItem[];
  removedConnections: DiffItem[];
};

export function summarizeProposalDiff(operations: CanvasOperation[]): ProposalDiffSummary {
  const addedNodes: DiffItem[] = [];
  const modifiedNodes: DiffItem[] = [];
  const removedNodes: DiffItem[] = [];
  const addedConnections: DiffItem[] = [];
  const removedConnections: DiffItem[] = [];

  const nodeInfo = new Map<string, { label: string; blockName: string }>();
  for (const op of operations) {
    if (op.type === "add_node") {
      const key = op.nodeKey || op.blockName;
      nodeInfo.set(key, { label: op.nodeName || op.blockName, blockName: op.blockName });
    }
  }

  const resolveBlockName = (ref?: { nodeKey?: string; nodeId?: string; nodeName?: string }) => {
    if (!ref) return "";
    const key = ref.nodeKey || ref.nodeId || ref.nodeName || "";
    return nodeInfo.get(key)?.blockName || "";
  };

  const resolveLabel = (ref?: { nodeKey?: string; nodeId?: string; nodeName?: string }) => {
    if (!ref) return "node";
    if (ref.nodeName) return ref.nodeName;
    const key = ref.nodeKey || ref.nodeId || "";
    return nodeInfo.get(key)?.label || ref.nodeKey || ref.nodeId || "node";
  };

  for (const op of operations) {
    switch (op.type) {
      case "add_node":
        addedNodes.push({ label: op.nodeName || op.blockName, blockName: op.blockName });
        break;
      case "update_node_config":
        modifiedNodes.push({
          label: op.nodeName || resolveLabel(op.target),
          blockName: resolveBlockName(op.target),
        });
        break;
      case "delete_node":
        removedNodes.push({
          label: resolveLabel(op.target),
          blockName: resolveBlockName(op.target),
        });
        break;
      case "connect_nodes":
        addedConnections.push({
          label: `${resolveLabel(op.source)} → ${resolveLabel(op.target)}`,
          blockName: "",
        });
        break;
      case "disconnect_nodes":
        removedConnections.push({
          label: `${resolveLabel(op.source)} → ${resolveLabel(op.target)}`,
          blockName: "",
        });
        break;
    }
  }

  return { addedNodes, modifiedNodes, removedNodes, addedConnections, removedConnections };
}
