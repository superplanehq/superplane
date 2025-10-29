import type { Meta, StoryObj } from "@storybook/react";
import type { Edge, Node } from "@xyflow/react";

import "@xyflow/react/dist/style.css";
import "./../canvas-reset.css";

import datadogIcon from "@/assets/icons/integrations/datadog.svg";
import pagerdutyIcon from "@/assets/icons/integrations/pagerduty.svg";

import { useCallback, useMemo, useState } from "react";
import type { BlockData } from "./../Block";
import { CanvasPage, type CanvasNode } from "./../index";
import { createGetSidebarData } from "./getSidebarData";

const meta = {
  title: "Pages/CanvasPage/Examples",
  component: CanvasPage,
  parameters: {
    layout: "fullscreen",
  },
  argTypes: {},
} satisfies Meta<typeof CanvasPage>;

export default meta;

type Story = StoryObj<typeof CanvasPage>;

const incidentResponseNodes: Node[] = [
  {
    id: "datadog-alert-trigger",
    position: { x: 0, y: 0 },
    data: {
      label: "Datadog Alert",
      state: "working",
      type: "trigger",
      trigger: {
        title: "Datadog Alert",
        iconSrc: datadogIcon,
        iconBackground: "bg-purple-600",
        headerColor: "bg-purple-100",
        collapsedBackground: "bg-purple-600",
        metadata: [
          {
            icon: "activity",
            label: "Monitors: Database Latency, API Error Rate",
          },
          { icon: "bell", label: "States: Alert, Recovered" },
          {
            icon: "filter",
            label: "Filters: env == production, service == payments",
          },
        ],
        lastEventData: {
          title: "Database Latency",
          subtitle: "Alert • monitor_id 12345",
          receivedAt: new Date(Date.now() - 1000 * 60 * 60 * 48), // 2 days ago
          state: "processed",
        },
        collapsed: true,
      },
    },
  },
  {
    id: "pagerduty-listener",
    position: { x: 0, y: 300 },
    data: {
      label: "PagerDuty Listener",
      state: "working",
      type: "trigger",
      trigger: {
        title: "PagerDuty",
        iconSrc: pagerdutyIcon,
        iconBackground: "bg-green-600",
        headerColor: "bg-green-100",
        collapsedBackground: "bg-green-600",
        metadata: [
          {
            icon: "bell",
            label: "Events: incident.triggered, incident.resolved",
          },
          {
            icon: "server",
            label: "Services: platform-api, database, auth-service",
          },
          { icon: "filter", label: "Filters: urgency == high, team == sre" },
        ],
        lastEventData: {
          title: "High error rate detected in production",
          subtitle: "incident PAB128 • triggered",
          sizeInMB: 0.08,
          receivedAt: new Date(Date.now() - 1000 * 60 * 60 * 6), // 6 hours ago
          state: "processed",
        },
        collapsed: true,
      },
    },
  },
  {
    id: "sentry-listener",
    position: { x: 0, y: 600 },
    data: {
      label: "Sentry Listener",
      state: "working",
      type: "trigger",
      trigger: {
        title: "Sentry",
        iconSlug: "bug",
        iconColor: "text-blue-700",
        headerColor: "bg-blue-100",
        collapsedBackground: "bg-blue-100",
        metadata: [
          { icon: "folder", label: "Project: checkout-service" },
          { icon: "cloud", label: "Environment: production" },
          {
            icon: "filter",
            label: "Filters: min_events >= 100, min_users >= 50",
          },
        ],
        lastEventData: {
          title: "TypeError: Cannot read property 'status' of undefined",
          subtitle: "release checkout@2025.10.27.4 • 150 users",
          receivedAt: new Date(Date.now() - 1000 * 60 * 25), // 25 minutes ago
          state: "processed",
        },
        collapsed: true,
      },
    },
  },
  {
    id: "prometheus-alert-listener",
    position: { x: 0, y: 900 },
    data: {
      label: "Prometheus Alert Listener",
      state: "working",
      type: "trigger",
      trigger: {
        title: "Prometheus",
        iconSlug: "activity",
        iconColor: "text-red-700",
        headerColor: "bg-red-100",
        collapsedBackground: "bg-red-100",
        metadata: [
          {
            icon: "server",
            label: "Alertmanager: https://alertmanager.prod.internal",
          },
          { icon: "box", label: "Cluster: prod-eu1" },
          { icon: "filter", label: "Rules: HighCPUUsage, PodCrashLoop" },
          { icon: "bell", label: "States: firing, resolved" },
          { icon: "alert-triangle", label: "Severity: critical" },
          { icon: "layers", label: "Namespace: production" },
        ],
        lastEventData: {
          title: "HighCPUUsage",
          subtitle: "critical • 92 percent CPU • node-12",
          sizeInMB: 0.04,
          receivedAt: new Date(Date.now() - 1000 * 60 * 60 * 24 * 5), // 5 days ago
          state: "processed",
        },
        collapsed: true,
      },
    },
  },
  {
    id: "signal-intake",
    position: { x: 650, y: 450 },
    data: {
      label: "Signal Intake",
      state: "working",
      type: "composite",
      composite: {
        title: "Normalize + Dedupe + Classify",
        description:
          "Normalize vendor alerts, suppress duplicates, classify root cause",
        iconSlug: "git-merge",
        iconColor: "text-slate-700",
        headerColor: "bg-slate-200",
        collapsedBackground: "bg-slate-200",
        parameters: [{ icon: "clock", items: ["Dedupe TTL: 10m"] }],
        metadata: [
          {
            icon: "alert-triangle",
            label: "Last issue: TypeError in CheckoutController",
          },
          { icon: "git-commit", label: "Release: checkout@2025.10.27.4" },
          { icon: "activity", label: "Events/min: 220" },
        ],

        lastRunItem: {
          title: "checkout latency spike",
          subtitle: "fingerprint svc:checkout | env:prod | alert:HighLatency",
          receivedAt: new Date(Date.now() - 1000 * 60 * 8), // 8 minutes ago
          childEventsInfo: {
            count: 3,
            state: "success",
            waitingInfos: [
              {
                icon: "clock",
                info: "Dedupe window active",
                futureTimeDate: new Date(Date.now() + 10 * 60 * 1000), // TTL ends in 10m
              },
            ],
          },
          state: "success",
          values: {
            Source: "prometheus",
            Severity: "critical",
            Service: "checkout",
            Env: "prod",
            Release: "app@2025.10.27.4",
          },
        },
        collapsed: false,
      },
    },
  },
  {
    id: "policy-router",
    position: { x: 1250, y: 450 },
    data: {
      label: "Policy Router",
      state: "working",
      type: "switch",
      switch: {
        title: "Route to Policy",
        stages: [
          {
            pathName: "DEPLOY_ERROR",
            field: "$.classifier",
            operator: "contains",
            value: '"recent_deploy"',
            receivedAt: new Date(Date.now() - 1000 * 60 * 30), // 30 minutes ago
            eventState: "success",
            eventTitle:
              "TypeError in CheckoutController after checkout@2025.10.27.4 deploy",
          },
          {
            pathName: "INFRA_FAIL",
            field: "$.alert_type",
            operator: "is",
            value: '"PodCrashLoop"',
            receivedAt: new Date(Date.now() - 1000 * 60 * 60 * 3), // 3 hours ago
            eventState: "success",
            eventTitle: "PodCrashLoopBackOff: payments-service-7d9f8b-xk2mn",
          },
          {
            pathName: "SLO_BREACH",
            field: "$.severity",
            operator: "is",
            value: '"critical"',
            receivedAt: new Date(Date.now() - 1000 * 60 * 8), // 8 minutes ago
            eventState: "success",
            eventTitle: "checkout latency spike - p99: 2.4s (SLO: 500ms)",
          },
          {
            pathName: "DEFAULT",
            field: "$.classifier",
            operator: "is",
            value: '"unknown"',
            receivedAt: new Date(Date.now() - 1000 * 60 * 60 * 24), // 1 day ago
            eventState: "success",
            eventTitle: "Unclassified alert: Memory usage spike in cache layer",
          },
        ],
        collapsed: false,
      },
    },
  },
  // Path 1: DEPLOY_ERROR - Rollback
  {
    id: "rollback-release",
    position: { x: 1800, y: 50 },
    data: {
      label: "Release Rollback",
      state: "working",
      type: "composite",
      composite: {
        title: "Release Rollback",
        description: "Rollback to previous stable release",
        iconSlug: "rotate-ccw",
        iconColor: "text-blue-700",
        headerColor: "bg-blue-100",
        collapsedBackground: "bg-blue-100",
        metadata: [
          { icon: "package", label: "Service: checkout-service" },
          {
            icon: "git-commit",
            label: "Rollback: v2025.10.27.4 → v2025.10.27.3",
          },
          { icon: "check-circle", label: "Status: Completed" },
        ],
        lastRunItem: {
          title: "Rolled back checkout-service",
          subtitle: "v2025.10.27.4 → v2025.10.27.3",
          receivedAt: new Date(Date.now() - 1000 * 60 * 25), // 25 minutes ago
          state: "success",
          values: {
            Service: "checkout-service",
            "Old Version": "v2025.10.27.4",
            "New Version": "v2025.10.27.3",
            Duration: "42s",
            "Pods Restarted": "6",
          },
        },
        collapsed: false,
      },
    },
  },
  // Path 2: INFRA_FAIL - Restart
  {
    id: "restart-workload",
    position: { x: 1800, y: 350 },
    data: {
      label: "Restart Workload",
      state: "working",
      type: "composite",
      composite: {
        title: "Restart Failing Workload",
        description: "Force restart pods and verify health",
        iconSlug: "refresh-cw",
        iconColor: "text-blue-700",
        headerColor: "bg-blue-100",
        collapsedBackground: "bg-blue-100",
        metadata: [
          { icon: "box", label: "Pod: payments-service-7d9f8b-xk2mn" },
          { icon: "layers", label: "Namespace: production" },
          { icon: "check-circle", label: "Status: Healthy" },
        ],
        lastRunItem: {
          title: "Restarted payments-service pod",
          subtitle: "Pod is now healthy and serving traffic",
          receivedAt: new Date(Date.now() - 1000 * 60 * 145), // 2h 25min ago
          state: "success",
          values: {
            "Pod Name": "payments-service-7d9f8b-xk2mn",
            Namespace: "production",
            "Restart Count": "3",
            Status: "Running",
            Health: "Passing",
          },
        },
        collapsed: false,
      },
    },
  },
  // Path 3: SLO_BREACH - Scale&Cache
  {
    id: "scale-and-cache",
    position: { x: 1800, y: 650 },
    data: {
      label: "Scale & Clear Cache",
      state: "working",
      type: "composite",
      composite: {
        title: "Scale Service + Clear Cache",
        description: "Scale up replicas and invalidate cache",
        iconSlug: "zap",
        iconColor: "text-blue-700",
        headerColor: "bg-blue-100",
        collapsedBackground: "bg-blue-100",
        metadata: [
          { icon: "package", label: "Service: checkout-service" },
          { icon: "server", label: "Scaled: 3 → 6 replicas" },
          { icon: "database", label: "Cache: Redis cleared" },
        ],
        lastRunItem: {
          title: "Scaled and cleared cache",
          subtitle: "checkout-service scaled 3→6, Redis cache cleared",
          receivedAt: new Date(Date.now() - 1000 * 60 * 5), // 5 minutes ago
          state: "success",
          values: {
            Service: "checkout-service",
            "Old Replicas": "3",
            "New Replicas": "6",
            "Cache Keys Cleared": "42,150",
            Duration: "18s",
          },
        },
        collapsed: false,
      },
    },
  },
  // Path 4: DEFAULT - Escalate to PagerDuty
  {
    id: "escalate-oncall",
    position: { x: 1800, y: 950 },
    data: {
      label: "Escalate to On-Call",
      state: "working",
      type: "composite",
      composite: {
        title: "Create PagerDuty Incident",
        description: "Escalate alerts to on-call engineer",
        iconSlug: "megaphone",
        iconColor: "text-green-700",
        headerColor: "bg-green-100",
        collapsedBackground: "bg-green-600",
        metadata: [
          { icon: "alert-circle", label: "Service: Platform Operations" },
          { icon: "zap", label: "Urgency: High" },
          { icon: "users", label: "Escalation: SRE On-Call Rotation" },
          { icon: "user", label: "Assigned: Jordan Lee" },
        ],
        lastRunItem: {
          title: "Created incident INC-9421",
          subtitle: "Memory usage spike in cache layer",
          receivedAt: new Date(Date.now() - 1000 * 60 * 60 * 24), // 1 day ago
          state: "success",
          values: {
            "Incident ID": "INC-9421",
            Alert: "Memory usage spike in cache layer",
            Service: "Platform Operations",
            Urgency: "High",
            "Assigned To": "Jordan Lee",
          },
        },
        collapsed: false,
      },
    },
  },
];

const incidentResponseEdges: Edge[] = [
  // Triggers to Signal Intake
  { id: "e1", source: "datadog-alert-trigger", target: "signal-intake" },
  { id: "e2", source: "pagerduty-listener", target: "signal-intake" },
  { id: "e3", source: "sentry-listener", target: "signal-intake" },
  { id: "e4", source: "prometheus-alert-listener", target: "signal-intake" },

  // Signal Intake to Policy Router
  { id: "e5", source: "signal-intake", target: "policy-router" },

  // Policy Router to Actions (Direct connections with sourceHandles)
  {
    id: "e6",
    source: "policy-router",
    sourceHandle: "DEPLOY_ERROR",
    target: "rollback-release",
  },
  {
    id: "e7",
    source: "policy-router",
    sourceHandle: "SLO_BREACH",
    target: "scale-and-cache",
  },
  {
    id: "e8",
    source: "policy-router",
    sourceHandle: "INFRA_FAIL",
    target: "restart-workload",
  },
  {
    id: "e9",
    source: "policy-router",
    sourceHandle: "DEFAULT",
    target: "escalate-oncall",
  },
];

export const IncidentResponse: Story = {
  args: {
    nodes: incidentResponseNodes,
    edges: incidentResponseEdges,
    title: "Incident Response",
  },
  render: (args) => {
    const [nodes, setNodes] = useState<CanvasNode[]>(args.nodes ?? []);

    const toggleNodeCollapse = useCallback((nodeId: string) => {
      console.log("toggleNodeCollapse called for nodeId:", nodeId);
      setNodes((prevNodes) => {
        console.log("Current nodes:", prevNodes.length);
        const newNodes = prevNodes.map((node) => {
          if (node.id !== nodeId) return node;

          console.log("Found node to toggle:", nodeId, node.data);
          const nodeData = { ...node.data } as unknown as BlockData;

          // Toggle collapse state based on node type
          if (nodeData.type === "composite" && nodeData.composite) {
            console.log(
              "Toggling composite from",
              nodeData.composite.collapsed,
              "to",
              !nodeData.composite.collapsed
            );
            nodeData.composite = {
              ...nodeData.composite,
              collapsed: !nodeData.composite.collapsed,
            };
          }

          if (nodeData.type === "approval" && nodeData.approval) {
            console.log(
              "Toggling approval from",
              nodeData.approval.collapsed,
              "to",
              !nodeData.approval.collapsed
            );
            nodeData.approval = {
              ...nodeData.approval,
              collapsed: !nodeData.approval.collapsed,
            };
          }

          if (nodeData.type === "trigger" && nodeData.trigger) {
            console.log(
              "Toggling trigger from",
              nodeData.trigger.collapsed,
              "to",
              !nodeData.trigger.collapsed
            );
            nodeData.trigger = {
              ...nodeData.trigger,
              collapsed: !nodeData.trigger.collapsed,
            };
          }

          const updatedNode: CanvasNode = {
            ...node,
            data: nodeData as unknown as Record<string, unknown>,
          };
          console.log("Updated node:", updatedNode);
          return updatedNode;
        });
        console.log("Returning new nodes:", newNodes.length);
        return newNodes;
      });
    }, []);

    const getSidebarData = useMemo(
      () => createGetSidebarData(nodes ?? []),
      [nodes]
    );

    return (
      <div className="h-[100vh] w-full ">
        <CanvasPage
          {...args}
          nodes={nodes}
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
          onEdit={(nodeId) => {
            console.log("Edit action for node:", nodeId);
          }}
          onToggleView={(nodeId) => {
            console.log("Toggle view action for node:", nodeId);
            console.log("Current nodes before toggle:", nodes.length);
            console.log(
              "Node data before toggle:",
              nodes.find((n) => n.id === nodeId)?.data
            );
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

IncidentResponse.storyName = "04 - Incident Response";
