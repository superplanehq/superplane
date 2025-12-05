import type { Node, Edge } from "@xyflow/react";
import { parseExpression } from "@/lib/expressionParser";

export const mockNodes: Node[] = [
  {
    id: "if-node-1",
    type: "default",
    position: { x: 100, y: 200 },
    data: {
      label: "Check Environment",
      state: "pending",
      type: "component",
      outputChannels: ["true", "false"],
      component: {
        title: "Check Environment",
        iconSlug: "split",
        headerColor: "bg-gray-50",
        specs: [
          {
            title: "condition",
            tooltipTitle: "conditions applied",
            values: parseExpression("config.environment == prod"),
          },
        ],
        eventSections: [
          {
            title: "TRUE",
            eventTitle: "No events received yet",
            eventState: "neutral" as const,
          },
          {
            title: "FALSE",
            eventTitle: "No events received yet",
            eventState: "neutral" as const,
          },
        ],
        collapsed: false,
      },
    },
  },
  {
    id: "approval-node-1",
    type: "default",
    position: { x: 700, y: 50 },
    data: {
      label: "Production Approval",
      state: "pending",
      type: "approval",
      outputChannels: ["default"],
      approval: {
        title: "Production Approval",
        description: "Get approval before deploying to production",
        iconSlug: "hand",
        iconColor: "text-orange-500",
        headerColor: "bg-orange-100",
        collapsedBackground: "bg-orange-100",
        approvals: [
          {
            id: "approval-1",
            title: "Security Team",
            approved: false,
          },
          {
            id: "approval-2",
            title: "Engineering Lead",
            approved: false,
          },
        ],
        awaitingEvent: null,
        collapsed: false,
      },
    },
  },
  {
    id: "filter-node-1",
    type: "default",
    position: { x: 1500, y: 200 },
    data: {
      label: "Filter Services",
      state: "pending",
      type: "filter",
      outputChannels: ["default"],
      filter: {
        title: "Filter Services",
        filters: [
          {
            field: "service.enabled",
            operator: "==",
            value: "true",
          },
        ],
        collapsed: false,
      },
    },
  },
  {
    id: "noop-node-1",
    type: "default",
    position: { x: 700, y: 400 },
    data: {
      label: "Skip Approval",
      state: "pending",
      type: "component",
      outputChannels: ["default"],
      component: {
        title: "Skip Approval",
        iconSlug: "circle-off",
        headerColor: "bg-gray-50",
        collapsed: false,
        eventSections: [
          {
            title: "Last Run",
            eventTitle: "No events received yet",
            eventState: "neutral" as const,
          },
        ],
      },
    },
  },
];

const EDGE_STYLE = {
  type: "default",
  style: { stroke: "#C9D5E1", strokeWidth: 3 },
} as const;

export const mockEdges: Edge[] = [
  {
    id: "e1",
    source: "if-node-1",
    sourceHandle: "true",
    target: "approval-node-1",
    ...EDGE_STYLE,
  },
  {
    id: "e2",
    source: "if-node-1",
    sourceHandle: "false",
    target: "noop-node-1",
    ...EDGE_STYLE,
  },
  {
    id: "e3",
    source: "approval-node-1",
    sourceHandle: "default",
    target: "filter-node-1",
    ...EDGE_STYLE,
  },
  {
    id: "e4",
    source: "noop-node-1",
    sourceHandle: "default",
    target: "filter-node-1",
    ...EDGE_STYLE,
  },
];
