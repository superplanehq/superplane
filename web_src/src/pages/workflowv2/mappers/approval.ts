/* eslint-disable @typescript-eslint/no-explicit-any */
import {
  ComponentsComponent,
  ComponentsNode,
  GroupsGroup,
  RolesRole,
  SuperplaneUsersUser,
  groupsListGroupUsers,
  workflowsInvokeNodeExecutionAction,
  WorkflowsWorkflowNodeExecution,
  WorkflowsWorkflowNodeQueueItem,
} from "@/api-client";
import { ComponentAdditionalDataBuilder, ComponentBaseMapper, EventStateRegistry, StateFunction } from "./types";
import {
  ComponentBaseProps,
  ComponentBaseSpec,
  EventSection,
  EventState,
  EventStateMap,
  DEFAULT_EVENT_STATE_MAP,
} from "@/ui/componentBase";
import { getTriggerRenderer } from ".";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { ApprovalGroup } from "@/ui/approvalGroup";
import React from "react";
import { ApprovalItemProps } from "@/ui/approvalItem";
import { QueryClient } from "@tanstack/react-query";
import { organizationKeys } from "@/hooks/useOrganizationData";
import { withOrganizationHeader } from "@/utils/withOrganizationHeader";
import { workflowKeys } from "@/hooks/useWorkflowData";
import { formatTimeAgo } from "@/utils/date";

type ApprovalItem = {
  type: string;
  user?: string;
  role?: string;
  group?: string;
};

type ApprovalLabelMaps = {
  rolesByName?: Record<string, string>;
  groupsByName?: Record<string, string>;
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
export const approvalStateFunction: StateFunction = (execution: WorkflowsWorkflowNodeExecution): EventState => {
  if (
    execution.resultMessage &&
    (execution.resultReason === "RESULT_REASON_ERROR" || execution.result === "RESULT_FAILED")
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
    const metadata = execution.metadata as Record<string, any> | undefined;
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
  props(
    nodes: ComponentsNode[],
    node: ComponentsNode,
    componentDefinition: ComponentsComponent,
    lastExecutions: WorkflowsWorkflowNodeExecution[],
    _?: WorkflowsWorkflowNodeQueueItem[],
    additionalData?: unknown,
  ): ComponentBaseProps {
    const lastExecution = lastExecutions.length > 0 ? lastExecutions[0] : null;
    const items = (node.configuration?.items || []) as ApprovalItem[];
    const approvals = (additionalData as { approvals?: ApprovalItemProps[] })?.approvals || [];
    return {
      iconSlug: componentDefinition.icon || "hand",
      iconColor: getColorClass("black"),
      collapsedBackground: getBackgroundColorClass("orange"),
      collapsed: node.isCollapsed,
      title: node.name || componentDefinition?.label || "Approval",
      eventSections: lastExecution ? getApprovalEventSections(nodes, lastExecution, additionalData) : undefined,
      includeEmptyState: !lastExecution,
      specs: getApprovalSpecs(items, additionalData),
      customField: getApprovalCustomField(lastExecution, approvals),
      eventStateMap: APPROVAL_STATE_MAP,
    };
  },
  subtitle(node, execution, additionalData) {
    return getComponentSubtitle(node, execution, additionalData);
  },
  getExecutionDetails(execution: WorkflowsWorkflowNodeExecution, _node: ComponentsNode): Record<string, any> {
    const details: Record<string, any> = {};
    const metadata = execution.metadata as Record<string, unknown> | undefined;
    const records = (metadata?.records as ApprovalRecord[] | undefined) || [];

    if (execution.createdAt) {
      details["Started at"] = new Date(execution.createdAt).toLocaleString();
    }

    if (execution.state === "STATE_FINISHED" && execution.updatedAt) {
      details["Finished at"] = new Date(execution.updatedAt).toLocaleString();
    }

    if (execution.result !== "RESULT_CANCELLED") {
      details["Approvals"] = buildApprovalTimeline(records);
    }

    return details;
  },
};

function getApprovalCustomField(
  lastExecution: WorkflowsWorkflowNodeExecution | null,
  approvals: ApprovalItemProps[],
): React.ReactNode | undefined {
  const isAwaitingApproval = ["STATE_STARTED", "STATE_PENDING"].includes(lastExecution?.state || "");
  if (!isAwaitingApproval || approvals.length == 0) return;
  return React.createElement(ApprovalGroup, { approvals, awaitingApproval: isAwaitingApproval });
}

function getApprovalSpecs(items: ApprovalItem[], additionalData?: unknown): ComponentBaseSpec[] {
  if (items.length === 0) return [];

  const usersById = (additionalData as { usersById?: Record<string, any> })?.usersById || {};
  const rolesByName = (additionalData as { rolesByName?: Record<string, any> })?.rolesByName || {};

  return [
    {
      title: "approvals required",
      tooltipTitle: "approvals required",
      values: items.map((item) => {
        const type = (item.type || "").toString();
        let value =
          type === "anyone"
            ? "Anyone"
            : type === "user"
              ? item.user || ""
              : type === "role"
                ? item.role || ""
                : type === "group"
                  ? item.group || ""
                  : "";
        const label = type ? `${type[0].toUpperCase()}${type.slice(1)}` : "Item";

        // Pretty-print values
        if (type === "user" && value && usersById[value]) {
          value = usersById[value].email || usersById[value].name || value;
        }
        if (type === "role" && value) {
          value = rolesByName[value] || value.replace(/^(org_|canvas_)/i, "");
          // Fallback to simple suffix mapping when not found
          const suffix = (item.role || "").split("_").pop();
          if (!rolesByName[item.role || ""] && suffix) {
            const map: any = { viewer: "Viewer", admin: "Admin", owner: "Owner" };
            value = map[suffix] || value;
          }
        }
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

function getApprovalEventSections(
  nodes: ComponentsNode[],
  execution: WorkflowsWorkflowNodeExecution,
  additionalData?: unknown,
): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");
  const { title: eventTitle } = rootTriggerRenderer.getTitleAndSubtitle(execution.rootEvent!);

  const eventSubtitle = getComponentSubtitle({} as ComponentsNode, execution, additionalData);

  const eventSection: EventSection = {
    receivedAt: new Date(execution.createdAt!),
    eventTitle: eventTitle,
    eventSubtitle: eventSubtitle,
    eventState: approvalStateFunction(execution),
    eventId: execution.rootEvent?.id,
  };

  return [eventSection];
}

function getComponentSubtitle(
  _node: ComponentsNode,
  execution: WorkflowsWorkflowNodeExecution,
  additionalData?: unknown,
): string | React.ReactNode {
  // Show progress for in-progress approvals
  if (execution.state === "STATE_STARTED") {
    const approvals = (additionalData as { approvals?: ApprovalItemProps[] })?.approvals;
    const approvalsCount = approvals?.length || 0;
    const approvalsApprovedCount = approvals?.filter((approval) => approval.approved).length || 0;
    const subtitle = `${approvalsApprovedCount}/${approvalsCount} approved`;
    if (execution.createdAt) {
      return `${subtitle} · ${formatTimeAgo(new Date(execution.createdAt))}`;
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
    const timeAgo = formatTimeAgo(date);

    if (result === "approved") {
      return `Approved · ${timeAgo}`;
    }

    if (result === "rejected") {
      return `Rejected · ${timeAgo}`;
    }

    return timeAgo;
  }

  return "";
}

function getApprovalDecisionLabel(record: ApprovalRecord, labelMaps?: ApprovalLabelMaps): string {
  const rolesByName = labelMaps?.rolesByName;
  const groupsByName = labelMaps?.groupsByName;

  if (record.type === "user") {
    return record.user?.name || record.user?.email || "User";
  }

  if (record.user?.name || record.user?.email) {
    return record.user?.name || record.user?.email || "User";
  }

  if (record.type === "role") {
    return (record.role ? rolesByName?.[record.role] : undefined) || record.role || "Role";
  }

  if (record.type === "group") {
    return (record.group ? groupsByName?.[record.group] : undefined) || record.group || "Group";
  }

  if (record.type === "anyone") {
    return "Any user";
  }

  return "Approver";
}

function buildApprovalTimeline(records: ApprovalRecord[]) {
  return records
    .map((record) => {
      const meta = getApprovalDecisionMeta(record);
      return {
        label: getApprovalDecisionLabel(record),
        status: meta.status,
        timestamp: meta.timestamp,
        comment: meta.comment,
      };
    })
    .sort((a, b) => {
      if (!a.timestamp && !b.timestamp) return 0;
      if (!a.timestamp) return 1;
      if (!b.timestamp) return -1;
      return new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime();
    });
}

function getApprovalDecisionMeta(record: ApprovalRecord): {
  status: string;
  timestamp?: string;
  comment?: string;
} {
  const approvalComment = record.approval?.comment?.trim();
  const rejectionReason = record.rejection?.reason?.trim();
  const comment = approvalComment || rejectionReason;

  if (record.state === "approved") {
    return {
      status: "Approved",
      timestamp: formatDecisionTimestamp(record.approval?.approvedAt),
      comment,
    };
  }

  if (record.state === "rejected") {
    return {
      status: "Rejected",
      timestamp: formatDecisionTimestamp(record.rejection?.rejectedAt),
      comment,
    };
  }

  return {
    status: "Pending",
    comment,
  };
}

function formatDecisionTimestamp(timestamp?: string): string | undefined {
  if (!timestamp) return undefined;

  const parsed = new Date(timestamp);
  if (Number.isNaN(parsed.getTime())) return undefined;

  return formatTimeAgo(parsed);
}

// ----------------------- Data Builder -----------------------

type ApprovalRecord = {
  index: number;
  state: string;
  type: string;
  user?: { id?: string; name?: string; email?: string; avatarUrl?: string };
  role?: string;
  group?: string;
  approval?: { approvedAt?: string; comment?: string };
  rejection?: { rejectedAt?: string; reason?: string };
};

export const approvalDataBuilder: ComponentAdditionalDataBuilder = {
  buildAdditionalData(
    _nodes: ComponentsNode[],
    node: ComponentsNode,
    _componentDefinition: ComponentsComponent,
    lastExecutions: WorkflowsWorkflowNodeExecution[],
    workflowId: string,
    queryClient: QueryClient,
    organizationId?: string,
    currentUser?: { id?: string; email?: string },
  ) {
    const execution = lastExecutions.length > 0 ? lastExecutions[0] : null;
    const executionMetadata = execution?.metadata as Record<string, unknown> | undefined;
    const usersById: Record<string, { email?: string; name?: string }> = {};
    const rolesByName: Record<string, string> = {};
    const groupsByName: Record<string, string> = {};
    let currentUserRoles: string[] = [];
    const currentUserId = currentUser?.id;
    const currentUserEmail = currentUser?.email;
    if (organizationId) {
      const usersResp: SuperplaneUsersUser[] | undefined = queryClient.getQueryData(
        organizationKeys.users(organizationId),
      );
      if (Array.isArray(usersResp)) {
        usersResp.forEach((u: SuperplaneUsersUser) => {
          const id = u.metadata?.id;
          const email = u.metadata?.email;
          const name = u.spec?.displayName;
          if (id) usersById[id] = { email, name };
        });

        if (currentUserId || currentUserEmail) {
          const currentOrgUser = usersResp.find(
            (u) =>
              (currentUserId && u.metadata?.id === currentUserId) ||
              (currentUserEmail && u.metadata?.email === currentUserEmail),
          );
          if (currentOrgUser?.status?.roleAssignments) {
            currentUserRoles = currentOrgUser.status.roleAssignments
              .filter((assignment) => !assignment.domainId || assignment.domainId === organizationId)
              .map((assignment) => assignment.roleName)
              .filter((roleName): roleName is string => !!roleName);
          }
        }
      }

      const rolesResp: RolesRole[] | undefined = queryClient.getQueryData(organizationKeys.roles(organizationId));
      if (Array.isArray(rolesResp)) {
        rolesResp.forEach((r: RolesRole) => {
          const name = r.metadata?.name;
          const display = r.spec?.displayName;
          if (name) rolesByName[name] = display || name;
        });
      }

      const groupsResp: GroupsGroup[] | undefined = queryClient.getQueryData(organizationKeys.groups(organizationId));
      if (Array.isArray(groupsResp)) {
        groupsResp.forEach((group: GroupsGroup) => {
          const name = group.metadata?.name;
          const display = group.spec?.displayName;
          if (name) groupsByName[name] = display || name;
        });
      }
    }

    if (organizationId) {
      const groupNames = new Set(
        ((executionMetadata?.records as ApprovalRecord[] | undefined) || [])
          .filter((record) => record.type === "group" && record.group)
          .map((record) => record.group as string),
      );

      groupNames.forEach((groupName) => {
        const queryKey = organizationKeys.groupUsers(organizationId, groupName);
        if (queryClient.getQueryData(queryKey)) return;
        queryClient.prefetchQuery({
          queryKey,
          queryFn: async () => {
            const response = await groupsListGroupUsers(
              withOrganizationHeader({
                path: { groupName },
                query: { domainId: organizationId, domainType: "DOMAIN_TYPE_ORGANIZATION" },
              }),
            );
            return response.data?.users || [];
          },
          staleTime: 5 * 60 * 1000,
          gcTime: 10 * 60 * 1000,
        });
      });
    }

    const approvalRecords = (executionMetadata?.records as ApprovalRecord[] | undefined) || [];
    const hasApprovedAnyRecord = hasCurrentUserApprovedAnyRecord(approvalRecords, currentUserId, currentUserEmail);
    const pendingUserRecordIndex = getPendingUserApprovalIndex(approvalRecords, currentUserId, currentUserEmail);
    const interactiveApprovalIndex =
      hasApprovedAnyRecord || execution?.state !== "STATE_STARTED"
        ? undefined
        : getInteractiveApprovalIndex(approvalRecords, {
            currentUserId,
            currentUserEmail,
            currentUserRoles,
            organizationId,
            queryClient,
          });

    // Map backend records to approval items
    const labelMaps = { rolesByName, groupsByName };
    const approvals = approvalRecords.map((record: ApprovalRecord) => {
      const isPending = record.state === "pending";
      const isExecutionActive = execution?.state === "STATE_STARTED";
      const approveIndex =
        record.type === "anyone" && pendingUserRecordIndex !== undefined ? pendingUserRecordIndex : record.index;
      const canAct =
        !hasApprovedAnyRecord &&
        isPending &&
        isExecutionActive &&
        record.index === interactiveApprovalIndex &&
        canCurrentUserActOnApproval(record, {
          currentUserId,
          currentUserEmail,
          currentUserRoles,
          organizationId,
          queryClient,
        });

      const approvalComment = record.approval?.comment as string | undefined;
      const hasApprovalArtifacts = record.state === "approved" && approvalComment;

      const userLabel = record.user?.name || record.user?.email;
      const title =
        userLabel ||
        (record.type === "user"
          ? record.user?.name || record.user?.email
          : record.type === "role" || record.type === "group"
            ? getApprovalDecisionLabel(record, labelMaps)
            : record.type === "anyone"
              ? "Any user"
              : "Unknown");

      return {
        id: `${record.index}`,
        title,
        approved: record.state === "approved",
        rejected: record.state === "rejected",
        approverName: record.user?.name,
        approverAvatar: record.user?.avatarUrl,
        rejectionComment: record.rejection?.reason,
        interactive: canAct,
        requireArtifacts: canAct
          ? [
              {
                label: "comment",
                optional: true,
              },
            ]
          : undefined,
        artifacts: hasApprovalArtifacts
          ? {
              Comment: approvalComment,
            }
          : undefined,
        artifactCount: hasApprovalArtifacts ? 1 : undefined,
        onApprove: async (artifacts?: Record<string, string>) => {
          if (!execution?.id) return;

          try {
            await workflowsInvokeNodeExecutionAction(
              withOrganizationHeader({
                path: {
                  workflowId: workflowId,
                  executionId: execution.id,
                  actionName: "approve",
                },
                body: {
                  parameters: {
                    index: approveIndex,
                    comment: artifacts?.comment,
                  },
                },
              }),
            );

            queryClient.invalidateQueries({
              queryKey: workflowKeys.nodeExecution(workflowId, node.id!),
            });
          } catch (error) {
            console.error("Failed to approve:", error);
          }
        },
        onReject: async (comment?: string) => {
          if (!execution?.id) return;

          try {
            await workflowsInvokeNodeExecutionAction(
              withOrganizationHeader({
                path: {
                  workflowId: workflowId,
                  executionId: execution.id,
                  actionName: "reject",
                },
                body: {
                  parameters: {
                    index: record.index,
                    reason: comment,
                  },
                },
              }),
            );

            queryClient.invalidateQueries({
              queryKey: workflowKeys.nodeExecution(workflowId, node.id!),
            });
          } catch (error) {
            console.error("Failed to reject:", error);
          }
        },
      };
    });

    return {
      approvals,
      usersById,
      rolesByName,
      groupsByName,
    };
  },
};

function canCurrentUserActOnApproval(
  record: ApprovalRecord,
  {
    currentUserId,
    currentUserEmail,
    currentUserRoles,
    organizationId,
    queryClient,
  }: {
    currentUserId?: string;
    currentUserEmail?: string;
    currentUserRoles: string[];
    organizationId?: string;
    queryClient: QueryClient;
  },
): boolean {
  switch (record.type) {
    case "anyone":
      return !!(currentUserId || currentUserEmail);
    case "user":
      return (
        (!!currentUserId && record.user?.id === currentUserId) ||
        (!!currentUserEmail && record.user?.email === currentUserEmail)
      );
    case "role":
      return !!record.role && currentUserRoles.includes(record.role);
    case "group": {
      if (!record.group || !organizationId) return false;
      const groupUsers = queryClient.getQueryData<SuperplaneUsersUser[]>(
        organizationKeys.groupUsers(organizationId, record.group),
      );
      if (!Array.isArray(groupUsers)) return false;
      return groupUsers.some(
        (user) =>
          (!!currentUserId && user.metadata?.id === currentUserId) ||
          (!!currentUserEmail && user.metadata?.email === currentUserEmail),
      );
    }
    default:
      return false;
  }
}

function hasCurrentUserApprovedAnyRecord(
  records: ApprovalRecord[],
  currentUserId?: string,
  currentUserEmail?: string,
): boolean {
  if (!currentUserId && !currentUserEmail) return false;

  return records.some(
    (record) =>
      record.state === "approved" &&
      ((currentUserId && record.user?.id === currentUserId) ||
        (currentUserEmail && record.user?.email === currentUserEmail)),
  );
}

function getPendingUserApprovalIndex(
  records: ApprovalRecord[],
  currentUserId?: string,
  currentUserEmail?: string,
): number | undefined {
  if (!currentUserId && !currentUserEmail) return undefined;

  const match = records.find(
    (record) =>
      record.type === "user" &&
      record.state === "pending" &&
      ((currentUserId && record.user?.id === currentUserId) ||
        (currentUserEmail && record.user?.email === currentUserEmail)),
  );

  return match?.index;
}

function getInteractiveApprovalIndex(
  records: ApprovalRecord[],
  {
    currentUserId,
    currentUserEmail,
    currentUserRoles,
    organizationId,
    queryClient,
  }: {
    currentUserId?: string;
    currentUserEmail?: string;
    currentUserRoles: string[];
    organizationId?: string;
    queryClient: QueryClient;
  },
): number | undefined {
  const pendingUserIndex = getPendingUserApprovalIndex(records, currentUserId, currentUserEmail);
  if (pendingUserIndex !== undefined) {
    const pendingUserRecord = records.find((record) => record.index === pendingUserIndex);
    if (
      pendingUserRecord &&
      pendingUserRecord.state === "pending" &&
      canCurrentUserActOnApproval(pendingUserRecord, {
        currentUserId,
        currentUserEmail,
        currentUserRoles,
        organizationId,
        queryClient,
      })
    ) {
      return pendingUserIndex;
    }
  }

  const fallback = records.find(
    (record) =>
      record.state === "pending" &&
      canCurrentUserActOnApproval(record, {
        currentUserId,
        currentUserEmail,
        currentUserRoles,
        organizationId,
        queryClient,
      }),
  );

  return fallback?.index;
}
