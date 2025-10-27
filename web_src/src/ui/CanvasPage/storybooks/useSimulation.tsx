import { useMemo } from "react";
import { CanvasEdge, CanvasNode } from "..";

export type UpdateDataFn = (path: string, data: any) => void;
export type OutputFn = (data: any) => void;

export interface Simulation {
  // Function that runs when there is any change to the queue
  onQueueChange?: (next: any, update: UpdateDataFn) => void;

  // Function that runs when the node is executed in a simulation
  run: (input: any, update: UpdateDataFn, output: OutputFn) => Promise<void>;
}

type SetNodesFn = React.Dispatch<React.SetStateAction<CanvasNode[]>>;

interface SimulationProps {
  nodes: CanvasNode[];
  edges: CanvasEdge[];
  setNodes: SetNodesFn;
}

export type RunSimulationFn = (startNodeId: string) => Promise<void>;

export function useSimulationRunner(props: SimulationProps): SimulationEngine {
  return useMemo(
    () => new SimulationEngine(props.nodes, props.edges, props.setNodes),
    []
  );
}

export const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));
const noOp = async (input: any) => input;

type CanvasEvent = {
  input?: any;
  process?: Promise<void>;
  output?: any;
};

type Queue = {
  events: CanvasEvent[];
  active: CanvasEvent | null;
  state: "idle" | "running";
};

export class SimulationEngine {
  private queues: Map<string, Queue>;

  constructor(
    private nodes: CanvasNode[],
    private edges: CanvasEdge[],
    private setNodes: SetNodesFn
  ) {
    this.queues = new Map();
    this.prepareQueues();
    this.processingLoop();
  }

  async run(startNodeId: string) {
    this.addToQueue(startNodeId, {});
  }

  private async processingLoop() {
    while (true) {
      this.queues.forEach((_queue, nodeId) => {
        this.processNode(nodeId);
      });

      await sleep(200);
    }
  }

  private async processNode(nodeId: string) {
    const node = this.findNodeById(nodeId);
    const run = node.__simulation?.run || noOp;
    const queue = this.queues.get(nodeId);

    if (!queue) return;
    if (queue.events.length === 0) return;
    if (queue.state === "running") return;

    const event = queue.events.shift()!;
    this.sendOnQueueChange(nodeId);

    const updateNode = this.updateNodeFn(nodeId);

    event.process = new Promise<void>(async () => {
      console.log(`Simulation: Running node ${node.id}`);
      queue.state = "running";

      await run(
        event.input,
        updateNode,
        (output: any) => (event.output = output)
      );

      queue.state = "idle";

      const outgoingEdges = this.edges.filter((e) => e.source === nodeId);

      for (const edge of outgoingEdges) {
        this.addToQueue(edge.target, { input: event.output });
      }

      console.log(`Simulation: Completed node ${node.id}`);
    });
  }

  private updateNodeFn(nodeId: string): UpdateDataFn {
    const node = this.findNodeById(nodeId);

    return (path: string, value: any) => {
      this.setNodes((prevNodes) =>
        prevNodes.map((n) => {
          if (n.id !== node.id) return n;

          const pathParts = path.split(".");
          const updatedNode = { ...n };
          let current: any = updatedNode;

          for (let i = 0; i < pathParts.length - 1; i++) {
            current[pathParts[i]] = { ...current[pathParts[i]] };
            current = current[pathParts[i]];
          }

          current[pathParts[pathParts.length - 1]] = value;
          return updatedNode;
        })
      );
    };
  }

  private prepareQueues() {
    for (const node of this.nodes) {
      this.queues.set(node.id, {
        events: [],
        active: null,
        state: "idle",
      });
    }
  }

  private addToQueue(nodeId: string, event: CanvasEvent) {
    const q = this.queues.get(nodeId);

    if (!q) {
      throw new Error(`Queue for node ${nodeId} not found`);
    }

    q.events.push(event);
    this.sendOnQueueChange(nodeId);
  }

  private sendOnQueueChange(nodeId: string) {
    const node = this.findNodeById(nodeId);
    const onQueueChange = node.__simulation?.onQueueChange;
    const updatedNode = this.updateNodeFn(nodeId);

    if (!onQueueChange) return;

    const queue = this.queues.get(nodeId);
    if (!queue) return;

    const next = queue.events[0] || null;
    onQueueChange(next?.input, updatedNode);
  }

  private findNodeById(nodeId: string): CanvasNode {
    const node = this.nodes.find((n) => n.id === nodeId);

    if (!node) {
      throw new Error(`Node with id ${nodeId} not found`);
    }

    return node;
  }
}
