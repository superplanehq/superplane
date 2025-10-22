import type { Meta, StoryObj } from "@storybook/react";
import '@xyflow/react/dist/style.css';
import './canvas-reset.css';
import type { Edge, Node } from "@xyflow/react";

import dockerIcon from "@/assets/icons/integrations/docker.svg";
import githubIcon from "@/assets/icons/integrations/github.svg";

import { useCallback, useState } from "react";
import { applyNodeChanges, type NodeChange } from "@xyflow/react";
import { CanvasPage } from "./index";
// Intentionally omit @xyflow/react base CSS to avoid default styles.

const meta = {
  title: "Pages/CanvasPage",
  component: CanvasPage,
  parameters: {
    layout: "fullscreen",
  },
  argTypes: {},
} satisfies Meta<typeof CanvasPage>;

export default meta;

type Story = StoryObj<typeof CanvasPage>;

const sampleNodes: Node[] = [
  {
    id: "listen-code",
    position: { x: 60, y: 80 },
    data: {
      label: "Listen to code changes",
      state: "working",
      type: "trigger",
      trigger: {
        title: "GitHub",
        iconSrc: githubIcon,
        iconBackground: "bg-black",
        headerColor: "bg-gray-100",
        metadata: [
          { icon: "book", label: "monarch-app" },
          { icon: "filter", label: "branch=main" },
        ],
        lastEventData: {
          title: "refactor: update README.md",
          sizeInMB: 1,
          receivedAt: new Date(),
          state: "processed",
        },
      },
    },
  },
  {
    id: "listen-image",
    position: { x: 60, y: 260 },
    data: {
      label: "Listen to Docker image updates",
      state: "pending",
      type: "trigger",
      trigger: {
        title: "DockerHub",
        iconSrc: dockerIcon,
        headerColor: "bg-sky-100",
        metadata: [
          { icon: "box", label: "monarch-app-base-image" },
          { icon: "filter", label: "push" },
        ],
        lastEventData: {
          title: "v3.18.217",
          sizeInMB: 972.5,
          receivedAt: new Date(),
          state: "processed",
        },
      },
    },
  },
  {
    id: "build-stage",
    position: { x: 320, y: 150 },
    data: {
      label: "Build/Test/Deploy to Stage",
      state: "pending",
      type: "composite",
    },
  },
  {
    id: "approve",
    position: { x: 620, y: 150 },
    data: { label: "Approve release", state: "pending", type: "composite" },
  },
  {
    id: "deploy-us",
    position: { x: 940, y: 40 },
    data: { label: "Deploy to US", state: "pending", type: "composite" },
  },
  {
    id: "deploy-eu",
    position: { x: 940, y: 180 },
    data: { label: "Deploy to EU", state: "pending", type: "composite" },
  },
  {
    id: "deploy-asia",
    position: { x: 940, y: 320 },
    data: { label: "Deploy to Asia", state: "pending", type: "composite" },
  },
];

const sampleEdges: Edge[] = [
  { id: "e1", source: "listen-code", target: "build-stage" },
  { id: "e2", source: "listen-image", target: "build-stage" },
  { id: "e3", source: "build-stage", target: "approve" },
  { id: "e4", source: "approve", target: "deploy-us" },
  { id: "e5", source: "approve", target: "deploy-eu" },
  { id: "e6", source: "approve", target: "deploy-asia" },
];

export const SimpleDeployment: Story = {
  args: {
    nodes: sampleNodes,
    edges: sampleEdges,
  },
  render: (args) => {
    const [nodes, setNodes] = useState<Node[]>(args.nodes ?? []);
    const [edges] = useState<Edge[]>(args.edges ?? []);

    const onNodesChange = useCallback((changes: NodeChange[]) => {
      setNodes((nds) => applyNodeChanges(changes, nds));
    }, []);

    const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));

    const runSimulation = useCallback(async () => {
      if (!nodes || nodes.length === 0) return;

      const outgoing = new Map<string, string[]>();
      edges?.forEach((e) => {
        if (!outgoing.has(e.source)) outgoing.set(e.source, []);
        outgoing.get(e.source)!.push(e.target);
      });

      const start = nodes.find((n) => n.type === "input") ?? nodes[0];
      if (!start) return;

      const event = { at: Date.now(), msg: "run" } as const;

      // Walk the graph in topological-ish layers with delays.
      const visited = new Set<string>();
      let frontier: Array<{ id: string; value: unknown }> = [
        { id: start.id, value: event },
      ];

      while (frontier.length) {
        // mark nodes in this layer as working + set lastEvent
        const layerIds = frontier.map((f) => f.id);
        const valuesById = new Map(
          frontier.map((f) => [f.id, f.value] as const)
        );

        setNodes((prev) =>
          prev.map((n) =>
            layerIds.includes(n.id)
              ? {
                  ...n,
                  data: {
                    ...n.data,
                    state: "working",
                    lastEvent: valuesById.get(n.id),
                  },
                }
              : n
          )
        );

        // wait 5 seconds to simulate processing
        await sleep(5000);

        // turn off working state for this layer
        setNodes((prev) =>
          prev.map((n) =>
            layerIds.includes(n.id)
              ? { ...n, data: { ...n.data, state: "pending" } }
              : n
          )
        );

        // build next layer
        const next: Array<{ id: string; value: unknown }> = [];
        frontier.forEach(({ id, value }) => {
          visited.add(id);
          const nexts = outgoing.get(id) ?? [];
          nexts.forEach((nid) => {
            if (!visited.has(nid)) {
              const transformed = { ...((value as any) ?? {}), via: id };
              next.push({ id: nid, value: transformed });
            }
          });
        });

        frontier = next;
      }
    }, [nodes, edges]);

    return (
      <div className="h-[100vh] w-full">
        <div className="absolute z-10 m-2">
          <button
            onClick={runSimulation}
            className="px-3 py-1 rounded bg-blue-600 text-white text-xs shadow hover:bg-blue-700"
          >
            Run
          </button>
        </div>
        <CanvasPage {...args} nodes={nodes} edges={edges} />
      </div>
    );
  },
};
