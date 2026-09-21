import type {
  FactoriesFactoryPullRequest,
  FactoriesWorkOrder,
  FactoriesWorkOrderArtifact,
  FactoriesWorkOrderCheck,
} from "@/api-client";

import {
  extractArtifactName,
  extractArtifactTitle,
  extractArtifactUrl,
  toArtifactDataRecord,
} from "../../lib/workOrderArtifact";
import { formatCheckScore, presentWorkOrderChecks } from "../../lib/workOrderChecks";
import { formatWorkOrderDateTime } from "../../lib/workOrderDateTime";
import { deriveFactoryLineRows } from "../../lib/workOrderFactoryLineRows";
import {
  getWorkOrderDisplayKey,
  getWorkOrderDisplayStatus,
  getWorkOrderDisplayStatusMeta,
  type WorkOrderDisplayStatus,
} from "../../lib/workOrderProgress";
import { withoutPullRequestArtifacts } from "../../lib/workOrderPullRequest";
import {
  formatCompactTokens,
  formatDurationSeconds,
  formatUsdCents,
  parseWorkOrderMetric,
} from "../../lib/workOrderUsage";
import { CREATED_MANUALLY, splitRunSourceForOrder } from "../work-order-split-run/splitRunSource";

export type TaskPagePrimaryAction = "start" | "review" | "reopen";

export interface TaskPageSource {
  label: string;
  href?: string;
}

export interface TaskPageCheck {
  id: string;
  name: string;
  scoreLabel: string;
  summary?: string;
}

export interface TaskPageRelatedItem {
  id: string;
  title: string;
  href?: string;
}

export interface TaskPageActivityItem {
  id: string;
  actor: string;
  text: string;
  timeLabel: string;
  kind: "event" | "comment";
}

export interface TaskPageRecord {
  key: string;
  title: string;
  description: string;
  status: WorkOrderDisplayStatus;
  statusLabel: string;
  author: string;
  assignees: string[];
  mission?: string;
  createdLabel: string;
  spendLabel: string;
  source: TaskPageSource;
  factoryLines: string[];
  checks: TaskPageCheck[];
  artifacts: TaskPageRelatedItem[];
  pullRequests: TaskPageRelatedItem[];
  activity: TaskPageActivityItem[];
  primaryAction: TaskPagePrimaryAction | null;
}

export type TaskPageView = { state: "loading" } | { state: "error" } | { state: "ready"; record: TaskPageRecord };

export const TASK_PAGE_COPY = {
  writeUp: "Write-up",
  checks: "Checks",
  artifacts: "Artifacts",
  pullRequests: "Pull requests",
  activity: "Activity",
  commentLabel: "Comment",
  commentPlaceholder: "Write a comment",
  sendComment: "Send comment",
  start: "Start",
  review: "Review",
  reopen: "Reopen",
  noWriteUp: "This task has no write-up yet.",
  noChecks: "No checks yet.",
  noArtifacts: "No artifacts yet.",
  noPullRequests: "No pull requests yet.",
  noActivity: "No activity yet.",
  noOwner: "No owner",
  noSpend: "No spend yet.",
  noLines: "Not run on a line yet.",
  unknownAuthor: "Unknown",
  loading: "Loading task…",
  errorTitle: "SuperPlane could not load this task.",
  errorBody: "Check your connection and try again.",
  retry: "Retry",
  status: "Status",
  author: "Author",
  assignees: "Assignees",
  mission: "Mission",
  created: "Created",
  spend: "Spend",
  source: "Source",
  factoryLines: "Factory lines",
  titleAriaLabel: "Task title",
} as const;

const PRIMARY_ACTION_LABEL: Record<TaskPagePrimaryAction, string> = {
  start: TASK_PAGE_COPY.start,
  review: TASK_PAGE_COPY.review,
  reopen: TASK_PAGE_COPY.reopen,
};

export function primaryActionForStatus(status: WorkOrderDisplayStatus): TaskPagePrimaryAction | null {
  if (status === "draft") {
    return "start";
  }
  if (status === "waiting") {
    return "review";
  }
  if (status === "completed" || status === "failed" || status === "rejected" || status === "cancelled") {
    return "reopen";
  }
  return null;
}

export function primaryActionLabel(action: TaskPagePrimaryAction): string {
  return PRIMARY_ACTION_LABEL[action];
}

export function formatTaskSpendLabel(
  order: Pick<FactoriesWorkOrder, "totalTokens" | "totalCostCents" | "totalDurationSeconds">,
): string {
  const totalTokens = parseWorkOrderMetric(order.totalTokens);
  const totalCostCents = parseWorkOrderMetric(order.totalCostCents);
  const durationSeconds = parseWorkOrderMetric(order.totalDurationSeconds);
  const parts: string[] = [];
  if (totalCostCents > 0) {
    parts.push(formatUsdCents(totalCostCents));
  }
  if (totalTokens > 0) {
    parts.push(formatCompactTokens(totalTokens));
  }
  if (durationSeconds > 0) {
    parts.push(formatDurationSeconds(durationSeconds));
  }
  return parts.join(" · ");
}

export function buildTaskPageRecord(input: {
  order: FactoriesWorkOrder;
  factoryKey: string;
  missionName?: string;
  checks?: FactoriesWorkOrderCheck[];
  artifacts?: FactoriesWorkOrderArtifact[];
  pullRequests?: FactoriesFactoryPullRequest[];
  activity?: TaskPageActivityItem[];
}): TaskPageRecord {
  const status = getWorkOrderDisplayStatus(input.order);
  const createdAt = input.order.createdAt ? new Date(input.order.createdAt) : null;

  return {
    key: getWorkOrderDisplayKey(input.order, input.factoryKey),
    title: input.order.title?.trim() || "Task",
    description: input.order.description?.trim() ?? "",
    status,
    statusLabel: getWorkOrderDisplayStatusMeta(status).label,
    author: authorLabel(input.order),
    assignees: (input.order.assignees ?? []).map((assignee) => assignee.name?.trim() || TASK_PAGE_COPY.unknownAuthor),
    mission: input.missionName?.trim() || undefined,
    createdLabel: createdAt ? formatWorkOrderDateTime(createdAt) : "",
    spendLabel: formatTaskSpendLabel(input.order),
    source: sourceFromOrder(input.order),
    factoryLines: deriveFactoryLineRows(input.order.lineDispatches ?? []).map((row) => row.lineName),
    checks: presentWorkOrderChecks(input.checks ?? []).map(toTaskPageCheck),
    artifacts: withoutPullRequestArtifacts(input.artifacts ?? []).map(toArtifactItem),
    pullRequests: (input.pullRequests ?? []).map(toPullRequestItem),
    activity: input.activity ?? [],
    primaryAction: primaryActionForStatus(status),
  };
}

function authorLabel(order: FactoriesWorkOrder): string {
  const automation = order.createdBy?.automation;
  const automationName = automation?.nodeName?.trim() || automation?.appName?.trim();
  if (automationName) {
    return automationName;
  }
  return order.createdBy?.user?.name?.trim() || TASK_PAGE_COPY.unknownAuthor;
}

function sourceFromOrder(order: FactoriesWorkOrder): TaskPageSource {
  const source = splitRunSourceForOrder(order);
  if (source.kind === "intake") {
    return {
      label: source.ticket?.label ?? source.name,
      href: source.ticket?.href,
    };
  }
  return { label: `${source.person.name} · ${CREATED_MANUALLY}` };
}

function toTaskPageCheck(check: ReturnType<typeof presentWorkOrderChecks>[number]): TaskPageCheck {
  const score = formatCheckScore(check);
  return {
    id: check.id,
    name: check.name,
    scoreLabel: `${score.value}${score.scale}`,
    summary: check.summary,
  };
}

function toArtifactItem(artifact: FactoriesWorkOrderArtifact): TaskPageRelatedItem {
  const data = toArtifactDataRecord(artifact.data);
  return {
    id: artifact.id ?? `${artifact.type}-${artifact.createdAt}`,
    title: extractArtifactTitle(data) || extractArtifactName(data) || "Artifact",
    href: extractArtifactUrl(data),
  };
}

function toPullRequestItem(pullRequest: FactoriesFactoryPullRequest): TaskPageRelatedItem {
  const number = pullRequest.number?.trim();
  const title = pullRequest.title?.trim();
  return {
    id: pullRequest.id ?? `${pullRequest.url}-${pullRequest.number}`,
    title: number && title ? `#${number} ${title}` : title || (number ? `#${number}` : "Pull request"),
    href: pullRequest.url,
  };
}
