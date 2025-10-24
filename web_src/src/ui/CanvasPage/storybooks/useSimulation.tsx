import { CanvasEdge, CanvasNode } from "..";

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));

interface SimulationProps {
  nodes: CanvasNode[];
  edges: CanvasEdge[];
  setNodes: React.Dispatch<React.SetStateAction<CanvasNode[]>>;
}

type RunSimulationFn = (startNodeId: string) => Promise<void>;

export function useSimulationRunner(props: SimulationProps): RunSimulationFn {
  return async (startNodeId: string) => {};

  // return useCallback(
  //   async (startNodeId: string) => {
  //     if (!simulationNodes || simulationNodes.length === 0) return;
  //     const outgoing = new Map<string, string[]>();
  //     simulationEdges?.forEach((e) => {
  //       if (!outgoing.has(e.source)) outgoing.set(e.source, []);
  //       outgoing.get(e.source)!.push(e.target);
  //     });
  //     const start =
  //       simulationNodes.find((n) => n.type === "input") ?? simulationNodes[0];
  //     if (!start) return;
  //     const event = { at: Date.now(), msg: "run" } as const;
  //     // Walk the graph in topological-ish layers with delays.
  //     const visited = new Set<string>();
  //     let frontier: Array<{ id: string; value: unknown }> = [
  //       { id: start.id, value: event },
  //     ];
  //     while (frontier.length) {
  //       // mark nodes in this layer as working + set lastEvent
  //       const layerIds = frontier.map((f) => f.id);
  //       const valuesById = new Map(
  //         frontier.map((f) => [f.id, f.value] as const)
  //       );
  //       setSimulationNodes((prev) =>
  //         prev.map((n) =>
  //           layerIds.includes(n.id)
  //             ? {
  //                 ...n,
  //                 data: {
  //                   ...n.data,
  //                   state: "working",
  //                   lastEvent: valuesById.get(n.id),
  //                 },
  //               }
  //             : n
  //         )
  //       );
  //       // wait 5 seconds to simulate processing
  //       await sleep(5000);
  //       // turn off working state for this layer
  //       setSimulationNodes((prev) =>
  //         prev.map((n) =>
  //           layerIds.includes(n.id)
  //             ? { ...n, data: { ...n.data, state: "pending" } }
  //             : n
  //         )
  //       );
  //       // build next layer
  //       const next: Array<{ id: string; value: unknown }> = [];
  //       frontier.forEach(({ id, value }) => {
  //         visited.add(id);
  //         const nexts = outgoing.get(id) ?? [];
  //         nexts.forEach((nid) => {
  //           if (!visited.has(nid)) {
  //             const transformed = {
  //               ...((value as Record<string, unknown>) ?? {}),
  //               via: id,
  //             };
  //             next.push({ id: nid, value: transformed });
  //           }
  //         });
  //       });
  //       frontier = next;
  //     }
  //   },
  //   [simulationNodes, simulationEdges]
  // );
}
