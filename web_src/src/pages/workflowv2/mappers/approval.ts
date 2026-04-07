import type { SuperplaneUsersUser, CanvasesCanvasNodeExecution } from "@/api-client";
import { canvasesInvokeNodeExecutionAction } from "@/api-client";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  EventStateRegistry,
  ExecutionDetailsContext,
  ExecutionInfo,
  GroupRef,
  NodeInfo,
  RoleRef,
  StateFunction,
  SubtitleContext,
  User,
} from "./types";
import type {
  ComponentBaseProps,
  ComponentBaseSpec,
  EventSection,
  EventState,
  EventStateMap,
} from "@/ui/componentBase";
import { DEFAULT_EVENT_STATE_MAP } from "@/ui/componentBase";
import { getTriggerRenderer } from ".";
import { getBackgroundColorClass, getColorClass } from "@/lib/colors";
import { ApprovalGroup } from "@/ui/approvalGroup";
import React from "react";
import type { QueryClient } from "@tanstack/react-query";
import { organizationKeys } from "@/hooks/useOrganizationData";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { canvasKeys } from "@/hooks/useCanvasData";
import { renderTimeAgo, renderWithTimeAgo } from "@/components/TimeAgo";
import { showErrorToast } from "@/lib/toast";
import { formatRelativeTime } from "@/lib/timezone";
import type { ApprovalItemProps } from "@/ui/approvalGroup/ApprovalItem";

type Metadata = {
  records: ApprovalRecord[];
};

type ExecutionMetadata = {
  result: string;
  records: ApprovalRecord[];
};

type ApprovalRecord = {
  index: number;
  state: string;
  type: string;
  user?: User;
  roleRef?: RoleRef;
  groupRef?: GroupRef;
  approval?: ApprovalDetail;
  rejection?: RejectionDetail;
};

type ApprovalDetail = {
  approvedAt?: string;
  comment?: string;
};

type RejectionDetail = {
  rejectedAt?: string;
  reason?: string;
};

export const APPROVAL_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  waiting: {
    icon: "clock",
    textColor: "text-gray-800",
    backgroundColor: "bg-orange-100",
    badgeColor: "bg-yellow-600",
  },
  approved: {
    icon: "circle-check",
    textColor: "text-gray-800",
    backgroundColor: "bg-green-100",
    badgeColor: "bg-emerald-500",
  },
  rejected: {
    icon: "circle-x",
    textColor: "text-gray-800",
    backgroundColor: "bg-red-100",
    badgeColor: "bg-red-400",
  },
  error: {
    icon: "triangle-alert",
    textColor: "text-gray-800",
    backgroundColor: "bg-red-100",
    badgeColor: "bg-red-400",
  },
  running: {
    icon: "clock",
    textColor: "text-gray-800",
    backgroundColor: "bg-amber-100",
    badgeColor: "bg-orange-500",
  },
};

/**
 * Approval-specific state logic function
 */
export const approvalStateFunction: StateFunction = (execution: CanvasesCanvasNodeExecution): EventState => {
  if (
    execution.resultMessage &&
    (execution.resultReason === "RESULT_REASON_ERROR" ||
      (execution.result === "RESULT_FAILED" && execution.resultReason !== "RESULT_REASON_ERROR_RESOLVED"))
  ) {
    return "error";
  }

  if (execution.result === "RESULT_CANCELLED") {
    return "cancelled";
  }

  // Error state - component could not evaluate or apply approval logic
  if (execution.state === "STATE_FINISHED" && execution.result === "RESULT_FAILED") {
    return "error";
  }

  // Waiting state - some or all required actors have not yet responded
  if (execution.state === "STATE_PENDING" || execution.state === "STATE_STARTED") {
    return "waiting";
  }

  // Check execution outputs for approval/rejection decision
  if (execution.state === "STATE_FINISHED" && execution.result === "RESULT_PASSED") {
    const metadata = execution.metadata as ExecutionMetadata | undefined;
    if (metadata?.result === "approved") {
      return "approved";
    }

    if (metadata?.result === "rejected") {
      return "rejected";
    }

    // Default to success if finished and passed but no specific result
    return "approved";
  }

  // Default fallback
  return "error";
};

/**
 * Approval-specific state registry
 */
export const APPROVAL_STATE_REGISTRY: EventStateRegistry = {
  stateMap: APPROVAL_STATE_MAP,
  getState: approvalStateFunction,
};

export const approvalMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const nodeMetadata = context.node.metadata as Metadata | undefined;
    const items = nodeMetadata?.records || [];

    return {
      iconSlug: context.componentDefinition.icon || "hand",
      iconColor: getColorClass("black"),
      collapsedBackground: getBackgroundColorClass("orange"),
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition?.label || "Approval",
      eventSections: lastExecution ? getApprovalEventSections(context.nodes, lastExecution) : undefined,
      includeEmptyState: !lastExecution,
      specs: getApprovalSpecs(items),
      eventStateMap: APPROVAL_STATE_MAP,
      customField: getApprovalCustomField(context),
    };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    return getComponentSubtitle(context.execution);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, any> {
    const details: Record<string, string> = {};
    const metadata = context.execution.metadata as ExecutionMetadata | undefined;

    if (context.execution.createdAt) {
      details["Started at"] = new Date(context.execution.createdAt).toLocaleString();
    }

    if (context.execution.state === "STATE_FINISHED" && context.execution.updatedAt) {
      details["Finished at"] = new Date(context.execution.updatedAt).toLocaleString();
    }

    return withApprovals(details, metadata);
  },
};

function getRecordTypeLabel(record: ApprovalRecord): string {
  switch (record.type) {
    case "role":
      return ` as ${record.roleRef?.displayName || "Role"}`;
    case "group":
      return ` as ${record.groupRef?.displayName || "Group"}`;
    default:
      return "";
  }
}

function getApprovalDetail(detail: ApprovalDetail, record: ApprovalRecord): string {
  if (!detail.approvedAt) {
    return "-";
  }

  let label = `Approved ${formatRelativeTime(detail.approvedAt, true)}` + getRecordTypeLabel(record);
  if (detail.comment) {
    label += ` - ${detail.comment}`;
  }

  return label;
}

function getRejectionDetail(detail: RejectionDetail, record: ApprovalRecord): string {
  if (!detail.rejectedAt) {
    return "-";
  }

  let label = `Rejected ${formatRelativeTime(detail.rejectedAt, true)}` + getRecordTypeLabel(record);
  if (detail.reason) {
    label += ` - ${detail.reason}`;
  }
  return label;
}

function withApprovals(
  details: Record<string, string>,
  metadata: ExecutionMetadata | undefined,
): Record<string, string> {
  if (!metadata) return details;

  details["Result"] = metadata.result;

  for (const record of metadata.records) {
    if (record.approval) {
      const userLabel = record.user?.name || record.user?.email || "User";
      details[userLabel] = getApprovalDetail(record.approval, record);
      continue;
    }

    if (record.rejection) {
      const userLabel = record.user?.name || record.user?.email || "User";
      details[userLabel] = getRejectionDetail(record.rejection, record);
      continue;
    }
  }

  return details;
}

function getApprovalCustomField(context: ComponentBaseContext): React.ReactNode | undefined {
  const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
  if (!lastExecution) {
    return;
  }

  const isAwaitingApproval = ["STATE_STARTED", "STATE_PENDING"].includes(lastExecution?.state || "");
  if (!isAwaitingApproval) {
    return;
  }

  const metadata = lastExecution.metadata as ExecutionMetadata | undefined;
  if (!metadata) {
    return;
  }

  if (metadata.records.length === 0) {
    return;
  }

  return React.createElement(ApprovalGroup, {
    awaitingApproval: isAwaitingApproval,
    approvals: metadata.records.map((record: ApprovalRecord) => {
      return approvalItemPropsForRecord(context, lastExecution, record, isAwaitingApproval);
    }),
  });
}

function approvalItemPropsForRecord(
  context: ComponentBaseContext,
  lastExecution: ExecutionInfo,
  record: ApprovalRecord,
  isAwaitingApproval: boolean,
): ApprovalItemProps {
  const canAct =
    record.state === "pending" &&
    isAwaitingApproval &&
    canCurrentUserActOnApproval(context.queryClient, context.organizationId, record, context.currentUser);

  const title = getApprovalDecisionLabel(record);

  return {
    id: `${record.index}`,
    title: title || "",
    approved: record.state === "approved",
    rejected: record.state === "rejected",
    approverName: record.user?.name,
    approvalComment: record.approval?.comment,
    rejectionReason: record.rejection?.reason,
    interactive: canAct,
    onApprove: async (comment?: string) => {
      if (!lastExecution?.id) return;

      try {
        await canvasesInvokeNodeExecutionAction(
          withOrganizationHeader({
            path: {
              canvasId: context.canvasId,
              executionId: lastExecution.id,
              actionName: "approve",
            },
            body: {
              parameters: {
                index: record.index,
                comment: comment,
              },
            },
          }),
        );

        context.queryClient.invalidateQueries({
          queryKey: canvasKeys.nodeExecution(context.canvasId, context.node.id!),
        });
      } catch {
        showErrorToast("Failed to approve");
      }
    },
    onReject: async (reason: string) => {
      if (!lastExecution?.id) return;

      try {
        await canvasesInvokeNodeExecutionAction(
          withOrganizationHeader({
            path: {
              canvasId: context.canvasId,
              executionId: lastExecution.id,
              actionName: "reject",
            },
            body: {
              parameters: {
                index: record.index,
                reason: reason,
              },
            },
          }),
        );

        context.queryClient.invalidateQueries({
          queryKey: canvasKeys.nodeExecution(context.canvasId, context.node.id!),
        });
      } catch {
        showErrorToast("Failed to reject");
      }
    },
  };
}

function getApprovalSpecs(items: ApprovalRecord[]): ComponentBaseSpec[] {
  if (items.length === 0) return [];

  return [
    {
      title: "approval",
      tooltipTitle: "approval",
      values: items.map((item) => {
        let value = "";
        const label = item.type ? `${item.type[0].toUpperCase()}${item.type.slice(1)}` : "Item";

        if (item.type === "anyone") {
          value = "Anyone";
        }

        if (item.type === "user") {
          value = item.user?.name || item.user?.email || "User";
        }

        if (item.type === "role") {
          value = item.roleRef?.displayName || "Role";
        }

        if (item.type === "group") {
          value = item.groupRef?.displayName || "Group";
        }

        // Pretty-print values
        return {
          badges: [
            { label: `${label}:`, bgColor: "bg-gray-100", textColor: "text-gray-700" },
            { label: value || "—", bgColor: "bg-emerald-100", textColor: "text-emerald-800" },
          ],
        };
      }),
    },
  ];
}

function getApprovalEventSections(nodes: NodeInfo[], execution: ExecutionInfo): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName || "");
  const { title: eventTitle } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  const eventSubtitle = getComponentSubtitle(execution);

  const eventSection: EventSection = {
    receivedAt: new Date(execution.createdAt!),
    eventTitle: eventTitle,
    eventSubtitle: eventSubtitle,
    eventState: approvalStateFunction(execution),
    eventId: execution.rootEvent!.id!,
  };

  return [eventSection];
}

function getComponentSubtitle(execution: ExecutionInfo): string | React.ReactNode {
  const metadata = execution.metadata as ExecutionMetadata | undefined;
  if (!metadata) return "";

  // Show progress for in-progress approvals
  if (execution.state === "STATE_STARTED") {
    const approvalsCount = metadata.records.length || 0;
    const approvalsApprovedCount = metadata.records.filter((record) => record.state === "approved").length || 0;
    const subtitle = `${approvalsApprovedCount}/${approvalsCount} approved`;
    if (execution.createdAt) {
      return renderWithTimeAgo(subtitle, new Date(execution.createdAt));
    }
    return subtitle;
  }

  // Show relative time for completed executions (use updatedAt for finished, createdAt otherwise)
  const timestamp =
    execution.state === "STATE_FINISHED" && execution.updatedAt ? execution.updatedAt : execution.createdAt;

  if (timestamp) {
    const date = new Date(timestamp);
    const metadata = execution.metadata as Record<string, unknown> | undefined;
    const result = metadata?.result;

    if (result === "approved") {
      return renderWithTimeAgo("Approved", date);
    }

    if (result === "rejected") {
      return renderWithTimeAgo("Rejected", date);
    }

    return renderTimeAgo(date);
  }

  return "";
}

function getApprovalDecisionLabel(record: ApprovalRecord): string {
  if (record.type === "user") {
    return record.user?.name || "User";
  }

  if (record.type === "role" && record.roleRef) {
    return record.roleRef.displayName || "Role";
  }

  if (record.type === "group" && record.groupRef) {
    return record.groupRef.displayName || "Group";
  }

  return "Any user";
}

function canCurrentUserActOnApproval(
  queryClient: QueryClient,
  organizationId: string,
  record: ApprovalRecord,
  currentUser?: User,
): boolean {
  if (!currentUser || !organizationId) {
    return false;
  }

  switch (record.type) {
    case "anyone":
      return true;

    case "user":
      return record.user?.id === currentUser.id || record.user?.email === currentUser.email;

    case "role":
      return !!record.roleRef && currentUser.roles.includes(record.roleRef.name);

    case "group": {
      if (!record.groupRef) {
        return false;
      }

      const groupUsers = queryClient.getQueryData<SuperplaneUsersUser[]>(
        organizationKeys.groupUsers(organizationId, record.groupRef.name),
      );

      if (!Array.isArray(groupUsers)) return false;
      return groupUsers.some(
        (user) => user.metadata?.id === currentUser.id || user.metadata?.email === currentUser.email,
      );
    }
  }

  return false;
}
