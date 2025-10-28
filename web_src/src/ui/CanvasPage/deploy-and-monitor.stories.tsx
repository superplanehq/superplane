import type { Meta, StoryObj } from "@storybook/react";
import type { Edge, Node } from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import "./canvas-reset.css";

import dockerIcon from "@/assets/icons/integrations/docker.svg";
import githubIcon from "@/assets/icons/integrations/github.svg";
import KubernetesIcon from "@/assets/icons/integrations/kubernetes.svg";

import { useCallback, useMemo, useState } from "react";
import { CanvasPage, type CanvasNode } from "./index";
import type { BlockData } from "./Block";
import { createGetSidebarData } from "./storybooks/getSidebarData";

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
        parameters: [
          { icon: "database", items: ["db-primary", "db-replica-1", "db-replica-2"] }
        ],
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
        parameters: [
          { icon: "server", items: ["prod-cluster-1", "prod-cluster-2"] }
        ],
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
        parameters: [
          { icon: "map", items: ["us-west-1", "eu-global-1", "asia-east-1"] }
        ],
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
  {
    id: "listen-code",
    position: { x: -500, y: -200 },
    data: {
      label: "Listen to code changes",
      state: "working",
      type: "trigger",
      trigger: {
        title: "GitHub",
        iconSrc: githubIcon,
        iconBackground: "bg-black",
        headerColor: "bg-gray-100",
        collapsedBackground: "bg-black",
        metadata: [
          { icon: "book", label: "monarch-app" },
          { icon: "filter", label: "push" },
        ],
        lastEventData: {
          title: "refactor: update README.md",
          subtitle: "ef53adfa",
          receivedAt: new Date(new Date().getTime() - 1000 * 60 * 45), // 45 minutes ago
          state: "processed",
        },
        collapsed: true,
      },
    },
  },
  {
    id: "listen-image",
    position: { x: -500, y: 200 },
    data: {
      label: "Listen to Docker image updates",
      state: "pending",
      type: "trigger",
      trigger: {
        title: "DockerHub",
        iconSrc: dockerIcon,
        headerColor: "bg-sky-100",
        collapsedBackground: "bg-sky-100",
        metadata: [
          { icon: "box", label: "monarch-app-base-image" },
          { icon: "filter", label: "push" },
        ],
        lastEventData: {
          title: "v3.18.217",
          subtitle: "972.5 MB",
          receivedAt: new Date(new Date().getTime() - 1000 * 60 * 60 * 3), // 3 hours ago
          state: "processed",
        },
        collapsed: true,
      },
    },
  },
  {
    id: "build-stage",
    position: { x: 0, y: 0 },
    data: {
      label: "Build/Test/Deploy to Stage",
      state: "pending",
      type: "composite",
      composite: {
        title: "Build/Test/Deploy Stage",
        description: "Build new release of the monarch app and runs all required tests",
        iconSlug: "git-branch",
        iconColor: "text-purple-700",
        headerColor: "bg-purple-100",
        collapsedBackground: "bg-purple-100",
        parameters: [],
        lastRunItem: {
          title: "fix: open rejected events tabs",
          subtitle: "ef758d40",
          receivedAt: new Date(new Date().getTime() - 1000 * 60 * 60 * 2), // 2 hours ago
          childEventsInfo: {
            count: 3,
            waitingInfos: [],
          },
          state: "failed",
          values: {
            "Author": "Bart Willems",
            "Commit": "FEAT-1234",
            "Sha": "ef758d40",
            "Image": "v3.18.217",
            "Size": "971.5 MB"
          },
        },
        nextInQueue: {
          title: "FEAT-1234: New feature",
          subtitle: "ef758d40",
          receivedAt: new Date(new Date().getTime() - 1000 * 60 * 30), // 30 minutes ago
        },
        collapsed: true
      }
    },
  },
  {
    id: "approve",
    position: { x: 500, y: 0 },
    data: {
      label: "Approve release",
      state: "pending",
      type: "approval",
      approval: {
        title: "Approve Release",
        description: "New releases are deployed to staging for testing and require approvals.",
        iconSlug: "hand",
        iconColor: "text-orange-500",
        headerColor: "bg-orange-100",
        collapsedBackground: "bg-orange-100",
        approvals: [
          {
            title: "Security",
            approved: false,
            interactive: true,
            requireArtifacts: [
              {
                label: "CVE Report",
              }
            ],
            onApprove: (artifacts) => console.log("Security approved with artifacts:", artifacts),
            onReject: (comment) => console.log("Security rejected with comment:", comment),
          },
          {
            title: "Compliance",
            approved: true,
            artifactCount: 1,
            artifacts: {
              "Security Audit Report": "https://example.com/audit-report.pdf",
              "Compliance Certificate": "https://example.com/cert.pdf",
            },
            href: "#",
          },
          {
            title: "Engineering",
            rejected: true,
            approverName: "Lucas Pinheiro",
            rejectionComment: "Security vulnerabilities need to be addressed before approval",
          },
          {
            title: "Josh Brown",
            approved: true,
          },
          {
            title: "Admin",
            approved: false,
          },
        ],
        awaitingEvent: {
          title: "fix: open rejected events tab",
          subtitle: "ef758d40",
        },
        receivedAt: new Date(new Date().getTime() - 1000 * 60 * 60 * 24),
        collapsed: false
      }
    },
  },
  {
    id: "deploy-us",
    position: { x: 1250, y: -600 },
    data: {
      label: "Deploy to US",
      state: "pending",
      type: "composite",
      composite: {
        title: "Deploy to US",
        iconSrc: KubernetesIcon,
        headerColor: "bg-blue-100",
        iconBackground: "bg-blue-500",
        collapsedBackground: "bg-blue-500",
        metadata: [
          { icon: "user", label: "Author: Bart Willems" },
          { icon: "git-commit", label: "Commit: FEAT-1234" },
          { icon: "git-commit", label: "Sha: ef758d40" },
          { icon: "package", label: "Image: v3.18.217" },
          { icon: "package", label: "Size: 971.5 MB" },
        ],
        parameters: [
          { icon: "map", items: ["us-west-1", "us-east-1"] }
        ],
        lastRunItem: {
          title: "FEAT-984: Autocomplete",
          subtitle: "ef758d40",
          receivedAt: new Date(new Date().getTime() - 1000 * 60 * 60 * 5), // 5 hours ago
          childEventsInfo: {
            count: 2,
            state: "processed",
            waitingInfos: [],
          },
          state: "success",
          values: {
            "Author": "Bart Willems",
            "Commit": "FEAT-1234",
            "Sha": "ef758d40",
            "Image": "v3.18.217",
            "Size": "971.5 MB"
          },
        },
        nextInQueue: {
          title: "FEAT-983: Better run names",
          subtitle: "ef758d40",
          receivedAt: new Date(new Date().getTime() - 1000 * 60 * 60), // 1 hour ago
        },
        collapsed: false
      }
    },
  },
  {
    id: "deploy-eu",
    position: { x: 1250, y: 0 },
    data: {
      label: "Deploy to EU",
      state: "pending",
      type: "composite",
      composite: {
        title: "Deploy to EU",
        description: "Deploy your application to the EU region",
        iconSrc: KubernetesIcon,
        headerColor: "bg-blue-100",
        iconBackground: "bg-blue-500",
        collapsedBackground: "bg-blue-500",
        metadata: [
          { icon: "user", label: "Author: Bart Willems" },
          { icon: "git-commit", label: "Commit: FEAT-1234" },
          { icon: "git-commit", label: "Sha: ef758d40" },
          { icon: "package", label: "Image: v3.18.217" },
          { icon: "package", label: "Size: 971.5 MB" },
        ],
        parameters: [
          { icon: "map", items: ["eu-global-1", "eu-global-2"] }
        ],
        lastRunItem: {
          title: "fix: open rejected events",
          subtitle: "ef758d40",
          receivedAt: new Date(new Date().getTime() - 1000 * 60 * 60 * 8), // 8 hours ago
          childEventsInfo: {
            count: 2,
            state: "running",
            waitingInfos: [
              {
                icon: "calendar",
                info: "Wait if it's weekend",
                futureTimeDate: new Date(new Date().getTime() + 200000000),
              },
              {
                icon: "calendar",
                info: "Halloween Holiday",
                futureTimeDate: new Date(new Date().getTime() + 300000000),
              },
            ],
          },
          state: "running",
          values: {
            "Author": "Bart Willems",
            "Commit": "FEAT-1234",
            "Sha": "ef758d40",
            "Image": "v3.18.217",
            "Size": "971.5 MB"
          },
        },
        nextInQueue: {
          title: "Deploy to EU",
          subtitle: "ef758d40",
          receivedAt: new Date(new Date().getTime() - 1000 * 60 * 60 * 4), // 4 hours ago
        },
        collapsed: false
      }
    },
  },
  {
    id: "deploy-asia",
    position: { x: 1250, y: 600 },
    data: {
      label: "Deploy to Asia",
      state: "pending",
      type: "composite",
      composite: {
        title: "Deploy to Asia",
        iconSrc: KubernetesIcon,
        headerColor: "bg-blue-100",
        iconBackground: "bg-blue-500",
        collapsedBackground: "bg-blue-500",
        metadata: [
          { icon: "user", label: "Author: Bart Willems" },
          { icon: "git-commit", label: "Commit: FEAT-1234" },
          { icon: "git-commit", label: "Sha: ef758d40" },
          { icon: "package", label: "Image: v3.18.217" },
          { icon: "package", label: "Size: 971.5 MB" },
        ],
        parameters: [
          { icon: "map", items: ["asia-east-1"] }
        ],
        lastRunItem: {
          title: "fix: open rejected events",
          subtitle: "ef758d40",
          receivedAt: new Date(new Date().getTime() - 1000 * 60 * 60 * 12), // 12 hours ago
          childEventsInfo: {
            count: 1,
            state: "processed",
            waitingInfos: [],
          },
          state: "fail",
          values: {
            "Author": "Bart Willems",
            "Commit": "FEAT-1234",
            "Sha": "ef758d40",
            "Image": "v3.18.217",
            "Size": "971.5 MB"
          },
        },
        startLastValuesOpen: false,
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

export const DeployAndMonitor: Story = {
  args: {
    nodes: sampleNodes,
    edges: sampleEdges,
  },
  render: function DeployAndMonitorRender(args) {
    const [simulationNodes, setSimulationNodes] = useState<CanvasNode[]>(args.nodes ?? []);
    const simulationEdges = useMemo(() => args.edges ?? [], [args.edges]);

    const toggleNodeCollapse = useCallback((nodeId: string) => {
      console.log('toggleNodeCollapse called for nodeId:', nodeId);
      setSimulationNodes(prevNodes => {
        console.log('Current nodes:', prevNodes.length);
        const newNodes = prevNodes.map(node => {
          if (node.id !== nodeId) return node;

          console.log('Found node to toggle:', nodeId, node.data);
          const nodeData = { ...node.data } as unknown as BlockData;

          // Toggle collapse state based on node type
          if (nodeData.type === "composite" && nodeData.composite) {
            console.log('Toggling composite from', nodeData.composite.collapsed, 'to', !nodeData.composite.collapsed);
            nodeData.composite = {
              ...nodeData.composite,
              collapsed: !nodeData.composite.collapsed,
            };
          }

          if (nodeData.type === "approval" && nodeData.approval) {
            console.log('Toggling approval from', nodeData.approval.collapsed, 'to', !nodeData.approval.collapsed);
            nodeData.approval = {
              ...nodeData.approval,
              collapsed: !nodeData.approval.collapsed,
            };
          }

          if (nodeData.type === "trigger" && nodeData.trigger) {
            console.log('Toggling trigger from', nodeData.trigger.collapsed, 'to', !nodeData.trigger.collapsed);
            nodeData.trigger = {
              ...nodeData.trigger,
              collapsed: !nodeData.trigger.collapsed,
            };
          }

          const updatedNode: CanvasNode = { ...node, data: nodeData as unknown as Record<string, unknown> };
          console.log('Updated node:', updatedNode);
          return updatedNode;
        });
        console.log('Returning new nodes:', newNodes.length);
        return newNodes;
      });
    }, []);

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

    const getSidebarData = useMemo(
      () => createGetSidebarData(simulationNodes ?? []),
      [simulationNodes]
    );

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
        <CanvasPage
          {...args}
          nodes={simulationNodes}
          edges={simulationEdges}
          getSidebarData={getSidebarData}
          onRun={(nodeId) => {
            console.log("Run action for node:", nodeId);
          }}
          onDuplicate={(nodeId) => {
            console.log("Duplicate action for node:", nodeId);
          }}
          onDocs={(nodeId) => {
            console.log("Documentation action for node:", nodeId);
          }}
          onToggleView={(nodeId) => {
            console.log("Toggle view action for node:", nodeId);
            console.log("Current nodes before toggle:", simulationNodes.length);
            console.log("Node data before toggle:", simulationNodes.find(n => n.id === nodeId)?.data);
            toggleNodeCollapse(nodeId);
          }}
          onDeactivate={(nodeId) => {
            console.log("Deactivate action for node:", nodeId);
          }}
          onDelete={(nodeId) => {
            console.log("Delete action for node:", nodeId);
          }}
        />
      </div>
    );
  },
};