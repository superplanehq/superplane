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
    createDescription: '{{ root().data.issue.body ?? "" }}',
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
    createDescription:
      '{{ sentryIssueDescription(root().data.data.issue, $["Get Sentry Issue"].data?.body, $["Get Sentry Event"].data?.body, $["Get Recommended Sentry Event"]?.data?.body) }}',
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
  datadog: {
    triggerComponent: "datadog.onErrorTrackingAlert",
    triggerName: "On Error Tracking Alert",
    createTitle: "{{ root().data.title }}",
    createDescription: "{{ root().data.body }}\n\n{{ root().data.link }}",
    title: "Datadog error intake",
  },
  "linear-issues": {
    triggerComponent: "linear.onIssue",
    triggerName: "On Issue",
    createTitle: "{{ root().data.data.identifier }}: {{ root().data.data.title }}",
    createDescription: "{{ root().data.data.description }}",
    title: "Linear issue intake",
  },
};

export function intakeCanvasForSource(source: LineIntakeSource): SplitRunCanvasModel {
  const spec = INTAKE_CANVAS_BY_SOURCE[source.id];
  const triggerId = `${source.id}-trigger`;
  const createId = `${source.id}-create`;
  const filterId = `${source.id}-filter`;
  const issueRequestId = `${source.id}-issue-request`;
  const eventRequestId = `${source.id}-event-request`;
  const eventHasStackId = `${source.id}-event-has-stack`;
  const recommendedEventRequestId = `${source.id}-recommended-event-request`;
  const sentry = source.id === "sentry-exceptions";

  const nodes: ComponentsNode[] = [
    {
      id: triggerId,
      name: spec.triggerName,
      type: "TYPE_TRIGGER",
      component: spec.triggerComponent,
      position: { x: 160, y: 80 },
    },
    ...(sentry
      ? ([
          {
            id: filterId,
            name: "Matches filters?",
            type: "TYPE_ACTION",
            component: "if",
            configuration: { expression: "true" },
            position: { x: 160, y: 260 },
          },
          {
            id: issueRequestId,
            name: "Get Sentry Issue",
            type: "TYPE_ACTION",
            component: "http",
            configuration: {
              method: "GET",
              redactSensitiveHeaders: true,
              url: "{{ root().data.issueApiUrl }}",
              authorization: {
                type: "integration_bearer",
                integration: { name: "Sentry" },
                integrationKey: "SENTRY_AUTH_TOKEN",
              },
            },
            position: { x: 160, y: 440 },
          },
          {
            id: eventRequestId,
            name: "Get Sentry Event",
            type: "TYPE_ACTION",
            component: "http",
            configuration: {
              method: "GET",
              redactSensitiveHeaders: true,
              url: "{{ root().data.issueApiUrl }}events/latest/",
              authorization: {
                type: "integration_bearer",
                integration: { name: "Sentry" },
                integrationKey: "SENTRY_AUTH_TOKEN",
              },
            },
            position: { x: 160, y: 620 },
          },
          {
            id: eventHasStackId,
            name: "Latest event has a stack?",
            type: "TYPE_ACTION",
            component: "if",
            configuration: { expression: 'sentryEventHasStack($["Get Sentry Event"].data?.body)' },
            position: { x: 160, y: 800 },
          },
          {
            id: recommendedEventRequestId,
            name: "Get Recommended Sentry Event",
            type: "TYPE_ACTION",
            component: "http",
            configuration: {
              method: "GET",
              redactSensitiveHeaders: true,
              url: "{{ root().data.issueApiUrl }}events/recommended/",
              authorization: {
                type: "integration_bearer",
                integration: { name: "Sentry" },
                integrationKey: "SENTRY_AUTH_TOKEN",
              },
            },
            position: { x: 160, y: 980 },
          },
        ] satisfies ComponentsNode[])
      : []),
    {
      id: createId,
      name: "Create Task",
      type: "TYPE_ACTION",
      component: "createWorkOrder",
      configuration: {
        title: spec.createTitle,
        description: spec.createDescription,
      },
      position: { x: 160, y: sentry ? 1160 : 260 },
    },
  ];
  const edges: ComponentsEdge[] = sentry
    ? [
        { channel: "default", sourceId: triggerId, targetId: filterId },
        { channel: "true", sourceId: filterId, targetId: issueRequestId },
        { channel: "success", sourceId: issueRequestId, targetId: eventRequestId },
        { channel: "failure", sourceId: issueRequestId, targetId: eventRequestId },
        { channel: "success", sourceId: eventRequestId, targetId: eventHasStackId },
        { channel: "failure", sourceId: eventRequestId, targetId: eventHasStackId },
        { channel: "true", sourceId: eventHasStackId, targetId: createId },
        { channel: "false", sourceId: eventHasStackId, targetId: recommendedEventRequestId },
        { channel: "success", sourceId: recommendedEventRequestId, targetId: createId },
        { channel: "failure", sourceId: recommendedEventRequestId, targetId: createId },
      ]
    : [{ channel: "default", sourceId: triggerId, targetId: createId }];

  return {
    key: "intake",
    title: spec.title,
    nodes,
    edges,
    statuses: {
      [triggerId]: "passed",
      ...(sentry
        ? {
            [filterId]: "passed",
            [issueRequestId]: "passed",
            [eventRequestId]: "passed",
            [eventHasStackId]: "passed",
            [recommendedEventRequestId]: "passed",
          }
        : {}),
      [createId]: "running",
    } satisfies Record<string, FactoryNodeStatus>,
    metrics: {},
  };
}
