/* eslint-disable @typescript-eslint/no-explicit-any */
import {
  ComponentsComponent,
  ComponentsNode,
  RolesRole,
  SuperplaneUsersUser,
  workflowsInvokeNodeExecutionAction,
  WorkflowsWorkflowNodeExecution,
  WorkflowsWorkflowNodeQueueItem,
} from "@/api-client";
import { ComponentAdditionalDataBuilder, ComponentBaseMapper } from "./types";
import { ComponentBaseProps, ComponentBaseSpec, EventSection, EventState } from "@/ui/componentBase";
import { getTriggerRenderer } from ".";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { ApprovalGroup } from "@/ui/approvalGroup";
import React from "react";
import { ApprovalItemProps } from "@/ui/approvalItem";
import { QueryClient } from "@tanstack/react-query";
import { organizationKeys } from "@/hooks/useOrganizationData";
import { withOrganizationHeader } from "@/utils/withOrganizationHeader";
import { workflowKeys } from "@/hooks/useWorkflowData";

type ApprovalItem = {
  type: string;
  user?: string;
  role?: string;
  group?: string;
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
      iconColor: getColorClass("orange"),
      headerColor: "bg-orange-100",
      iconBackground: getBackgroundColorClass("orange"),
      collapsedBackground: getBackgroundColorClass("orange"),
      collapsed: node.isCollapsed,
      title: node.name || componentDefinition?.label || "Approval",
      description: componentDefinition?.description,
      eventSections: getApprovalEventSections(nodes, lastExecution),
      specs: getApprovalSpecs(items, additionalData),
      customField: getApprovalCustomField(lastExecution, approvals),
    };
  },
};

function getApprovalCustomField(
  lastExecution: WorkflowsWorkflowNodeExecution | null,
  approvals: ApprovalItemProps[],
): React.ReactNode | undefined {
  const isAwaitingApproval = ["STATE_STARTED", "STATE_PENDING"].includes(lastExecution?.state || "");
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
          type === "user"
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
  execution: WorkflowsWorkflowNodeExecution | null,
): EventSection[] {
  if (!execution) {
    return [
      {
        title: "Last Run",
        eventTitle: "No events received yet",
        eventState: "neutral" as const,
      },
    ];
  }

  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");
  const { title: eventTitle } = rootTriggerRenderer.getTitleAndSubtitle(execution.rootEvent!);
  let sectionTitle = "Last Run";

  if (execution.state === "STATE_STARTED") {
    sectionTitle = "Awaiting Approval";
  }

  return [
    {
      title: sectionTitle,
      receivedAt: new Date(execution.createdAt!),
      eventTitle: eventTitle,
      eventState: executionToEventSectionState(execution),
    },
  ];
}

function executionToEventSectionState(execution: WorkflowsWorkflowNodeExecution): EventState {
  if (execution.state == "STATE_PENDING" || execution.state == "STATE_STARTED") {
    return "running";
  }

  if (execution.state == "STATE_FINISHED" && execution.result == "RESULT_PASSED") {
    return "success";
  }

  return "failed";
}

// ----------------------- Data Builder -----------------------

type ApprovalRecord = {
  index: number;
  state: string;
  type: string;
  user?: { name?: string; email?: string; avatarUrl?: string };
  role?: string;
  group?: string;
  approval?: { comment?: string };
  rejection?: { reason?: string };
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
  ) {
    const execution = lastExecutions.length > 0 ? lastExecutions[0] : null;
    const executionMetadata = execution?.metadata as Record<string, unknown> | undefined;
    const usersById: Record<string, { email?: string; name?: string }> = {};
    const rolesByName: Record<string, string> = {};
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
      }

      const rolesResp: RolesRole[] | undefined = queryClient.getQueryData(organizationKeys.roles(organizationId));
      if (Array.isArray(rolesResp)) {
        rolesResp.forEach((r: RolesRole) => {
          const name = r.metadata?.name;
          const display = r.spec?.displayName;
          if (name) rolesByName[name] = display || name;
        });
      }
    }

    // Map backend records to approval items
    const approvals = ((executionMetadata?.records as ApprovalRecord[] | undefined) || []).map(
      (record: ApprovalRecord) => {
        const isPending = record.state === "pending";
        const isExecutionActive = execution?.state === "STATE_STARTED";

        const approvalComment = record.approval?.comment as string | undefined;
        const hasApprovalArtifacts = record.state === "approved" && approvalComment;

        return {
          id: `${record.index}`,
          title:
            record.type === "user" && record.user
              ? record.user.name || record.user.email
              : record.type === "role" && record.role
                ? record.role
                : record.type === "group" && record.group
                  ? record.group
                  : "Unknown",
          approved: record.state === "approved",
          rejected: record.state === "rejected",
          approverName: record.user?.name,
          approverAvatar: record.user?.avatarUrl,
          rejectionComment: record.rejection?.reason,
          interactive: isPending && isExecutionActive,
          requireArtifacts:
            isPending && isExecutionActive
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
                      index: record.index,
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
      },
    );

    return {
      approvals,
      usersById,
      rolesByName,
    };
  },
};
