import type { Meta, StoryObj } from "@storybook/react";
import type { Edge, Node } from "@xyflow/react";

import "@xyflow/react/dist/style.css";
import "./../canvas-reset.css";

import datadogIcon from "@/assets/icons/integrations/datadog.svg";
import pagerdutyIcon from "@/assets/icons/integrations/pagerduty.svg";

import { CanvasPage } from "./../index";

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
          { icon: "activity", label: "Monitors: Database Latency, API Error Rate" },
          { icon: "bell", label: "States: Alert, Recovered" },
          { icon: "filter", label: "Filters: env == production, service == payments" }
        ],
        lastEventData: {
          title: "Database Latency",
          subtitle: "Alert • monitor_id 12345",
          receivedAt: new Date("2025-10-27T10:15:00Z"),
          state: "processed",
        },
        collapsed: false,
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
          { icon: "bell", label: "Events: incident.triggered, incident.resolved" },
          { icon: "server", label: "Services: platform-api, database, auth-service" },
          { icon: "filter", label: "Filters: urgency == high, team == sre" },
        ],
        lastEventData: {
          title: "High error rate detected in production",
          subtitle: "incident PAB128 • triggered",
          sizeInMB: 0.08,
          receivedAt: new Date("2025-10-27T11:20:00Z"),
          state: "processed",
        },
        collapsed: false,
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
          { icon: "filter", label: "Filters: min_events >= 100, min_users >= 50" },
        ],
        lastEventData: {
          title: "TypeError: Cannot read property 'status' of undefined",
          subtitle: "release checkout@2025.10.27.4 • 150 users",
          receivedAt: new Date(new Date().getTime() - 1000 * 60 * 5), // 5 minutes ago
          state: "processed",
        },
        collapsed: false,
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
          { icon: "server", label: "Alertmanager: https://alertmanager.prod.internal" },
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
          receivedAt: new Date("2025-10-27T11:45:00Z"),
          state: "processed",
        },
        collapsed: false,
      },
    },
  },
];

const incidentResponseEdges: Edge[] = [
  // Edges will be added here
];

export const IncidentResponse: Story = {
  args: {
    nodes: incidentResponseNodes,
    edges: incidentResponseEdges,
  },
  render: (args) => {
    return (
      <div className="h-[100vh] w-full ">
        <CanvasPage {...args} />
      </div>
    );
  },
};

IncidentResponse.storyName = "04 - Incident Response";
