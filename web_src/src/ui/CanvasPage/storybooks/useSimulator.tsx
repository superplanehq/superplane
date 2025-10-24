import { CanvasEdge, CanvasNode } from "..";

export function sleep(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

const noOp = async (input: any) => input;

export function useSimulator(
  nodes: CanvasNode[],
  edges: CanvasEdge[],
  setNodes: (updater: (nodes: CanvasNode[]) => CanvasNode[]) => void
) {
  const updateNode = (id: string, set: (n: CanvasNode) => CanvasNode) => {
    setNodes((nodes) => {
      const r = nodes.map((n) => (n.id === id ? set({ ...n }) : n));
      return r;
    });
  };

  const executeAndPropagate = async (nodeId: string, input: unknown) => {
    const node = nodes.find((n) => n.id === nodeId);
    if (!node) return undefined as unknown;

    console.log(`Executing node ${nodeId}`);

    const run = node._run || noOp;
    const result = await run(input);

    updateNode(nodeId, (n: CanvasNode) => {
      if (n.data.type === "trigger") {
        (n.data.trigger as any).lastEventData = result;
      }

      if (n.data.type === "composite") {
        (n.data.composite as any).lastRunItem = result;
      }

      return n;
    });

    const nextIds = calcOutputNodeIds(nodeId, edges);
    for (const nextId of nextIds) {
      await executeAndPropagate(nextId, result);
    }

    return result;
  };

  const run = async (startNodeId: string) => {
    await executeAndPropagate(startNodeId, {});
  };

  return run;
}

//
// Find all the nodes that are connected to the output of the given node
//
function calcOutputNodeIds(nodeId: string, edges: CanvasEdge[]) {
  const outputEdges = edges.filter((e) => e.source === nodeId);
  const outputNodeIds = outputEdges.map((e) => e.target);
  return outputNodeIds;
}
