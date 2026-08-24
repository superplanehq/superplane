import type {
  ComponentsEdge,
  FactoriesWorkOrderArtifact,
  SuperplaneComponentsNode as ComponentsNode,
} from "@/api-client";
import githubIcon from "@/assets/icons/integrations/github.svg";
import pagerdutyIcon from "@/assets/icons/integrations/pagerduty.svg";
import sentryIcon from "@/assets/icons/integrations/sentry.svg";
import { getUserInitials } from "@/lib/orgUserDisplay";
import type { FactoryNodeStatus } from "@/ui/factoryNodeChrome/types";

import { ACME_ONBOARDING_FACTORY_KEY, GITHUB_ISSUES_INTAKE_APP_ID } from "../__fixtures__/factoryPageIds";
import {
  STORYBOOK_ME_USER_AVATAR_URL,
  STORYBOOK_ME_USER_ID,
  STORYBOOK_ME_USER_NAME,
} from "../__fixtures__/factoryPageResponses";
import { CONFIDENCE_CHECK_NAME, CONFIDENCE_SCORE_MAX, confidenceCheckLevel } from "../lib/confidenceScore";
import type { WorkOrderCheckPresentation } from "../lib/workOrderChecks";
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

export function isFirstRunOnboardingFactory(factoryKey: string | undefined): boolean {
  return factoryKey === ACME_ONBOARDING_FACTORY_KEY;
}

/** Acme onboarding starts with GitHub issues only. Semaphore keeps the full set. */
export function lineIntakeSourcesForFactory(factoryKey: string | undefined): LineIntakeSource[] {
  if (isFirstRunOnboardingFactory(factoryKey)) {
    return LINE_INTAKE_SOURCES.filter((source) => source.id === "github-issues");
  }
  return LINE_INTAKE_SOURCES;
}

export function isLineIntakeSourceId(id: string | null | undefined): id is LineIntakeSourceId {
  return Boolean(id && lineIntakeSourceById(id));
}

export function intakeAutomationAppId(apps: Array<{ id?: string }>): string | undefined {
  return apps.find((app) => app.id === GITHUB_ISSUES_INTAKE_APP_ID)?.id ?? apps.find((app) => app.id)?.id;
}

export interface LineIntakeAnalyzingTicket {
  id: string;
  title: string;
  detailsMarkdown?: string;
  issueKey?: string;
  issueUrl?: string;
  planMarkdown?: string;
  confidenceScore?: number;
  confidenceSummary?: string;
  confidenceAnalysis?: string;
}

export const LINE_INTAKE_COPY = {
  analyzingTitle: "Analyzing",
  analyzingHelper: "Tickets from this intake.",
  analyzingStatus: "Analyzing",
  analyzingEmpty: "No tickets in analysis.",
  analysisHeadline: "SuperPlane is analyzing this ticket",
  analysisHelper: "SuperPlane reads the ticket and the repository. It does not start work yet.",
  analysisCompleteHeadline: "Ticket analysis finished",
  analysisCompleteHelper: "SuperPlane did not change the ticket. Review the plan before work starts.",
} as const;

/** GitHub issues pulled in for first-run analysis. Scores land on the board later. */
export const GITHUB_ISSUES_ANALYZING_TICKETS: LineIntakeAnalyzingTicket[] = [
  { id: "gh-issue-1", title: "Handle duplicate refunds on retry" },
  { id: "gh-issue-2", title: "Return 409 when the invoice is already paid" },
  { id: "gh-issue-3", title: "Show a clearer empty state on the billing page" },
  { id: "gh-issue-4", title: "Upgrade the Node 20 base image" },
  { id: "gh-issue-5", title: "Add a flake retry to the checkout e2e suite" },
];

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
 * Ticket click from GitHub issues: same split-run popup, with a canvas
 * for ingest, analyze, create plan, and score.
 */
const ANALYSIS_PHASE_DURATION = {
  ingest: "2s",
  analyze: "3m 45s",
  analyzeRunning: "3m 12s so far",
  plan: "18s",
  score: "7s",
} as const;

export function intakeTicketAnalysisFixture(
  ticket: LineIntakeAnalyzingTicket,
  options?: { complete?: boolean },
): SplitRunFixture {
  const complete = Boolean(options?.complete);
  const canvas = ticketAnalysisCanvas(complete);
  const checks = complete ? confidenceChecks(ticket) : [];
  return {
    title: ticket.title,
    owner: OWNER,
    elapsed: complete ? "4m 12s" : "Running",
    startedLabel: "Analyze ticket",
    costUsd: complete ? "0.18" : "—",
    tokensLabel: "Analysis",
    lineName: "Intake",
    lineStatus: complete ? "passed" : "running",
    currentPhaseId: complete ? "score" : "analyze",
    waitingNotes: [
      {
        key: `${ticket.id}-${complete ? "complete" : "analyzing"}`,
        headline: complete ? LINE_INTAKE_COPY.analysisCompleteHeadline : LINE_INTAKE_COPY.analysisHeadline,
        text: complete ? LINE_INTAKE_COPY.analysisCompleteHelper : LINE_INTAKE_COPY.analysisHelper,
      },
    ],
    checks,
    footerTone: complete ? undefined : "waiting",
    phases: [
      ticketAnalysisPhase({
        id: "ingest",
        name: "Ingest",
        status: "passed",
        duration: ANALYSIS_PHASE_DURATION.ingest,
        componentName: "Ingest",
        detail: "Ticket received from GitHub issues.",
        canvas,
        artifacts: ingestArtifacts(ticket),
      }),
      ticketAnalysisPhase({
        id: "analyze",
        name: "Analyze",
        status: complete ? "passed" : "running",
        duration: complete ? ANALYSIS_PHASE_DURATION.analyze : ANALYSIS_PHASE_DURATION.analyzeRunning,
        componentName: "Analyze ticket",
        detail: complete ? "Read the ticket and the repository." : "Reading the ticket and the repository.",
        canvas,
      }),
      ticketAnalysisPhase({
        id: "plan",
        name: "Create plan",
        status: complete ? "passed" : "pending",
        duration: complete ? ANALYSIS_PHASE_DURATION.plan : "—",
        componentName: "Create plan",
        detail: complete ? "Wrote the implementation plan." : "Waiting for analysis to finish.",
        canvas,
        artifacts: complete ? planArtifact(ticket) : [],
      }),
      ticketAnalysisPhase({
        id: "score",
        name: "Score",
        status: complete ? "passed" : "pending",
        duration: complete ? ANALYSIS_PHASE_DURATION.score : "—",
        componentName: "Score",
        detail: complete ? "Scored the ticket against the codebase." : "Waiting for a plan.",
        canvas,
        checks,
      }),
    ],
  };
}

function ingestArtifacts(ticket: LineIntakeAnalyzingTicket): FactoriesWorkOrderArtifact[] {
  const issueKey = ticket.issueKey ?? ticket.id;
  return [
    {
      id: `${ticket.id}-details`,
      type: "TYPE_MARKDOWN",
      data: {
        name: "details.md",
        title: "details.md",
        body: ticket.detailsMarkdown ?? ticket.title,
      },
    },
    {
      id: `${ticket.id}-issue-link`,
      type: "TYPE_LINK",
      data: {
        title: issueKey,
        url: ticket.issueUrl ?? githubIssueUrlForTicket(issueKey),
      },
    },
  ];
}

function planArtifact(ticket: LineIntakeAnalyzingTicket): FactoriesWorkOrderArtifact[] {
  if (!ticket.planMarkdown) {
    return [];
  }
  return [
    {
      id: `${ticket.id}-plan`,
      type: "TYPE_MARKDOWN",
      data: {
        name: "plan.md",
        title: "plan.md",
        body: ticket.planMarkdown,
      },
    },
  ];
}

function confidenceChecks(ticket: LineIntakeAnalyzingTicket): WorkOrderCheckPresentation[] {
  if (ticket.confidenceScore == null) {
    return [];
  }
  return [
    {
      id: `${ticket.id}-confidence`,
      name: CONFIDENCE_CHECK_NAME,
      score: ticket.confidenceScore,
      maxScore: CONFIDENCE_SCORE_MAX,
      format: "fraction",
      level: confidenceCheckLevel(ticket.confidenceScore),
      summary: ticket.confidenceSummary,
      analysis: ticket.confidenceAnalysis,
      sourceName: "Score",
    },
  ];
}

function githubIssueUrlForTicket(issueKey: string): string {
  const number = issueKey.replace(/\D/g, "") || "1";
  return `https://github.com/acme/payments-service/issues/${number}`;
}

function ticketAnalysisPhase({
  id,
  name,
  status,
  duration,
  componentName,
  detail,
  canvas,
  artifacts = [],
  checks = [],
}: {
  id: SplitRunPhase["id"];
  name: string;
  status: SplitRunPhase["status"];
  duration: string;
  componentName: string;
  detail: string;
  canvas: SplitRunCanvasModel;
  artifacts?: FactoriesWorkOrderArtifact[];
  checks?: WorkOrderCheckPresentation[];
}): SplitRunPhase {
  return {
    id,
    name,
    status,
    duration,
    componentName,
    artifacts,
    checks,
    canvas,
    stream: [
      {
        id: `ticket-${id}`,
        at: "12:00:04",
        componentName,
        status,
        detail,
      },
    ],
  };
}

function ticketAnalysisCanvas(complete: boolean): SplitRunCanvasModel {
  const ingestId = "ticket-ingest";
  const analyzeId = "ticket-analyze";
  const planId = "ticket-plan";
  const scoreId = "ticket-score";

  const nodes: ComponentsNode[] = [
    {
      id: ingestId,
      name: "Ingest",
      type: "TYPE_TRIGGER",
      component: "github.onIssue",
      position: { x: 160, y: 40 },
    },
    {
      id: analyzeId,
      name: "Analyze ticket",
      type: "TYPE_ACTION",
      component: "runnerClaudeCode",
      configuration: {
        prompt: "Read this ticket and the repository. Find the files and risks that matter.",
      },
      position: { x: 160, y: 200 },
    },
    {
      id: planId,
      name: "Create plan",
      type: "TYPE_ACTION",
      component: "addWorkOrderArtifact",
      configuration: {
        name: "plan.md",
      },
      position: { x: 160, y: 360 },
    },
    {
      id: scoreId,
      name: "Score",
      type: "TYPE_ACTION",
      component: "reportWorkOrderCheck",
      configuration: {
        name: "confidence",
      },
      position: { x: 160, y: 520 },
    },
  ];

  return {
    key: "intake",
    title: "Ticket analysis",
    nodes,
    edges: [
      { channel: "default", sourceId: ingestId, targetId: analyzeId },
      { channel: "default", sourceId: analyzeId, targetId: planId },
      { channel: "default", sourceId: planId, targetId: scoreId },
    ],
    statuses: {
      [ingestId]: "triggered",
      [analyzeId]: complete ? "passed" : "running",
      [planId]: complete ? "passed" : "pending",
      [scoreId]: complete ? "passed" : "pending",
    } satisfies Record<string, FactoryNodeStatus>,
    metrics: {
      [ingestId]: ANALYSIS_PHASE_DURATION.ingest,
      [analyzeId]: complete ? ANALYSIS_PHASE_DURATION.analyze : ANALYSIS_PHASE_DURATION.analyzeRunning,
      [planId]: complete ? ANALYSIS_PHASE_DURATION.plan : "—",
      [scoreId]: complete ? ANALYSIS_PHASE_DURATION.score : "—",
    },
  };
}

/**
 * Builds the same popup shape as a line-board work order: log on the left,
 * automation canvas on the right. Phases are listen → evaluate → backlog.
 */
export function intakeAutomationCanvas(source: LineIntakeSource): SplitRunCanvasModel {
  return intakeCanvasForSource(source);
}

export function intakeAutomationFixture(source: LineIntakeSource): SplitRunFixture {
  const canvas = intakeAutomationCanvas(source);
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
