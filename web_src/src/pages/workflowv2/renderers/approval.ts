import { ComponentsNode, TriggersTrigger, WorkflowsWorkflowEvent } from "@/api-client";
import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerRenderer } from "./types";
import { TriggerProps } from "@/ui/trigger";
import { flattenObject } from "@/lib/utils";

type ApprovalState = "approved" | "rejected" | "error";

interface ApprovalEventData {
  state?: ApprovalState;
  meta?: string;
  approved_by?: string[];
  rejected_by?: string[];
  required_approvals?: number;
  received_approvals?: number;
  error_message?: string;
}

export const approvalTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (event: WorkflowsWorkflowEvent): { title: string; subtitle: string } => {
    const eventData = event.data as ApprovalEventData;

    if (eventData?.state === "approved") {
      return {
        title: "Approved",
        subtitle: eventData.approved_by?.join(", ") || "",
      };
    }

    if (eventData?.state === "rejected") {
      return {
        title: "Rejected",
        subtitle: eventData.rejected_by?.join(", ") || "",
      };
    }

    if (eventData?.state === "error") {
      return {
        title: "Error",
        subtitle: eventData.error_message || "Configuration error",
      };
    }

    return { title: event.id || "Approval Pending", subtitle: "" };
  },

  getRootEventValues: (event: WorkflowsWorkflowEvent): Record<string, string> => {
    const eventData = event.data as ApprovalEventData;

    const values: Record<string, string> = {
      State: eventData?.state || "pending",
    };

    if (eventData?.required_approvals !== undefined) {
      values["Required Approvals"] = eventData.required_approvals.toString();
    }

    if (eventData?.received_approvals !== undefined) {
      values["Received Approvals"] = eventData.received_approvals.toString();
    }

    if (eventData?.approved_by?.length) {
      values["Approved By"] = eventData.approved_by.join(", ");
    }

    if (eventData?.rejected_by?.length) {
      values["Rejected By"] = eventData.rejected_by.join(", ");
    }

    if (eventData?.error_message) {
      values["Error Message"] = eventData.error_message;
    }

    return { ...values, ...flattenObject(event.data || {}) };
  },

  getTriggerProps: (node: ComponentsNode, _trigger: TriggersTrigger, lastEvent: WorkflowsWorkflowEvent) => {
    const eventData = lastEvent?.data as ApprovalEventData;
    const state = eventData?.state;

    let iconSlug = "circle-dashed";
    let iconColor = getColorClass("gray");
    let headerColor = getBackgroundColorClass("gray");
    let collapsedBackground = getBackgroundColorClass("gray");
    const zeroStateText = "Awaiting events for approval";

    if (state === "approved") {
      iconSlug = "check";
      iconColor = getColorClass("green");
      headerColor = getBackgroundColorClass("green");
      collapsedBackground = getBackgroundColorClass("green");
    } else if (state === "rejected") {
      iconSlug = "x";
      iconColor = getColorClass("red");
      headerColor = getBackgroundColorClass("red");
      collapsedBackground = getBackgroundColorClass("red");
    } else if (state === "error") {
      iconSlug = "triangle-alert";
      iconColor = getColorClass("red");
      headerColor = getBackgroundColorClass("red");
      collapsedBackground = getBackgroundColorClass("red");
    }

    const props: TriggerProps = {
      title: node.name || "Approval",
      iconSlug,
      iconColor,
      headerColor,
      collapsedBackground,
      metadata: [],
      zeroStateText,
    };

    if (lastEvent) {
      const titleData = approvalTriggerRenderer.getTitleAndSubtitle(lastEvent);
      props.lastEventData = {
        title: titleData.title,
        subtitle: titleData.subtitle,
        receivedAt: new Date(lastEvent.createdAt!),
        state: state === "approved" ? "processed" : state === "rejected" || state === "error" ? "discarded" : "processed",
      };
    }

    return props;
  },
};