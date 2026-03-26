import type { EventSection } from "@/ui/componentBase";
import { formatTimeAgo } from "@/utils/date";
import type { ExecutionInfo, NodeInfo, StateFunction, TriggerRenderer } from "../types";

export function buildEventSections(
  nodes: NodeInfo[],
  execution: ExecutionInfo,
  componentName: string,
  getTriggerRenderer: (name: string) => TriggerRenderer,
  getState: (componentName: string) => StateFunction,
): EventSection[] | undefined {
  const rootEvent = execution.rootEvent;
  const createdAt = execution.createdAt;
  const rootTriggerNode = nodes.find((n) => n.id === rootEvent?.nodeId);
  const rootComponentName = rootTriggerNode?.componentName;

  if (!rootEvent || !createdAt || !rootComponentName) {
    return undefined;
  }

  const rootTriggerRenderer = getTriggerRenderer(rootComponentName);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: rootEvent });

  return [
    {
      receivedAt: new Date(createdAt),
      eventTitle: title,
      eventSubtitle: formatTimeAgo(new Date(createdAt)),
      eventState: getState(componentName)(execution),
      eventId: rootEvent.id || "",
    },
  ];
}

export function splitSentryIssueTitle(title?: string): { title?: string; prefix?: string } {
  if (!title) {
    return {};
  }

  const trimmedTitle = title.trim();
  if (!trimmedTitle) {
    return {};
  }

  const separatorIndex = trimmedTitle.indexOf(":");
  if (separatorIndex <= 0 || separatorIndex >= trimmedTitle.length - 1) {
    return { title: trimmedTitle };
  }

  const prefix = trimmedTitle.slice(0, separatorIndex).trim();
  const suffix = trimmedTitle.slice(separatorIndex + 1).trim();

  if (!prefix || !suffix) {
    return { title: trimmedTitle };
  }

  return {
    title: suffix,
    prefix,
  };
}

export function addDetail(details: Record<string, string>, label: string, value?: string) {
  if (!value) {
    return;
  }

  details[label] = value;
}

export function addFormattedTimestamp(details: Record<string, string>, label: string, value?: string) {
  if (!value) {
    return;
  }

  details[label] = new Date(value).toLocaleString();
}

export interface OrderedDetail {
  label: string;
  value?: string;
  isTimestamp?: boolean;
}

export function addOrderedDetails(details: Record<string, string>, orderedDetails: OrderedDetail[], maxItems = 6) {
  for (const detail of orderedDetails) {
    if (Object.keys(details).length >= maxItems) {
      break;
    }

    if (detail.isTimestamp) {
      addFormattedTimestamp(details, detail.label, detail.value);
      continue;
    }

    addDetail(details, detail.label, detail.value);
  }
}

export function getProjectLabel(issue?: { project?: { name?: string; slug?: string } }) {
  return issue?.project?.name || issue?.project?.slug;
}
