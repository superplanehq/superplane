import type { ComponentsEdge, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import githubIcon from "@/assets/icons/integrations/github.svg";
import pagerdutyIcon from "@/assets/icons/integrations/pagerduty.svg";
import sentryIcon from "@/assets/icons/integrations/sentry.svg";
import { getUserInitials } from "@/lib/orgUserDisplay";
import type { FactoryNodeStatus } from "@/ui/factoryNodeChrome/types";

import {
  STORYBOOK_ME_USER_AVATAR_URL,
  STORYBOOK_ME_USER_ID,
  STORYBOOK_ME_USER_NAME,
} from "../__fixtures__/factoryPageResponses";
import type { WorkOrderStatusNotePresentation } from "../lib/workOrderStatusNote";
import type { SplitRunCanvasModel } from "./work-order-split-run/splitRunCanvases";
import type { SplitRunFixture, SplitRunPhase, SplitRunStreamLine } from "./work-order-split-run/splitRunMocks";

export type LineIntakeSourceId = "github-issues" | "sentry-exceptions" | "pagerduty-incidents";

export type LineIntakeListenKind = "webhook" | "poll";

export interface LineIntakeSource {
  id: LineIntakeSourceId;
  name: string;
  description: string;
  iconSrc: string;
  iconAlt: string;
  /** How SuperPlane receives events from this source. */
  listen: {
    kind: LineIntakeListenKind;
    label: string;
  };
  /** Runner that classifies whether the event becomes a work order. */
  evaluate: {
    label: string;
    rule: string;
  };
  /** Accepted events land in the line backlog. */
  accept: {
    destination: "backlog";
    label: string;
  };
}

/**
 * Intake is an automation that listens to an external source, decides which
 * events to take in, and creates backlog work orders with that context.
 */
export const LINE_INTAKE_SOURCES: LineIntakeSource[] = [
  {
    id: "github-issues",
    name: "GitHub issues",
    description: "Open issues from connected repositories.",
    iconSrc: githubIcon,
    iconAlt: "GitHub",
    listen: {
      kind: "webhook",
      label: "On GitHub issue",
    },
    evaluate: {
      label: "Classify with a runner",
      rule: "A runner classifies the issue and decides whether to create a work order.",
    },
    accept: {
      destination: "backlog",
      label: "Create a work order in Backlog",
    },
  },
  {
    id: "sentry-exceptions",
    name: "Sentry exceptions",
    description: "Unresolved errors from production.",
    iconSrc: sentryIcon,
    iconAlt: "Sentry",
    listen: {
      kind: "webhook",
      label: "On Sentry exception",
    },
    evaluate: {
      label: "Classify with a runner",
      rule: "A runner classifies the exception and decides whether to create a work order.",
    },
    accept: {
      destination: "backlog",
      label: "Create a work order in Backlog",
    },
  },
  {
    id: "pagerduty-incidents",
    name: "PagerDuty incidents",
    description: "Firing incidents that need a work order.",
    iconSrc: pagerdutyIcon,
    iconAlt: "PagerDuty",
    listen: {
      kind: "webhook",
      label: "On PagerDuty incident",
    },
    evaluate: {
      label: "Classify with a runner",
      rule: "A runner classifies the incident and decides whether to create a work order.",
    },
    accept: {
      destination: "backlog",
      label: "Create a work order in Backlog",
    },
  },
];

export function lineIntakeSourceById(id: string): LineIntakeSource | undefined {
  return LINE_INTAKE_SOURCES.find((source) => source.id === id);
}

export interface AddIntakeTemplate {
  id: string;
  name: string;
  description: string;
  /** Optional integration icon. Letter glyph when omitted. */
  iconSrc?: string;
}

/**
 * Templates in the Add intake picker. Source-based intakes and a few
 * common improvement automations.
 */
export const ADD_INTAKE_TEMPLATES: AddIntakeTemplate[] = [
  {
    id: "github-issues",
    name: "GitHub issues",
    description: "Open issues from connected repositories.",
    iconSrc: githubIcon,
  },
  {
    id: "sentry-exceptions",
    name: "Sentry exceptions",
    description: "Unresolved errors from production.",
    iconSrc: sentryIcon,
  },
  {
    id: "pagerduty-incidents",
    name: "PagerDuty incidents",
    description: "Firing incidents that need a work order.",
    iconSrc: pagerdutyIcon,
  },
  {
    id: "improve-ci-runtime",
    name: "Improve CI runtime",
    description: "Find slow jobs and cut pipeline wait time.",
  },
  {
    id: "improve-page-performance",
    name: "Improve page performance",
    description: "Track slow pages and open work to speed them up.",
  },
  {
    id: "flaky-tests",
    name: "Flaky tests",
    description: "Catch unstable tests and create fix work orders.",
  },
];

export function filterAddIntakeTemplates(query: string): AddIntakeTemplate[] {
  const needle = query.trim().toLowerCase();
  if (!needle) {
    return ADD_INTAKE_TEMPLATES;
  }
  return ADD_INTAKE_TEMPLATES.filter((template) => {
    const haystack = `${template.name} ${template.description}`.toLowerCase();
    return haystack.includes(needle);
  });
}

const OWNER = {
  id: STORYBOOK_ME_USER_ID,
  name: STORYBOOK_ME_USER_NAME,
  initials: getUserInitials(STORYBOOK_ME_USER_NAME),
  avatarUrl: STORYBOOK_ME_USER_AVATAR_URL,
};

/**
 * Builds the same popup shape as a line-board work order: log on the left,
 * automation canvas on the right. Phases are listen → evaluate → backlog.
 */
export function intakeAutomationFixture(source: LineIntakeSource): SplitRunFixture {
  const canvas = intakeCanvasForSource(source);
  const waitingNotes: WorkOrderStatusNotePresentation[] = [
    {
      key: `${source.id}-backlog`,
      headline: "Accepted events go to Backlog",
      text: `${source.evaluate.rule} Accepted items become work orders in Backlog with the source context.`,
    },
  ];

  return {
    title: source.name,
    owner: OWNER,
    elapsed: "Running",
    startedLabel: source.listen.label,
    costUsd: "—",
    tokensLabel: "Automation",
    lineName: "Intake",
    lineStatus: "running",
    currentPhaseId: "evaluate",
    waitingNotes,
    checks: [],
    footerTone: "waiting",
    phases: [listenPhase(source, canvas), evaluatePhase(source, canvas), backlogPhase(source, canvas)],
  };
}

function listenPhase(source: LineIntakeSource, canvas: SplitRunCanvasModel): SplitRunPhase {
  return {
    id: "listen",
    name: "Listen",
    status: "passed",
    duration: "—",
    componentName: source.listen.label,
    artifacts: [],
    canvas,
    stream: [
      {
        id: `${source.id}-listen`,
        at: "12:00:01",
        componentName: source.listen.label,
        status: "passed",
        detail: source.listen.kind === "webhook" ? "Webhook received" : "Poll completed",
      },
    ],
  };
}

function evaluatePhase(source: LineIntakeSource, canvas: SplitRunCanvasModel): SplitRunPhase {
  return {
    id: "evaluate",
    name: "Evaluate",
    status: "running",
    duration: "—",
    componentName: source.evaluate.label,
    artifacts: [],
    canvas,
    stream: [
      {
        id: `${source.id}-evaluate`,
        at: "12:00:04",
        componentName: source.evaluate.label,
        status: "running",
        detail: source.evaluate.rule,
      },
    ],
  };
}

function backlogPhase(source: LineIntakeSource, canvas: SplitRunCanvasModel): SplitRunPhase {
  const stream: SplitRunStreamLine[] = [
    {
      id: `${source.id}-backlog`,
      at: "12:00:08",
      componentName: source.accept.label,
      status: "pending",
      detail: "Waiting for accept",
    },
  ];
  return {
    id: "backlog",
    name: "Backlog",
    status: "pending",
    duration: "—",
    componentName: source.accept.label,
    artifacts: [],
    canvas,
    stream,
  };
}

interface IntakeCanvasSpec {
  triggerComponent: string;
  triggerName: string;
  classifyPrompt: string;
  createTitle: string;
  createDescription: string;
  title: string;
}

const INTAKE_CANVAS_BY_SOURCE: Record<LineIntakeSourceId, IntakeCanvasSpec> = {
  "github-issues": {
    triggerComponent: "github.onIssue",
    triggerName: "On Issue",
    classifyPrompt: "Classify this GitHub issue. Accept it only when it should become a work order.",
    createTitle: "{{ root().data.issue.title }}",
    createDescription: "{{ root().data.issue.body }}",
    title: "GitHub issue intake",
  },
  "sentry-exceptions": {
    triggerComponent: "sentry.onIssue",
    triggerName: "On Issue",
    classifyPrompt: "Classify this Sentry exception. Accept it only when it should become a work order.",
    createTitle: "{{ root().data.data.issue.title }}",
    createDescription: "{{ root().data.data.issue.permalink }}",
    title: "Sentry exception intake",
  },
  "pagerduty-incidents": {
    triggerComponent: "pagerduty.onIncident",
    triggerName: "On Incident",
    classifyPrompt: "Classify this PagerDuty incident. Accept it only when it should become a work order.",
    createTitle: "{{ root().data.incident.title }}",
    createDescription: "{{ root().data.incident.html_url }}",
    title: "PagerDuty incident intake",
  },
};

function intakeCanvasForSource(source: LineIntakeSource): SplitRunCanvasModel {
  const spec = INTAKE_CANVAS_BY_SOURCE[source.id];
  const triggerId = `${source.id}-trigger`;
  const runnerId = `${source.id}-classify`;
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
      id: runnerId,
      name: "Classify intake",
      type: "TYPE_ACTION",
      component: "runnerClaudeCode",
      configuration: {
        prompt: spec.classifyPrompt,
      },
      position: { x: 160, y: 260 },
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
      position: { x: 160, y: 440 },
    },
  ];
  const edges: ComponentsEdge[] = [
    { channel: "default", sourceId: triggerId, targetId: runnerId },
    { channel: "default", sourceId: runnerId, targetId: createId },
  ];

  return {
    key: "intake",
    title: spec.title,
    nodes,
    edges,
    statuses: {
      [triggerId]: "succeeded",
      [runnerId]: "running",
      [createId]: "pending",
    } satisfies Record<string, FactoryNodeStatus>,
    metrics: {},
  };
}
