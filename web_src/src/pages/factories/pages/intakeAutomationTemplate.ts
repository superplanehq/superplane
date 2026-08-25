import yaml from "js-yaml";

import type { LineIntakeSourceId } from "./lineIntakeModel";

interface IntakeAutomationSpec {
  name: string;
  description: string;
  triggerComponent: string;
  triggerName: string;
  triggerConfiguration: Record<string, unknown>;
  analysisSubject: string;
  createTitle: string;
  createDescription: string;
}

const INTAKE_AUTOMATION_SPECS: Record<LineIntakeSourceId, IntakeAutomationSpec> = {
  "github-issues": {
    name: "GitHub issues",
    description: "Analyze new GitHub issues and create work orders for suitable changes.",
    triggerComponent: "github.onIssue",
    triggerName: "On Issue",
    triggerConfiguration: { actions: ["opened"] },
    analysisSubject: "GitHub issue",
    createTitle: "{{ root().data.issue.title }}",
    createDescription: "{{ root().data.issue.body }}",
  },
  "sentry-exceptions": {
    name: "Sentry exceptions",
    description: "Analyze new Sentry exceptions and create work orders for suitable fixes.",
    triggerComponent: "sentry.onIssue",
    triggerName: "On Issue Event",
    triggerConfiguration: { actions: ["created", "unresolved"] },
    analysisSubject: "Sentry exception",
    createTitle: "{{ root().data.data.issue.title }}",
    createDescription: "{{ root().data.data.issue.permalink }}",
  },
  "pagerduty-incidents": {
    name: "PagerDuty incidents",
    description: "Analyze triggered PagerDuty incidents and create work orders for suitable follow-up work.",
    triggerComponent: "pagerduty.onIncident",
    triggerName: "On Incident",
    triggerConfiguration: { events: ["incident.triggered"], urgencies: ["high", "low"] },
    analysisSubject: "PagerDuty incident",
    createTitle: "{{ root().data.incident.title }}",
    createDescription: "{{ root().data.incident.html_url }}",
  },
};

export function intakeAutomationName(sourceId: LineIntakeSourceId): string {
  return INTAKE_AUTOMATION_SPECS[sourceId].name;
}

export function intakeAutomationDescription(sourceId: LineIntakeSourceId): string {
  return INTAKE_AUTOMATION_SPECS[sourceId].description;
}

export function buildIntakeAutomationYaml(sourceId: LineIntakeSourceId, confidencePct: number): string {
  const spec = INTAKE_AUTOMATION_SPECS[sourceId];
  const threshold = Math.min(100, Math.max(0, Math.round(confidencePct)));
  const triggerId = `${sourceId}-trigger`;
  const analysisId = `${sourceId}-analysis`;
  const thresholdId = `${sourceId}-threshold`;
  const createId = `${sourceId}-create`;

  return yaml.dump(
    {
      apiVersion: "v1",
      kind: "Canvas",
      metadata: {
        name: spec.name,
        description: spec.description,
      },
      spec: {
        edges: [
          { channel: "default", sourceId: triggerId, targetId: analysisId },
          { channel: "passed", sourceId: analysisId, targetId: thresholdId },
          { channel: "true", sourceId: thresholdId, targetId: createId },
        ],
        nodes: [
          {
            id: triggerId,
            name: spec.triggerName,
            type: "TYPE_TRIGGER",
            component: spec.triggerComponent,
            configuration: spec.triggerConfiguration,
            position: { x: 160, y: 80 },
          },
          {
            id: analysisId,
            name: "Analyze intake",
            type: "TYPE_ACTION",
            component: "runnerClaudeCode",
            configuration: {
              steps: [
                {
                  name: "Analyze and score",
                  type: "prompt",
                  prompt: buildAnalysisPrompt(spec.analysisSubject),
                },
              ],
            },
            position: { x: 160, y: 260 },
          },
          {
            id: thresholdId,
            name: "Meets confidence threshold?",
            type: "TYPE_ACTION",
            component: "if",
            configuration: {
              expression: `int($["Analyze intake"].data[0].result.result) >= ${threshold}`,
            },
            position: { x: 160, y: 440 },
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
            position: { x: 160, y: 620 },
          },
        ],
      },
    },
    { lineWidth: -1, noRefs: true },
  );
}

function buildAnalysisPrompt(subject: string): string {
  return [
    `Analyze this ${subject} and decide whether it is suitable for an engineering work order.`,
    "Consider impact, clarity, feasibility, and whether an engineer can take a concrete action.",
    "Return only one integer from 0 through 100. A higher value means greater confidence.",
    "",
    "Event:",
    "{{ root().data }}",
  ].join("\n");
}
