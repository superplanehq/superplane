import type { FactoriesFactoryPullRequest, FactoriesWorkOrder, FactoriesWorkOrderArtifact } from "@/api-client";

import { implementationPlanMarkdown } from "../onboarding/first-run/reviewCandidateModel";
import { doneFooterForStatus } from "./splitRunFooter";
import type { SplitRunFixture, SplitRunPhase, SplitRunStreamLine } from "./splitRunMocks";

const NOTIFY_BRANCH = "fix/bug-not-getting-notified-for-status-change-when-re-1787246840-4193b6d9";
const NOTIFY_PR_URL = "https://github.com/superplanehq/superplane/pull/6837";

function markdownArtifact(id: string, name: string, body: string): FactoriesWorkOrderArtifact {
  return {
    id,
    type: "TYPE_MARKDOWN",
    data: { name, title: name, body },
  };
}

function notifyPlanMarkdown(order: FactoriesWorkOrder): string {
  return implementationPlanMarkdown({
    goal: order.title ?? "Send a notification when a work-order status changes after a reopen.",
    files: ["pkg/workers/work_order_status.go", "web_src/src/pages/factories/lib/workOrderStatusNote.ts"],
    steps: [
      "Find the reopen path that updates work-order status.",
      "Send the same status-change notification that other status updates send.",
      "Cover the reopen path with a regression test.",
    ],
    verify: ["A reopen sends one status-change notification.", "The existing notification suite passes."],
  });
}

function notifyBranchArtifact(orderId: string): FactoriesWorkOrderArtifact {
  return {
    id: `art-branch-${orderId}`,
    type: "TYPE_BRANCH",
    data: { name: NOTIFY_BRANCH },
  };
}

function notifyPullRequest(orderId: string): FactoriesFactoryPullRequest {
  return {
    id: `pr-${orderId}`,
    workOrderId: orderId,
    number: "6837",
    url: NOTIFY_PR_URL,
    title: "Notify on status change after a reopen",
    state: "STATE_MERGED",
  };
}

function orderRun(order: FactoriesWorkOrder): { appId: string; runId: string } | undefined {
  const run = order.lineDispatches?.[0]?.stepExecutions?.[0]?.run;
  const appId = run?.appId;
  const runId = run?.id;
  if (!appId || !runId) {
    return undefined;
  }
  return { appId, runId };
}

function passedPhase(input: {
  id: string;
  name: string;
  componentName: string;
  duration: string;
  artifacts?: FactoriesWorkOrderArtifact[];
  stream?: SplitRunStreamLine[];
  appId?: string;
  runId?: string;
}): SplitRunPhase {
  return {
    id: input.id,
    name: input.name,
    status: "passed",
    duration: input.duration,
    componentName: input.componentName,
    artifacts: input.artifacts ?? [],
    stream: input.stream ?? [],
    canvasSteps: [],
    canvasKey: null,
    appId: input.appId,
    runId: input.runId,
  };
}

function streamLine(input: {
  id: string;
  at: string;
  componentType: string;
  componentName: string;
  action: string;
  iconSlug: string;
  /** Triggers fire at a point in time, so they carry no duration. */
  duration?: string;
  artifact?: FactoriesWorkOrderArtifact;
  pullRequest?: FactoriesFactoryPullRequest;
}): SplitRunStreamLine {
  return {
    id: input.id,
    nodeId: input.id,
    at: input.at,
    componentName: input.componentName,
    status: "passed",
    kind: input.componentType === "On Run" ? "trigger" : "action",
    componentType: input.componentType,
    action: input.action,
    iconSlug: input.iconSlug,
    duration: input.duration,
    artifact: input.artifact,
    pullRequest: input.pullRequest,
  };
}

function notifyCiLoopStream(): SplitRunStreamLine[] {
  return [
    streamLine({
      id: "ci-on-run",
      at: "19:52:40",
      componentType: "On Run",
      componentName: "CI verification",
      action: "triggered",
      iconSlug: "play",
    }),
    streamLine({
      id: "ci-report-check",
      at: "20:02:50",
      componentType: "Report Work Order Check",
      componentName: "Report CI Check",
      action: "passed",
      iconSlug: "factory",
      duration: "1s",
    }),
    streamLine({
      id: "ci-mark-pr-ready",
      at: "20:02:50",
      componentType: "github.markPullRequestReadyForReview",
      componentName: "Mark Pull Request Ready",
      action: "passed",
      iconSlug: "box",
      duration: "2s",
    }),
    streamLine({
      id: "ci-loop",
      at: "19:52:40",
      componentType: "loop",
      componentName: "loop",
      action: "passed",
      iconSlug: "box",
      duration: "10m 9s",
    }),
    streamLine({
      id: "ci-run-workflow",
      at: "19:52:40",
      componentType: "semaphore.runWorkflow",
      componentName: "Run Semaphore CI",
      action: "passed",
      iconSlug: "box",
      duration: "9m 47s",
    }),
  ];
}

function notifyUiPreviewStream(): SplitRunStreamLine[] {
  return [
    streamLine({
      id: "preview-on-run",
      at: "20:03:22",
      componentType: "On Run",
      componentName: "Start",
      action: "triggered",
      iconSlug: "play",
    }),
    streamLine({
      id: "preview-detect-ui",
      at: "20:03:22",
      componentType: "Run Bash",
      componentName: "Detect UI Changes",
      action: "passed",
      iconSlug: "code",
      duration: "1s",
    }),
    streamLine({
      id: "preview-has-ui-changes",
      at: "20:03:23",
      componentType: "If",
      componentName: "Has UI changes?",
      action: "passed",
      iconSlug: "split",
      duration: "1s",
    }),
    streamLine({
      id: "preview-assess-coverage",
      at: "20:03:23",
      componentType: "Run Claude Code",
      componentName: "Assess Storybook Coverage",
      action: "passed",
      iconSlug: "code",
      duration: "34s",
    }),
    streamLine({
      id: "preview-format-review",
      at: "20:03:57",
      componentType: "Run JavaScript",
      componentName: "Format Coverage Review",
      action: "passed",
      iconSlug: "code",
      duration: "1s",
    }),
    streamLine({
      id: "preview-deploy-storybook",
      at: "20:03:23",
      componentType: "Run Bash",
      componentName: "Deploy Storybook",
      action: "passed",
      iconSlug: "code",
      duration: "48s",
    }),
    streamLine({
      id: "preview-report-coverage",
      at: "20:03:58",
      componentType: "Report Work Order Check",
      componentName: "Report Coverage Check",
      action: "passed",
      iconSlug: "factory",
      duration: "1s",
    }),
    streamLine({
      id: "preview-update-pr",
      at: "20:04:46",
      componentType: "github.updatePullRequest",
      componentName: "Update PR with preview links",
      action: "passed",
      iconSlug: "box",
      duration: "2s",
    }),
  ];
}

function notifyPrCreationStream(pr: FactoriesFactoryPullRequest): SplitRunStreamLine[] {
  return [
    streamLine({
      id: "onrun-onrun-otn0e9",
      at: "19:51:16",
      componentType: "On Run",
      componentName: "Create",
      action: "triggered",
      iconSlug: "play",
    }),
    streamLine({
      id: "component-node-39xj5a",
      at: "19:51:16",
      componentType: "Filter",
      componentName: "PR does not exist?",
      action: "passed",
      iconSlug: "funnel",
      duration: "1s",
    }),
    streamLine({
      id: "runnerclaudecode-runnerclaudecode-7ei2ul",
      at: "19:51:17",
      componentType: "Run Claude Code",
      componentName: "Generate PR title and description",
      action: "passed",
      iconSlug: "code",
      duration: "1m 20s",
    }),
    streamLine({
      id: "github-createpullrequest-github-createpullrequest-z7g0h5",
      at: "19:52:37",
      componentType: "github.createPullRequest",
      componentName: "Create Draft Pull Request",
      action: "passed",
      iconSlug: "github",
      duration: "1s",
    }),
    streamLine({
      id: "component-node-5nu6l9",
      at: "19:52:38",
      componentType: "github.addIssueLabel",
      componentName: "Add Label to Pull Request",
      action: "passed",
      iconSlug: "github",
      duration: "1s",
    }),
    streamLine({
      id: "component-node-f069ua",
      at: "19:52:39",
      componentType: "Add Pull Request",
      componentName: "Attach PR to Work Order",
      action: "passed",
      iconSlug: "factory",
      duration: "1s",
      pullRequest: pr,
    }),
    streamLine({
      id: "set-pr-closure-note",
      at: "19:52:39",
      componentType: "setWorkOrderStatusNote",
      componentName: "Set PR closure note",
      action: "passed",
      iconSlug: "box",
      duration: "1s",
    }),
  ];
}

export function notifyImplementLogPhases(order: FactoriesWorkOrder): SplitRunPhase[] {
  const orderId = order.id ?? "notify";
  const description = markdownArtifact(`art-description-${orderId}`, "description.md", order.description ?? "");
  const plan = markdownArtifact(`art-plan-${orderId}`, "PLAN.md", notifyPlanMarkdown(order));
  const branch = notifyBranchArtifact(orderId);
  const pullRequest = notifyPullRequest(orderId);
  const run = orderRun(order);

  return [
    passedPhase({
      id: "backlog",
      name: "Backlog",
      componentName: "Created manually",
      duration: "2s",
      artifacts: [description],
    }),
    passedPhase({
      id: "planning-0",
      name: "Plan",
      componentName: "Planning",
      duration: "2m 59s",
      artifacts: [plan],
    }),
    passedPhase({
      id: "implementation-1",
      name: "Implement",
      componentName: "Implementation",
      duration: "23m 56s",
      artifacts: [branch],
      appId: run?.appId,
      runId: run?.runId,
    }),
    passedPhase({
      id: "pr-creation-2",
      name: "PR Creation",
      componentName: "PR Creation",
      duration: "1m 23s",
      stream: notifyPrCreationStream(pullRequest),
    }),
    passedPhase({
      id: "ci-loop-3",
      name: "Verify",
      componentName: "Risk Assessment",
      duration: "10m 12s",
      stream: notifyCiLoopStream(),
    }),
    passedPhase({
      id: "risk-assessment-4",
      name: "Verify",
      componentName: "Risk Assessment",
      duration: "29s",
    }),
    passedPhase({
      id: "ui-preview-storybook-coverage-5",
      name: "UI Preview & Storybook Coverage",
      componentName: "UI Preview & Storybook Coverage",
      duration: "1m 26s",
      stream: notifyUiPreviewStream(),
    }),
  ];
}

export function withNotifyImplementLog(fixture: SplitRunFixture, order: FactoriesWorkOrder): SplitRunFixture {
  return {
    ...fixture,
    elapsed: fixture.elapsed.replace(/\s+so far$/i, "").trim() || fixture.elapsed,
    lineStatus: "passed",
    currentPhaseId: "pr-creation-2",
    openPhaseId: "pr-creation-2",
    phases: notifyImplementLogPhases(order),
    waitingNotes: [],
    checks: [],
    footer: { ...doneFooterForStatus("completed"), run: fixture.footer.run ?? orderRun(order) },
    footerTone: "done",
  };
}
