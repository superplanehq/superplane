import datadogIcon from "@/assets/icons/integrations/datadog.svg";
import githubIcon from "@/assets/icons/integrations/github.svg";
import jiraIcon from "@/assets/icons/integrations/jira.svg";
import pagerdutyIcon from "@/assets/icons/integrations/pagerduty.svg";
import productiveIcon from "@/assets/icons/integrations/productive.svg";
import sentryIcon from "@/assets/icons/integrations/sentry.svg";

import type { FactoriesFactoryPullRequest, FactoriesWorkOrder } from "@/api-client";
import { ANALYSIS_ENTRY, PR_CLOSURE_ENTRY, prFeedbackSentence } from "../lib/columnAutomationCatalog";
import type { ColumnAutomation, ColumnAutomationKind } from "../lib/columnAutomations";
import { buildAssigneeFilterOptions, buildSourceFilterOptions } from "../lib/workOrderFilterOptions";
import { applyWorkOrderFilters, applyWorkOrderSearch, buildWorkOrderListEntry } from "../lib/workOrderListModel";
import type { WorkOrderFilters } from "../lib/workOrderListModel";
import { lineIntakeSourceById } from "./lineIntakeModel";

export interface PublicAutomation {
  id: string;
  kind: string;
  name: string;
  catalogId?: string;
  icon?: string;
  health?: string;
}

export interface PublicBoardCard {
  id?: string;
  title: string;
  createdAt: string;
  state?: string;
  result?: string;
  running?: boolean;
  stopped?: boolean;
  failed?: boolean;
  approval?: boolean;
  origin?: { url?: string; label?: string };
  assignee?: { name?: string; avatarUrl?: string };
  confidence?: number;
  clarity?: number;
  pullRequest?: { number: number; state: string; mergeable: boolean; extraCount: number };
  agentQuestion?: boolean;
}

export interface PublicBoardColumn {
  key: string;
  title: string;
  color?: string;
  automations?: PublicAutomation[];
  cards: PublicBoardCard[];
}

export interface PublicBoard {
  workspaceName: string;
  workspaceKey?: string;
  lineName: string;
  showClarity: boolean;
  showConfidence: boolean;
  columns: PublicBoardColumn[];
}

export type BoardLoad =
  | { status: "loading" }
  | { status: "ready"; board: PublicBoard }
  | { status: "denied" }
  | { status: "missing" };

const PUBLIC_AUTOMATION_KINDS = new Set<ColumnAutomationKind>([
  "intake",
  "analysis",
  "agent-step",
  "custom",
  "pr-discussion",
  "pr-checks",
  "pr-closure",
  "risk-score",
]);

const PUBLIC_AUTOMATION_ICONS: Record<string, { src: string; alt: string }> = {
  github: { src: githubIcon, alt: "GitHub" },
  sentry: { src: sentryIcon, alt: "Sentry" },
  jira: { src: jiraIcon, alt: "Jira" },
  pagerduty: { src: pagerdutyIcon, alt: "PagerDuty" },
  productive: { src: productiveIcon, alt: "Productive" },
  datadog: { src: datadogIcon, alt: "Datadog" },
};

export function columnAutomations(column: PublicBoardColumn): ColumnAutomation[] {
  return (column.automations ?? []).map(toColumnAutomation);
}

export function visibleBoardColumns(
  columns: PublicBoardColumn[],
  filters: WorkOrderFilters,
  search: string,
): PublicBoardColumn[] {
  return columns.map((column) => ({
    ...column,
    cards: column.cards.filter((card, index) => cardMatches(card, `${column.key}-${index}`, filters, search)),
  }));
}

export function boardFilterOptions(columns: PublicBoardColumn[], workspaceKey: string | undefined) {
  const entries = columns.flatMap((column) =>
    column.cards.map((card, index) =>
      buildWorkOrderListEntry(publicBoardOrder(card, `${column.key}-${index}`), { key: workspaceKey }),
    ),
  );
  return {
    sources: buildSourceFilterOptions([], entries),
    assignees: buildAssigneeFilterOptions(entries),
  };
}

export function publicBoardOrder(card: PublicBoardCard, id: string): FactoriesWorkOrder {
  return {
    id: card.id || id,
    title: card.title,
    state: (card.state ?? "STATE_OPEN") as FactoriesWorkOrder["state"],
    result: card.result as FactoriesWorkOrder["result"],
    createdAt: card.createdAt,
    updatedAt: card.createdAt,
    origin: card.origin?.url ? { url: card.origin.url, label: card.origin.label } : undefined,
    assignees: card.assignee?.name
      ? [{ id: `${id}-member`, name: card.assignee.name, avatarUrl: card.assignee.avatarUrl }]
      : [],
    pullRequests: publicPullRequests(card, id),
    lineDispatches: publicLineDispatches(card),
    statusNotes: card.approval ? [{ key: "approval", headline: "Waiting for approval" }] : undefined,
  } as unknown as FactoriesWorkOrder;
}

export function publicBoardUrl(organizationId: string, factoryKey: string, lineId: string): string {
  return `/api/v1/public/organizations/${encodeURIComponent(organizationId)}/workspaces/${encodeURIComponent(factoryKey)}/lines/${encodeURIComponent(lineId)}/board`;
}

export function publicBoardSocketPath(organizationId: string, factoryKey: string, lineId: string): string {
  return `/ws/public/organizations/${encodeURIComponent(organizationId)}/workspaces/${encodeURIComponent(factoryKey)}/lines/${encodeURIComponent(lineId)}`;
}

export function parseBoardEvent(data: unknown): string {
  if (typeof data !== "string") {
    return "";
  }
  try {
    const message = JSON.parse(data) as { event?: string };
    return message.event ?? "";
  } catch {
    return "";
  }
}

export function loadStateForErrorStatus(status: number, signedIn: boolean): BoardLoad | undefined {
  if (status === 404) {
    return { status: signedIn ? "denied" : "missing" };
  }
  if (status < 200 || status >= 300) {
    return { status: "missing" };
  }
  return undefined;
}

export function columnEmptyDescription(column: PublicBoardColumn, narrowed: boolean): string {
  if (narrowed) {
    return "No matching tasks.";
  }
  if (column.key === "backlog") {
    return "No tasks in the backlog.";
  }
  if (column.key === "verify") {
    return "No tasks in Verify.";
  }
  if (column.key === "done") {
    return "No tasks in Done.";
  }
  return "Nothing here.";
}

function cardMatches(card: PublicBoardCard, id: string, filters: WorkOrderFilters, search: string): boolean {
  const entry = buildWorkOrderListEntry(publicBoardOrder(card, id), undefined);
  return applyWorkOrderSearch(applyWorkOrderFilters([entry], filters), search).length > 0;
}

function toColumnAutomation(item: PublicAutomation): ColumnAutomation {
  const kind = automationKind(item.kind);
  const source = kind === "intake" ? lineIntakeSourceById(item.catalogId ?? "") : undefined;
  const copy = automationCopy(kind, item.name);
  const icon = automationIcon(kind, item.icon, source);
  return {
    id: item.id,
    kind,
    name: item.name,
    trigger: source?.listen.label ?? copy.trigger,
    action: source?.accept.label ?? copy.action,
    iconSrc: icon.src,
    iconAlt: icon.alt,
    health: item.health === "disabled" ? "disabled" : "healthy",
    runningCount: 0,
    catalogId: item.catalogId || kind,
  };
}

function automationKind(kind: string): ColumnAutomationKind {
  if (PUBLIC_AUTOMATION_KINDS.has(kind as ColumnAutomationKind)) {
    return kind as ColumnAutomationKind;
  }
  return "custom";
}

function automationIcon(
  kind: ColumnAutomationKind,
  icon: string | undefined,
  source: ReturnType<typeof lineIntakeSourceById>,
): { src: string; alt: string } {
  if (source) {
    return { src: source.iconSrc, alt: source.iconAlt };
  }
  if (kind === "pr-closure" || kind === "risk-score") {
    return { src: PR_CLOSURE_ENTRY.iconSrc, alt: PR_CLOSURE_ENTRY.iconAlt };
  }
  return PUBLIC_AUTOMATION_ICONS[icon ?? ""] ?? { src: "", alt: "" };
}

function automationCopy(kind: ColumnAutomationKind, name: string): { trigger: string; action: string } {
  if (kind === "analysis") {
    return { trigger: ANALYSIS_ENTRY.trigger, action: ANALYSIS_ENTRY.action };
  }
  if (kind === "agent-step") {
    return { trigger: `On task in ${name}`, action: `Run the ${name} agent` };
  }
  if (kind === "pr-closure") {
    return { trigger: PR_CLOSURE_ENTRY.trigger, action: PR_CLOSURE_ENTRY.action };
  }
  if (kind === "pr-checks") {
    return prFeedbackSentence("checks");
  }
  if (kind === "pr-discussion") {
    return prFeedbackSentence("discussion");
  }
  return { trigger: "", action: "" };
}

function publicPullRequests(card: PublicBoardCard, id: string): FactoriesWorkOrder["pullRequests"] {
  if (!card.pullRequest) {
    return undefined;
  }
  return [
    {
      workOrderId: card.id || id,
      number: String(card.pullRequest.number),
      state: card.pullRequest.state as FactoriesFactoryPullRequest["state"],
      mergeable: card.pullRequest.mergeable,
    },
  ];
}

function publicLineDispatches(card: PublicBoardCard): FactoriesWorkOrder["lineDispatches"] {
  if (card.running) {
    return [{ state: "STATE_ACTIVE" }];
  }
  const execution = finishedExecution(card);
  if (!execution) {
    return [];
  }
  return [{ state: "STATE_FINISHED", stepExecutions: [execution] }];
}

function finishedExecution(card: PublicBoardCard) {
  if (card.failed) {
    return { result: "RESULT_FAILED" as const, updatedAt: card.createdAt };
  }
  if (card.stopped) {
    return { result: "RESULT_CANCELLED" as const, updatedAt: card.createdAt };
  }
  return undefined;
}
