import {
  ComponentBaseMapper,
  ComponentBaseContext,
  SubtitleContext,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
} from "../types";
import { getState, getStateMap, getTriggerRenderer } from "..";
import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import snykIcon from "@/assets/icons/integrations/snyk.svg";
import { formatTimeAgo } from "@/utils/date";

interface IgnoreIssueMetadata {
  projectId: string;
  issueId: string;
}

interface IgnoreIssueOutput {
  success: boolean;
  message: string;
  projectId: string;
  issueId: string;
  reason: string;
}

const COMPONENT_NAME = "snyk.ignoreIssue";

export const ignoreIssueMapper: ComponentBaseMapper = {
  props: (context: ComponentBaseContext): ComponentBaseProps => {
    const { node } = context;
    const metadata = node.metadata as IgnoreIssueMetadata;
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;

    const metadataItems = [];

    if (metadata?.projectId) {
      metadataItems.push({
        icon: "project",
        label: metadata.projectId.substring(0, 8),
      });
    }

    if (metadata?.issueId) {
      metadataItems.push({
        icon: "bug",
        label: metadata.issueId,
      });
    }

    return {
      title: node.name || "Ignore Issue",
      iconSrc: snykIcon,
      collapsed: node.isCollapsed,
      metadata: metadataItems,
      eventSections: lastExecution ? buildEventSections(context.nodes, lastExecution) : undefined,
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(COMPONENT_NAME),
    };
  },

  subtitle: (context: SubtitleContext): string => {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? formatTimeAgo(new Date(timestamp)) : "";
  },

  getExecutionDetails: (context: ExecutionDetailsContext): Record<string, unknown> => {
    const { execution } = context;
    const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
    const output = outputs?.default?.[0]?.data as IgnoreIssueOutput | undefined;
    const details: Record<string, unknown> = {};

    details["Started At"] = new Date(execution.createdAt).toLocaleString();

    if (output) {
      if (output.message) {
        details["Result"] = output.message;
      }

      if (output.projectId) {
        details["Project"] = output.projectId;
      }

      if (output.issueId) {
        details["Issue"] = output.issueId;
      }

      if (output.reason) {
        details["Reason"] = output.reason;
      }
    }

    return details;
  },
};

function buildEventSections(nodes: NodeInfo[], execution: ExecutionInfo): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName ?? "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });
  const subtitleTimestamp = execution.updatedAt || execution.createdAt;
  const eventSubtitle = subtitleTimestamp ? formatTimeAgo(new Date(subtitleTimestamp)) : "";

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle,
      eventState: getState(COMPONENT_NAME)(execution),
      eventId: execution.rootEvent!.id!,
    },
  ];
}

export default ignoreIssueMapper;
