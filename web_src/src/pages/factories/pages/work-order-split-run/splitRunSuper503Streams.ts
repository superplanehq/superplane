import type { FactoriesFactoryPullRequest, FactoriesWorkOrderArtifact } from "@/api-client";

import { YESTERDAY } from "../../__fixtures__/factoryPageIds";
import { claudeLogToStreamNotes } from "./claudeLogToStreamNotes";
import { clockLabel } from "./splitRunFormat";
import type { SplitRunPhaseStatus, SplitRunStreamLine } from "./splitRunMocks";
import analysisLog from "./super503/analysis-log.txt?raw";
import implementLog from "./super503/implement-log.txt?raw";
import review1Log from "./super503/review-1-log.txt?raw";
import review2Log from "./super503/review-2-log.txt?raw";
import review3Log from "./super503/review-3-log.txt?raw";
import review4Log from "./super503/review-4-log.txt?raw";
import review5Log from "./super503/review-5-log.txt?raw";

/**
 * Stream lines for the SUPER-503 Storybook fixture. Node names follow the
 * canvas runs on "Superplane Steam Machine". Transcripts are the real
 * runner logs.
 */

export const SUPER503_MODEL = "openrouter/x-ai/grok-4.6";

const AGENT_COMPONENT_TYPE = "Run OpenRouter Agent";
const AGENT_COMPONENT = "runnerOpenRouter";

/** Maps a real UTC clock from 2026-09-23 onto yesterday, so relative labels stay stable. */
export function super503Time(clock: string): string {
  const day = YESTERDAY.slice(0, 10);
  return new Date(`${day}T${clock}Z`).toISOString();
}

export const SUPER503_REVIEW_LOGS = {
  review1: review1Log,
  review2: review2Log,
  review3: review3Log,
  review4: review4Log,
  review5: review5Log,
} as const;

export type Super503ReviewLogKey = keyof typeof SUPER503_REVIEW_LOGS;

const ANALYSIS_STEPS = [
  { name: "Clone repository", type: "bash" },
  { name: "Refine Task", type: "prompt" },
  { name: "Wait for the next message", type: "prompt" },
  {
    name: "I actually don't think we should have this transcript preview at the bottom, just add the text directly in the input (agent chat and create task form) what's the point of generating that text bellow and then inserting it in the input, just do that generation in input directly.",
    type: "prompt",
  },
];

const IMPLEMENT_STEPS = [
  { name: "Clone Repo", type: "bash" },
  { name: "Implementation", type: "prompt" },
  { name: "Commit and Push", type: "bash" },
  { name: "Generate PR title and description", type: "prompt" },
  { name: "Push output", type: "bash" },
];

const REVIEW_STEPS = [
  { name: "Set Up Git User", type: "bash" },
  { name: "Checkout Pull Request", type: "bash" },
  { name: "Set Up DCO Signing", type: "bash" },
  { name: "Address PR feedback", type: "prompt" },
  { name: "Commit and Push", type: "bash" },
];

function streamLine(input: {
  id: string;
  at: string;
  componentType: string;
  componentName: string;
  iconSlug: string;
  duration?: string;
  component?: string;
  artifact?: FactoriesWorkOrderArtifact;
  pullRequest?: FactoriesFactoryPullRequest;
}): SplitRunStreamLine {
  const status: SplitRunPhaseStatus = "passed";
  const trigger = input.componentType.startsWith("On ");
  return {
    id: input.id,
    nodeId: input.id,
    at: clockLabel(super503Time(input.at)),
    componentName: input.componentName,
    status,
    kind: trigger ? "trigger" : "action",
    componentType: input.componentType,
    action: trigger ? "triggered" : status,
    iconSlug: input.iconSlug,
    duration: input.duration,
    component: input.component,
    artifact: input.artifact,
    pullRequest: input.pullRequest,
  };
}

export function super503BacklogStream(description: FactoriesWorkOrderArtifact): SplitRunStreamLine[] {
  return [
    streamLine({
      id: "backlog-created",
      at: "11:28:53",
      componentType: "Create Task",
      componentName: "Aleksandar Mitrovic created this task",
      iconSlug: "user",
      duration: "1s",
      artifact: description,
    }),
  ];
}

export function super503AnalysisStream(plan: FactoriesWorkOrderArtifact): SplitRunStreamLine[] {
  const agentNodeId = "analysis-refine-task";
  return [
    streamLine({
      id: "analysis-on-draft",
      at: "11:29:06",
      componentType: "On Task",
      componentName: "On Draft",
      iconSlug: "play",
    }),
    streamLine({
      id: agentNodeId,
      at: "11:29:06",
      componentType: AGENT_COMPONENT_TYPE,
      componentName: "Refine Task",
      iconSlug: "code",
      duration: "4m 8s",
      component: AGENT_COMPONENT,
      artifact: plan,
    }),
    ...claudeLogToStreamNotes(agentNodeId, analysisLog, ANALYSIS_STEPS),
  ];
}

export function super503ImplementStream(
  branch: FactoriesWorkOrderArtifact,
  pullRequest: FactoriesFactoryPullRequest,
): SplitRunStreamLine[] {
  const agentNodeId = "implementation-agent-no-issue";
  return [
    streamLine({
      id: "onrun-implement",
      at: "11:37:08",
      componentType: "On Run",
      componentName: "Start Implementation",
      iconSlug: "play",
    }),
    streamLine({
      id: agentNodeId,
      at: "11:37:09",
      componentType: AGENT_COMPONENT_TYPE,
      componentName: "Implementation Agent",
      iconSlug: "code",
      duration: "19m 44s",
      component: AGENT_COMPONENT,
    }),
    ...claudeLogToStreamNotes(agentNodeId, implementLog, IMPLEMENT_STEPS),
    streamLine({
      id: "add-branch-artifact",
      at: "11:56:54",
      componentType: "Add Task Artifact",
      componentName: "Add Branch Artifact",
      iconSlug: "factory",
      duration: "1s",
      artifact: branch,
    }),
    streamLine({
      id: "find-pr",
      at: "11:56:54",
      componentType: "github.findPullRequest",
      componentName: "Find Pull Request",
      iconSlug: "github",
      duration: "1s",
    }),
    streamLine({
      id: "create-pr",
      at: "11:56:55",
      componentType: "github.createPullRequest",
      componentName: "Create Pull Request",
      iconSlug: "github",
      duration: "2s",
      pullRequest,
    }),
    streamLine({
      id: "attach-pr-artifact",
      at: "11:56:57",
      componentType: "Attach Pull Request",
      componentName: "Attach PR to Task",
      iconSlug: "factory",
      duration: "1s",
    }),
    streamLine({
      id: "has-visual-evidence",
      at: "11:56:57",
      componentType: "Filter",
      componentName: "Has Visual Evidence?",
      iconSlug: "funnel",
      duration: "1s",
    }),
    streamLine({
      id: "comment-visual-evidence",
      at: "11:56:57",
      componentType: "Add Pull Request Comment",
      componentName: "Comment Visual Evidence",
      iconSlug: "github",
      duration: "1s",
    }),
  ];
}

export function super503ChecksStream(input: {
  prefix: string;
  at: string;
  waitDuration: string;
}): SplitRunStreamLine[] {
  return [
    streamLine({
      id: `${input.prefix}-on-pull-request`,
      at: input.at,
      componentType: "On Pull Request",
      componentName: "On Pull Request",
      iconSlug: "github",
    }),
    streamLine({
      id: `${input.prefix}-find-pull-request`,
      at: input.at,
      componentType: "github.findPullRequest",
      componentName: "Find Pull Request",
      iconSlug: "github",
      duration: "1s",
    }),
    streamLine({
      id: `${input.prefix}-add-pr-activity`,
      at: input.at,
      componentType: "Add PR Activity",
      componentName: "Add PR Activity",
      iconSlug: "factory",
      duration: "1s",
    }),
    streamLine({
      id: `${input.prefix}-wait-pr-checks`,
      at: input.at,
      componentType: "Wait For Checks",
      componentName: "Wait For Checks",
      iconSlug: "box",
      duration: input.waitDuration,
    }),
    streamLine({
      id: `${input.prefix}-mark-checks-passed`,
      at: input.at,
      componentType: "Add PR Activity",
      componentName: "Mark Checks Passed",
      iconSlug: "factory",
      duration: "1s",
    }),
  ];
}

export function super503StorybookStream(input: {
  prefix: string;
  at: string;
  duration: string;
  link: FactoriesWorkOrderArtifact;
}): SplitRunStreamLine[] {
  return [
    streamLine({
      id: `${input.prefix}-pr-trigger`,
      at: input.at,
      componentType: "On Pull Request",
      componentName: "On Pull Request",
      iconSlug: "github",
    }),
    streamLine({
      id: `${input.prefix}-deploy-storybook`,
      at: input.at,
      componentType: "Run Workflow",
      componentName: "Deploy Storybook",
      iconSlug: "box",
      duration: input.duration,
    }),
    streamLine({
      id: `${input.prefix}-add-link-artifact`,
      at: input.at,
      componentType: "Add Task Artifact",
      componentName: "Add Preview Link",
      iconSlug: "factory",
      duration: "1s",
      artifact: input.link,
    }),
  ];
}

export function super503ReviewStream(input: {
  prefix: string;
  at: string;
  agentDuration: string;
  log: Super503ReviewLogKey;
}): SplitRunStreamLine[] {
  const agentNodeId = `${input.prefix}-address-pr-review-feedback`;
  return [
    streamLine({
      id: `${input.prefix}-on-pr-review`,
      at: input.at,
      componentType: "On Pull Request Review",
      componentName: "On Pull Request Review",
      iconSlug: "github",
    }),
    streamLine({
      id: `${input.prefix}-find-pull-request-for-review`,
      at: input.at,
      componentType: "github.findPullRequest",
      componentName: "Find Pull Request",
      iconSlug: "github",
      duration: "1s",
    }),
    streamLine({
      id: `${input.prefix}-add-pr-review-activity`,
      at: input.at,
      componentType: "Add PR Activity",
      componentName: "Add PR Review Activity",
      iconSlug: "factory",
      duration: "1s",
    }),
    streamLine({
      id: agentNodeId,
      at: input.at,
      componentType: AGENT_COMPONENT_TYPE,
      componentName: "Address PR Review Feedback",
      iconSlug: "code",
      duration: input.agentDuration,
      component: AGENT_COMPONENT,
    }),
    ...claudeLogToStreamNotes(agentNodeId, SUPER503_REVIEW_LOGS[input.log], REVIEW_STEPS),
  ];
}

export function super503ClosureStream(pullRequest: FactoriesFactoryPullRequest): SplitRunStreamLine[] {
  return [
    streamLine({
      id: "closure-on-pr-merged",
      at: "13:11:07",
      componentType: "On Pull Request",
      componentName: "On Pull Request Merged",
      iconSlug: "github",
    }),
    streamLine({
      id: "stamp-pr-merged",
      at: "13:11:09",
      componentType: "PR Closure",
      componentName: "PR Closure",
      iconSlug: "factory",
      duration: "2s",
      pullRequest,
    }),
  ];
}
