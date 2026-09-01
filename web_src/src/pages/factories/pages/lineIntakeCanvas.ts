import type { ComponentsEdge, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import type { FactoryNodeStatus } from "@/ui/factoryNodeChrome/types";

import type { LineIntakeSource, LineIntakeSourceId } from "./lineIntakeModel";
import type { SplitRunCanvasModel } from "./work-order-split-run/splitRunCanvases";

interface IntakeCanvasSpec {
  triggerComponent: string;
  triggerName: string;
  createTitle: string;
  createDescription: string;
  title: string;
}

const INTAKE_CANVAS_BY_SOURCE: Record<LineIntakeSourceId, IntakeCanvasSpec> = {
  "github-issues": {
    triggerComponent: "github.onIssue",
    triggerName: "On Issue",
    createTitle: "{{ root().data.issue.title }}",
    createDescription: "{{ root().data.issue.body }}",
    title: "GitHub issue intake",
  },
  "sentry-exceptions": {
    triggerComponent: "sentry.onIssue",
    triggerName: "On Issue",
    createTitle: "{{ root().data.data.issue.title }}",
    createDescription: "{{ root().data.data.issue.permalink }}",
    title: "Sentry exception intake",
  },
  "pagerduty-incidents": {
    triggerComponent: "pagerduty.onIncident",
    triggerName: "On Incident",
    createTitle: "{{ root().data.incident.title }}",
    createDescription: "{{ root().data.incident.html_url }}",
    title: "PagerDuty incident intake",
  },
};

export function intakeCanvasForSource(source: LineIntakeSource): SplitRunCanvasModel {
  const spec = INTAKE_CANVAS_BY_SOURCE[source.id];
  const triggerId = `${source.id}-trigger`;
  const createId = `${source.id}-create`;

  const nodes: ComponentsNode[] = [
    {
      id: triggerId,
      name: spec.triggerName,
      type: "TYPE_TRIGGER",
      component: spec.triggerComponent,
      position: { x: 160, y: 80 },
    },
    {
      id: createId,
      name: "Create Work Order",
      type: "TYPE_ACTION",
      component: "createWorkOrder",
      configuration: {
        title: spec.createTitle,
        description: spec.createDescription,
      },
      position: { x: 160, y: 260 },
    },
  ];
  const edges: ComponentsEdge[] = [{ channel: "default", sourceId: triggerId, targetId: createId }];

  return {
    key: "intake",
    title: spec.title,
    nodes,
    edges,
    statuses: {
      [triggerId]: "passed",
      [createId]: "running",
    } satisfies Record<string, FactoryNodeStatus>,
    metrics: {},
  };
}
