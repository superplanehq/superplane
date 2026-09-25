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
  "dependabot-alerts": {
    triggerComponent: "github.onDependabotAlert",
    triggerName: "On Dependabot Alert",
    createTitle:
      'Bump {{ root().data.alert.dependency.package.name ?? "dependency" }} in {{ root().data.alert.dependency.manifest_path ?? "the manifest" }}',
    createDescription: "{{ root().data.alert.html_url }}",
    title: "Dependabot alert intake",
  },
  "jira-issues": {
    triggerComponent: "jira.onIssue",
    triggerName: "On Issue",
    createTitle: "{{ root().data.issue.key }}: {{ root().data.issue.fields.summary }}",
    // Jira sends the raw description as an Atlassian Document Format object.
    // The trigger reports a plain text copy next to it.
    createDescription: "{{ root().data.description }}",
    title: "Jira issue intake",
  },
  "sentry-exceptions": {
    triggerComponent: "sentry.onIssue",
    triggerName: "On Issue",
    createTitle: "{{ root().data.data.issue.title }}",
    createDescription: "{{ root().data.description }}",
    title: "Sentry exception intake",
  },
  "pagerduty-incidents": {
    triggerComponent: "pagerduty.onIncident",
    triggerName: "On Incident",
    createTitle: "{{ root().data.incident.title }}",
    createDescription: "{{ root().data.incident.html_url }}",
    title: "PagerDuty incident intake",
  },
  "productive-tasks": {
    triggerComponent: "productive.onTask",
    triggerName: "On Task",
    createTitle: "{{ root().data.data.attributes.title }}",
    createDescription: "{{ root().data.data.attributes.description }}",
    title: "Productive task intake",
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
      name: "Create Task",
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
