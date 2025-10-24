import type { Meta, StoryObj } from "@storybook/react";
import type { Edge, Node } from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import "./canvas-reset.css";

import dockerIcon from "@/assets/icons/integrations/docker.svg";
import githubIcon from "@/assets/icons/integrations/github.svg";
import KubernetesIcon from "@/assets/icons/integrations/kubernetes.svg";

import { useCallback, useMemo, useState } from "react";
import { CanvasPage } from "./index";

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
    id: "db-health",
    position: { x: 500, y: -800 },
    data: {
      label: "Database Health Monitor",
      state: "pending",
      type: "composite",
      composite: {
        title: "Database Health Monitor",
        description: "",
        iconSlug: "database",
        iconColor: "text-green-700",
        headerColor: "bg-green-100",
        collapsedBackground: "bg-green-100",
        metadata: [
          { icon: "check-circle", label: "Connection: Healthy" },
          { icon: "clock", label: "Replication Lag: 45ms" },
          { icon: "zap", label: "Query Time: 12ms avg" },
          { icon: "activity", label: "Pool: 45/100 connections" },
        ],
        parameters: ["db-primary", "db-replica-1", "db-replica-2"],
        parametersIcon: "database",
        lastRunItem: {
          title: "Database health check",
          subtitle: "45ms lag",
          receivedAt: new Date(new Date().getTime() - 1000 * 60 * 6), // 6 minutes ago
          childEventsInfo: {
            count: 3,
            state: "processed",
            waitingInfos: [],
          },
          state: "success",
          values: {
            "Connection": "Healthy",
            "Replication Lag": "45ms",
            "Avg Query Time": "12ms",
            "Pool Usage": "45/100",
          },
        },
        collapsed: false
      }
    },
  },
  {
    id: "infra-monitor",
    position: { x: 0, y: -800 },
    data: {
      label: "Infrastructure Resource Monitor",
      state: "pending",
      type: "composite",
      composite: {
        title: "Infrastructure Resource Monitor",
        description: "",
        iconSlug: "cpu",
        iconColor: "text-green-700",
        headerColor: "bg-green-100",
        collapsedBackground: "bg-green-100",
        metadata: [
          { icon: "cpu", label: "CPU: 45%" },
          { icon: "hard-drive", label: "Memory: 12.3 GB available" },
          { icon: "hard-drive", label: "Disk: 85% used" },
          { icon: "box", label: "Pods: 11/24 healthy" },
        ],
        parameters: ["prod-cluster-1", "prod-cluster-2"],
        parametersIcon: "server",
        lastRunItem: {
          title: "Resource check",
          subtitle: "11/24 pods",
          receivedAt: new Date(new Date().getTime() - 1000 * 60 * 8), // 8 minutes ago
          childEventsInfo: {
            count: 2,
            state: "processed",
            waitingInfos: [],
          },
          state: "failure",
          values: {
            "CPU Usage": "45%",
            "Memory": "12.3 GB available",
            "Disk": "85% used",
            "Pods": "24/24",
          },
        },
        collapsed: false
      }
    },
  },
  {
    id: "deploy-test",
    position: { x: -500, y: -800 },
    data: {
      label: "Traffic / Load Monitor",
      state: "pending",
      type: "composite",
      composite: {
        title: "Traffic / Load Monitor",
        description: "",
        iconSlug: "trending-up",
        iconColor: "text-green-700",
        headerColor: "bg-green-100",
        collapsedBackground: "bg-green-100",
        metadata: [
          { icon: "activity", label: "Requests/sec: 1,247 req/s" },
          { icon: "users", label: "Active Connections: 3,842" },
          { icon: "alert-circle", label: "Error Rate: 0.3%" },
          { icon: "server", label: "Load Balancer: Healthy" },
        ],
        parameters: ["us-west-1", "eu-global-1", "asia-east-1"],
        parametersIcon: "map",
        lastRunItem: {
          title: "Traffic monitoring check",
          subtitle: "1,247 req/s",
          receivedAt: new Date(new Date().getTime() - 1000 * 60 * 15), // 15 minutes ago
          childEventsInfo: {
            count: 3,
            state: "processed",
            waitingInfos: [],
          },
          state: "success",
          values: {
            "Requests/sec": "1,247",
            "Connections": "3,842",
            "Error Rate": "0.3%",
            "Status": "Healthy",
          },
        },
        collapsed: false
      }
    },
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

export const Monitor: Story = {
  args: {
    nodes: sampleNodes,
    edges: sampleEdges,
  },
  render: function MonitorRender(args) {
    const [simulationNodes, setSimulationNodes] = useState<Node[]>(args.nodes ?? []);
    const simulationEdges = useMemo(() => args.edges ?? [], [args.edges]);

    const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));

    const runSimulation = useCallback(async () => {
      if (!simulationNodes || simulationNodes.length === 0) return;

      const outgoing = new Map<string, string[]>();
      simulationEdges?.forEach((e) => {
        if (!outgoing.has(e.source)) outgoing.set(e.source, []);
        outgoing.get(e.source)!.push(e.target);
      });

      const start = simulationNodes.find((n) => n.type === "input") ?? simulationNodes[0];
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

        setSimulationNodes((prev) =>
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
        setSimulationNodes((prev) =>
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
              const transformed = { ...(value as Record<string, unknown> ?? {}), via: id };
              next.push({ id: nid, value: transformed });
            }
          });
        });

        frontier = next;
      }
    }, [simulationNodes, simulationEdges]);

    return (
      <div className="h-[100vh] w-full ">
        <div className="absolute z-10 m-2">
          <button
            onClick={runSimulation}
            className="px-3 py-1 rounded bg-blue-600 text-white text-xs shadow hover:bg-blue-700"
          >
            Run
          </button>
        </div>
        <CanvasPage {...args} nodes={simulationNodes} edges={simulationEdges} />
      </div>
    );
  },
};