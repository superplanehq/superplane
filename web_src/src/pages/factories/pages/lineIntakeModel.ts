import type {
  FactoriesFactoryIntake,
  FactoriesFactoryIntakeSource,
  FactoriesWorkOrderArtifact,
  SuperplaneComponentsNode as ComponentsNode,
} from "@/api-client";
import githubIcon from "@/assets/icons/integrations/github.svg";
import pagerdutyIcon from "@/assets/icons/integrations/pagerduty.svg";
import sentryIcon from "@/assets/icons/integrations/sentry.svg";
import { getUserInitials } from "@/lib/orgUserDisplay";
import type { FactoryNodeStatus } from "@/ui/factoryNodeChrome/types";

import { ACME_ONBOARDING_FACTORY_KEY } from "../__fixtures__/factoryPageIds";
import {
  STORYBOOK_ME_USER_AVATAR_URL,
  STORYBOOK_ME_USER_ID,
  STORYBOOK_ME_USER_NAME,
} from "../__fixtures__/factoryPageResponses";
import {
  CONFIDENCE_CHECK_NAME,
  CONFIDENCE_SCORE_MAX,
  confidenceBandForScore,
  confidenceCheckLevel,
  confidenceScoreFromPercent,
  confidenceSuitabilitySummary,
} from "../lib/confidenceScore";
import type { WorkOrderCheckPresentation } from "../lib/workOrderChecks";
import type { WorkOrderStatusNotePresentation } from "../lib/workOrderStatusNote";
import { intakeSettingsFromApi, type IntakeSourceSettings } from "./intakeSourceSettingsModel";
import { intakeCanvasForSource } from "./lineIntakeCanvas";
import type { SplitRunCanvasModel } from "./work-order-split-run/splitRunCanvases";
import type { SplitRunFixture, SplitRunPhase, SplitRunStreamLine } from "./work-order-split-run/splitRunMocks";
import { splitRunIntakeSource } from "./work-order-split-run/splitRunSource";

export { ADD_INTAKE_TEMPLATES, filterAddIntakeTemplates, type AddIntakeTemplate } from "./addIntakeTemplates";

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
 * Intake is an automation that listens to an external source and creates
 * backlog work orders. SuperPlane scores those work orders in Backlog.
 */
export const LINE_INTAKE_SOURCES: LineIntakeSource[] = [
  {
    id: "github-issues",
    name: "GitHub issues",
    description: "Creates tasks from GitHub issues.",
    iconSrc: githubIcon,
    iconAlt: "GitHub",
    listen: {
      kind: "webhook",
      label: "On GitHub issue",
    },
    evaluate: {
      label: "Create a work order",
      rule: "A matching GitHub issue becomes a work order in Backlog. SuperPlane scores it there.",
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
      label: "Create a work order",
      rule: "A matching Sentry exception becomes a work order in Backlog. SuperPlane scores it there.",
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
      label: "Create a work order",
      rule: "A matching PagerDuty incident becomes a work order in Backlog. SuperPlane scores it there.",
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

export function lineIntakeSourceForApiSource(apiSource: string | undefined): LineIntakeSource | undefined {
  if (!apiSource) {
    return undefined;
  }
  const sourceId = LINE_INTAKE_SOURCE_ID_BY_API_SOURCE[apiSource];
  return sourceId ? lineIntakeSourceById(sourceId) : undefined;
}

export function isFirstRunOnboardingFactory(factoryKey: string | undefined): boolean {
  return factoryKey === ACME_ONBOARDING_FACTORY_KEY;
}

export function isLineIntakeSourceId(id: string | null | undefined): id is LineIntakeSourceId {
  return Boolean(id && lineIntakeSourceById(id));
}

/**
 * An intake the workspace has declared, joined with the presentation copy for
 * its source. `intakeId` is the identity: a workspace can run two GitHub
 * intakes with different filters.
 */
export interface ConfiguredLineIntakeSource {
  intakeId: string;
  /** Canvas that implements the intake, used to open the automation editor. */
  appId: string;
  healthy: boolean;
  settings: IntakeSourceSettings;
  source: LineIntakeSource;
}

const LINE_INTAKE_SOURCE_ID_BY_API_SOURCE: Record<string, LineIntakeSourceId> = {
  SOURCE_GITHUB_ISSUES: "github-issues",
  SOURCE_SENTRY_EXCEPTIONS: "sentry-exceptions",
  SOURCE_PAGERDUTY_INCIDENTS: "pagerduty-incidents",
};

const API_SOURCE_BY_LINE_INTAKE_SOURCE_ID: Record<LineIntakeSourceId, FactoriesFactoryIntakeSource> = {
  "github-issues": "SOURCE_GITHUB_ISSUES",
  "sentry-exceptions": "SOURCE_SENTRY_EXCEPTIONS",
  "pagerduty-incidents": "SOURCE_PAGERDUTY_INCIDENTS",
};

export function apiIntakeSource(sourceId: LineIntakeSourceId): FactoriesFactoryIntakeSource {
  return API_SOURCE_BY_LINE_INTAKE_SOURCE_ID[sourceId];
}

export function intakeSourcesFromFactoryIntakes(intakes: FactoriesFactoryIntake[]): ConfiguredLineIntakeSource[] {
  return intakes.flatMap((intake) => {
    const intakeId = intake.id?.trim();
    const source = lineIntakeSourceForApiSource(intake.source);
    if (!intakeId || !source) {
      return [];
    }

    const name = intake.name?.trim() || source.name;
    return [
      {
        intakeId,
        appId: intake.canvasId?.trim() ?? "",
        healthy: intake.healthy !== false,
        settings: intakeSettingsFromApi(name, intake.settings),
        source: { ...source, name },
      },
    ];
  });
}

/**
 * A ticket the intake still analyzes, or one that finished with a score under
 * the minimum confidence. Tickets over the minimum leave the list for Backlog.
 */
export type LineIntakeTicketOutcome = "analyzing" | "below-threshold";

export interface LineIntakeAnalyzingTicket {
  id: string;
  title: string;
  appId?: string;
  runId?: string;
  outcome?: LineIntakeTicketOutcome;
  detailsMarkdown?: string;
  issueKey?: string;
  issueUrl?: string;
  planMarkdown?: string;
  /** Score the intake reports, on the same scale as the minimum confidence. */
  confidencePct?: number;
  confidenceScore?: number;
  confidenceSummary?: string;
  confidenceAnalysis?: string;
}

export function isBelowThresholdTicket(ticket: LineIntakeAnalyzingTicket): boolean {
  return ticket.outcome === "below-threshold";
}

export function intakeTicketConfidenceScore(ticket: LineIntakeAnalyzingTicket): number | undefined {
  if (ticket.confidenceScore != null) {
    return ticket.confidenceScore;
  }
  if (ticket.confidencePct != null) {
    return confidenceScoreFromPercent(ticket.confidencePct);
  }
  return undefined;
}

export const LINE_INTAKE_COPY = {
  needsRepair: "Needs repair",
  needsRepairHelper: "The automation can no longer create work orders. Open it to repair the steps.",
  analysisHeadline: "SuperPlane is analyzing this ticket",
  analysisHelper: "SuperPlane reads the ticket and the repository. It does not start work yet.",
  analysisCompleteHeadline: "Ticket analysis finished",
  analysisCompleteHelper: "SuperPlane did not change the ticket. Review the plan before work starts.",
} as const;

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

function ticketAnalysisPresentation(complete: boolean) {
  return {
    elapsed: complete ? "4m 12s" : "Running",
    costUsd: complete ? "0.18" : "—",
    lineStatus: complete ? "passed" : "running",
    currentPhaseId: complete ? "score" : "analyze",
    noteKey: complete ? "complete" : "analyzing",
    headline: complete ? LINE_INTAKE_COPY.analysisCompleteHeadline : LINE_INTAKE_COPY.analysisHeadline,
    text: complete ? LINE_INTAKE_COPY.analysisCompleteHelper : LINE_INTAKE_COPY.analysisHelper,
    analyzeStatus: complete ? "passed" : "running",
    laterStatus: complete ? "passed" : "pending",
    analyzeDetail: complete ? "Read the ticket and the repository." : "Reading the ticket and the repository.",
    planDetail: complete ? "Wrote the implementation plan." : "Waiting for analysis to finish.",
    scoreDetail: complete ? "Scored the ticket against the codebase." : "Waiting for a plan.",
  } as const;
}

export function intakeTicketAnalysisFixture(
  ticket: LineIntakeAnalyzingTicket,
  options?: { complete?: boolean },
): SplitRunFixture {
  const complete = options?.complete ?? isBelowThresholdTicket(ticket);
  const canvas = ticketAnalysisCanvas(complete);
  const checks = complete ? confidenceChecks(ticket) : [];
  const view = ticketAnalysisPresentation(complete);
  return {
    title: ticket.title,
    owner: OWNER,
    elapsed: view.elapsed,
    startedLabel: "Analyze ticket",
    costUsd: view.costUsd,
    tokensLabel: "Analysis",
    lineName: "Intake",
    lineStatus: view.lineStatus,
    currentPhaseId: view.currentPhaseId,
    currentStepIndex: 0,
    waitingNotes: [
      {
        key: `${ticket.id}-${view.noteKey}`,
        headline: view.headline,
        text: view.text,
      },
    ],
    checks,
    // Analysis is not a line run — the footer narrates without line actions.
    footer: {
      kind: complete ? "done" : "running",
      sentence: "",
      actions: [],
      note: { headline: view.headline, text: view.text },
    },
    footerTone: complete ? "done" : "running",
    source: splitRunIntakeSource(ticket.issueUrl ?? githubIssueUrlForTicket(ticket.issueKey ?? ticket.id)),
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
        appId: ticket.appId,
      }),
      ticketAnalysisPhase({
        id: "analyze",
        name: "Analyze",
        status: view.analyzeStatus,
        duration: complete ? ANALYSIS_PHASE_DURATION.analyze : ANALYSIS_PHASE_DURATION.analyzeRunning,
        componentName: "Analyze ticket",
        detail: view.analyzeDetail,
        canvas,
        appId: ticket.appId,
      }),
      ticketAnalysisPhase({
        id: "plan",
        name: "Create plan",
        status: view.laterStatus,
        duration: complete ? ANALYSIS_PHASE_DURATION.plan : "—",
        componentName: "Create plan",
        detail: view.planDetail,
        canvas,
        artifacts: complete ? planArtifact(ticket) : [],
        appId: ticket.appId,
      }),
      ticketAnalysisPhase({
        id: "score",
        name: "Score",
        status: view.laterStatus,
        duration: complete ? ANALYSIS_PHASE_DURATION.score : "—",
        componentName: "Score",
        detail: view.scoreDetail,
        canvas,
        checks,
        appId: ticket.appId,
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
  const score = intakeTicketConfidenceScore(ticket);
  if (score == null) {
    return [];
  }
  return [
    {
      id: `${ticket.id}-confidence`,
      name: CONFIDENCE_CHECK_NAME,
      score,
      maxScore: CONFIDENCE_SCORE_MAX,
      format: "fraction",
      level: confidenceCheckLevel(score),
      summary: ticket.confidenceSummary ?? confidenceSuitabilitySummary(confidenceBandForScore(score)),
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
  appId,
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
  appId?: string;
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
    canvasSteps: [],
    appId,
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

export function intakeAutomationFixture(source: LineIntakeSource, appId?: string): SplitRunFixture {
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
    currentStepIndex: 0,
    waitingNotes,
    checks: [],
    // An always-on automation, not a line run — no line actions in the footer.
    footer: {
      kind: "running",
      sentence: "",
      actions: [],
      note: { headline: waitingNotes[0].headline, text: waitingNotes[0].text },
    },
    footerTone: "running",
    phases: [
      listenPhase(source, canvas, appId),
      evaluatePhase(source, canvas, appId),
      backlogPhase(source, canvas, appId),
    ],
  };
}

function listenPhase(source: LineIntakeSource, canvas: SplitRunCanvasModel, appId?: string): SplitRunPhase {
  return {
    id: "listen",
    name: "Listen",
    status: "passed",
    duration: "—",
    componentName: source.listen.label,
    artifacts: [],
    canvas,
    canvasSteps: [],
    appId,
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

function evaluatePhase(source: LineIntakeSource, canvas: SplitRunCanvasModel, appId?: string): SplitRunPhase {
  return {
    id: "evaluate",
    name: "Evaluate",
    status: "running",
    duration: "—",
    componentName: source.evaluate.label,
    artifacts: [],
    canvas,
    canvasSteps: [],
    appId,
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

function backlogPhase(source: LineIntakeSource, canvas: SplitRunCanvasModel, appId?: string): SplitRunPhase {
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
    canvasSteps: [],
    appId,
    stream,
  };
}
