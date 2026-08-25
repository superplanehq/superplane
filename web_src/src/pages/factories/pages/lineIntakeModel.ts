import type {
  FactoriesFactoryIntake,
  FactoriesFactoryIntakeSource,
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
import type { WorkOrderStatusNotePresentation } from "../lib/workOrderStatusNote";
import { intakeSettingsFromApi, type IntakeSourceSettings } from "./intakeSourceSettingsModel";
import { intakeCanvasForSource } from "./lineIntakeCanvas";
import type { SplitRunCanvasModel } from "./work-order-split-run/splitRunCanvases";
import type { SplitRunFixture, SplitRunPhase, SplitRunStreamLine } from "./work-order-split-run/splitRunMocks";

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
    const sourceId = intake.source ? LINE_INTAKE_SOURCE_ID_BY_API_SOURCE[intake.source] : undefined;
    const source = sourceId ? lineIntakeSourceById(sourceId) : undefined;
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

export interface LineIntakeAnalyzingTicket {
  id: string;
  title: string;
  appId?: string;
  runId?: string;
}

export const LINE_INTAKE_COPY = {
  analyzingTitle: "Analyzing",
  analyzingHelper: "Tickets from this intake.",
  analyzingStatus: "Analyzing",
  analyzingEmpty: "No tickets in analysis.",
  needsRepair: "Needs repair",
  needsRepairHelper: "The automation can no longer create work orders. Open it to repair the steps.",
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
function ticketAnalysisPresentation(complete: boolean) {
  return {
    elapsed: complete ? "4m 12s" : "Running",
    costUsd: complete ? "0.18" : "—",
    lineStatus: complete ? "passed" : "running",
    currentPhaseId: complete ? "score" : "analyze",
    noteKey: complete ? "complete" : "analyzing",
    headline: complete ? LINE_INTAKE_COPY.analysisCompleteHeadline : LINE_INTAKE_COPY.analysisHeadline,
    text: complete ? LINE_INTAKE_COPY.analysisCompleteHelper : LINE_INTAKE_COPY.analysisHelper,
    footerTone: complete ? undefined : ("waiting" as const),
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
  const canvas = ticketAnalysisCanvas();
  const view = ticketAnalysisPresentation(Boolean(options?.complete));
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
    waitingNotes: [
      {
        key: `${ticket.id}-${view.noteKey}`,
        headline: view.headline,
        text: view.text,
      },
    ],
    checks: [],
    footerTone: view.footerTone,
    phases: [
      ticketAnalysisPhase({
        id: "ingest",
        name: "Ingest",
        status: "passed",
        componentName: "Ingest",
        detail: "Ticket received from GitHub issues.",
        canvas,
      }),
      ticketAnalysisPhase({
        id: "analyze",
        name: "Analyze",
        status: view.analyzeStatus,
        componentName: "Analyze ticket",
        detail: view.analyzeDetail,
        canvas,
      }),
      ticketAnalysisPhase({
        id: "plan",
        name: "Create plan",
        status: view.laterStatus,
        componentName: "Create plan",
        detail: view.planDetail,
        canvas,
      }),
      ticketAnalysisPhase({
        id: "score",
        name: "Score",
        status: view.laterStatus,
        componentName: "Score",
        detail: view.scoreDetail,
        canvas,
      }),
    ],
  };
}

function ticketAnalysisPhase({
  id,
  name,
  status,
  componentName,
  detail,
  canvas,
}: {
  id: SplitRunPhase["id"];
  name: string;
  status: SplitRunPhase["status"];
  componentName: string;
  detail: string;
  canvas: SplitRunCanvasModel;
}): SplitRunPhase {
  return {
    id,
    name,
    status,
    duration: "—",
    componentName,
    artifacts: [],
    canvas,
    canvasSteps: [],
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

function ticketAnalysisCanvas(): SplitRunCanvasModel {
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
      [analyzeId]: "running",
      [planId]: "pending",
      [scoreId]: "pending",
    } satisfies Record<string, FactoryNodeStatus>,
    metrics: {},
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
    canvasSteps: [],
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
    canvasSteps: [],
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
    canvasSteps: [],
    stream,
  };
}
