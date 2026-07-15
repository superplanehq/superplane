import type React from "react";
import type { ComponentBaseProps } from "@/ui/componentBase";
import type { MetadataItem } from "@/ui/metadataList";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { formatTimestamp } from "../utils";
import { baseProps } from "./base";
import type { GitLabNodeMetadata, MergeRequestApproval } from "./types";
import { buildGitlabExecutionSubtitle } from "./utils";

interface ApproveMergeRequestConfiguration {
  project?: string;
  mergeRequestIid?: string;
}

export const approveMergeRequestMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const props = baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
    const configuration = (context.node.configuration as ApproveMergeRequestConfiguration | undefined) ?? {};
    const metadata = (context.node.metadata as GitLabNodeMetadata | undefined) ?? ({} as GitLabNodeMetadata);

    const project = metadata?.project?.name || configuration.project;
    const metadataItems: MetadataItem[] = [];

    if (project) {
      metadataItems.push({ icon: "book", label: project });
    }

    if (configuration.mergeRequestIid) {
      metadataItems.push({ icon: "git-pull-request", label: `!${configuration.mergeRequestIid}` });
    }

    return {
      ...props,
      metadata: metadataItems.length > 0 ? metadataItems : props.metadata,
    };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    return buildGitlabExecutionSubtitle(context.execution);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {};

    if (!outputs?.default || outputs.default.length === 0) {
      return details;
    }

    const payload = outputs.default[0];
    const approval = payload.data as MergeRequestApproval | undefined;

    details["Approved At"] = formatTimestamp(payload.timestamp);
    details["Merge Request"] = approval?.iid ? `!${approval.iid} ${approval.title || ""}`.trim() : "-";

    const mergeRequestUrl = buildMergeRequestUrl(context, approval?.iid);
    if (mergeRequestUrl) {
      details["Merge Request URL"] = mergeRequestUrl;
    }

    const approvedBy = (approval?.approved_by || [])
      .map((approver) => approver.user?.username)
      .filter(Boolean)
      .join(", ");
    if (approvedBy) {
      details["Approved By"] = approvedBy;
    }

    if (approval?.approvals_required) {
      details["Approvals Required"] = approval.approvals_required.toString();
      details["Approvals Left"] = (approval.approvals_left ?? 0).toString();
    }

    return details;
  },
};

function buildMergeRequestUrl(context: ExecutionDetailsContext, iid?: number): string | undefined {
  const metadata = context.node.metadata as GitLabNodeMetadata | undefined;
  const configuration = (context.node.configuration as ApproveMergeRequestConfiguration | undefined) ?? {};
  const projectUrl = metadata?.project?.url;
  const mergeRequestIid = iid ?? configuration.mergeRequestIid;

  if (!projectUrl || !mergeRequestIid) {
    return undefined;
  }

  return `${projectUrl}/-/merge_requests/${mergeRequestIid}`;
}
