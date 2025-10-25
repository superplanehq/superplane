import { useRef } from "react";
import { CanvasEdge, CanvasNode } from "..";

type SetNodesFn = React.Dispatch<React.SetStateAction<CanvasNode[]>>;

interface SimulationProps {
  nodes: CanvasNode[];
  edges: CanvasEdge[];
  setNodes: SetNodesFn;
}

export type RunSimulationFn = (startNodeId: string) => Promise<void>;

export function useSimulationRunner(props: SimulationProps): RunSimulationFn {
  const engine = useRef<Engine | null>(null);

  return async (startNodeId: string) => {
    if (!engine.current) {
      engine.current = new Engine(props.nodes, props.edges, props.setNodes);
    }

    engine.current.run(startNodeId);
  };
}

export const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));
const noOp = async (input: any) => input;

type CanvasEvent = {
  state: "pending" | "running" | "completed";
  input?: any;
  process?: Promise<void>;
  output?: any;
};

class Engine {
  private queues: Map<string, CanvasEvent[]>;
  private state: "idle" | "running";

  constructor(
    private nodes: CanvasNode[],
    private edges: CanvasEdge[],
    private setNodes: SetNodesFn
  ) {
    this.queues = new Map();
    this.prepareQueues();
    this.state = "idle";
  }

  async run(startNodeId: string) {
    console.log(startNodeId);
    this.addToQueue(startNodeId, { state: "pending" });
    this.processingLoop();
  }

  private async processingLoop() {
    if (this.state === "running") {
      return;
    }

    this.state = "running";
    console.log("Simulation started");

    while (true) {
      this.queues.forEach((_queue, nodeId) => {
        this.processNode(nodeId);
      });

      let activeProcesses = 0;

      for (const [_nodeId, queue] of this.queues.entries()) {
        activeProcesses += queue.length;
      }

      if (activeProcesses === 0) {
        break;
      }

      await sleep(200);
    }

    this.state = "idle";
    console.log("Simulation completed");
  }

  private async processNode(nodeId: string) {
    const node = this.findNodeById(nodeId);
    const run = node.__run || noOp;
    const queue = this.queues.get(nodeId);

    if (!queue) return;
    if (queue.length === 0) return;

    const head = queue[0]!;

    const updateNode = (path: string, value: any) => {
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

    const setOutput = (output: any) => {
      head.output = output;
    };

    if (head.state === "pending") {
      head.process = new Promise<void>(async () => {
        console.log(`Simulation: Running node ${node.id}`);
        head.state = "running";
        await run(head.input, updateNode, setOutput);
        head.state = "completed";
        console.log(`Simulation: Completed node ${node.id}`);
      });

      return;
    }

    if (head.state === "completed") {
      queue.shift();

      const outgoingEdges = this.edges.filter((e) => e.source === nodeId);

      for (const edge of outgoingEdges) {
        this.addToQueue(edge.target, { state: "pending", input: head.output });
      }

      return;
    }
  }

  private prepareQueues() {
    for (const node of this.nodes) {
      this.queues.set(node.id, []);
    }
  }

  private addToQueue(nodeId: string, event: CanvasEvent) {
    const q = this.queues.get(nodeId);

    if (!q) {
      throw new Error(`Queue for node ${nodeId} not found`);
    }

    q.push(event);
  }

  private findNodeById(nodeId: string): CanvasNode {
    const node = this.nodes.find((n) => n.id === nodeId);

    if (!node) {
      throw new Error(`Node with id ${nodeId} not found`);
    }

    return node;
  }
}
