import { CanvasEdge, CanvasNode } from "..";

type SetNodesFn = React.Dispatch<React.SetStateAction<CanvasNode[]>>;

interface SimulationProps {
  nodes: CanvasNode[];
  edges: CanvasEdge[];
  setNodes: SetNodesFn;
}

type RunSimulationFn = (startNodeId: string) => Promise<void>;

export function useSimulationRunner(props: SimulationProps): RunSimulationFn {
  return async (startNodeId: string) => {
    const engine = new Engine(props.nodes, props.edges, props.setNodes);
    await engine.run(startNodeId);
  };
}

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));
const noOp = async (input: any) => input;

type CanvasEvent = {
  state: "pending" | "running" | "completed";
  process?: Promise<void>;
};

class Engine {
  private queues: Map<string, CanvasEvent[]>;

  constructor(
    private nodes: CanvasNode[],
    private edges: CanvasEdge[],
    private setNodes: SetNodesFn
  ) {
    console.log(this.setNodes);

    this.queues = new Map();
    this.prepareQueues();
  }

  async run(startNodeId: string) {
    this.addToQueue(startNodeId, { state: "pending" });

    await this.processingLoop();
  }

  private async processingLoop() {
    this.queues.forEach((_queue, nodeId) => {
      this.processNode(nodeId);
    });

    while (true) {
      let activeProcesses = 0;

      for (const [_nodeId, queue] of this.queues.entries()) {
        activeProcesses += queue.length;
      }

      if (activeProcesses === 0) {
        break;
      }

      console.log("Engine: processing loop tick", activeProcesses);
      console.log(this.queues);

      await sleep(1000);
    }
  }

  private async processNode(nodeId: string) {
    const node = this.findNodeById(nodeId);
    const run = node.__run || noOp;
    const queue = this.queues.get(nodeId);

    if (!queue) return;
    if (queue.length === 0) return;

    const head = queue[0]!;

    if (head.state === "pending") {
      head.process = new Promise<void>(async () => {
        console.log(`Node ${nodeId}: starting execution`);
        head.state = "running";
        await run("a");
        head.state = "completed";
        console.log(`Node ${nodeId}: completed execution`);
      });

      return;
    }

    if (head.state === "completed") {
      queue.shift();

      const outgoingEdges = this.edges.filter((e) => e.source === nodeId);

      for (const edge of outgoingEdges) {
        this.addToQueue(edge.target, { state: "pending" });
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
