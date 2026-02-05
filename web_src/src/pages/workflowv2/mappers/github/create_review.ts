import { ComponentBaseProps } from "@/ui/componentBase";
import {
  ComponentBaseMapper,
  ComponentBaseContext,
  SubtitleContext,
  ExecutionDetailsContext,
  OutputPayload,
  NodeInfo,
} from "../types";
import { baseProps } from "./base";
import { buildGithubExecutionSubtitle } from "./utils";
import { MetadataItem } from "@/ui/metadataList";

interface ReviewOutput {
  id?: number;
  node_id?: string;
  state?: string;
  body?: string;
  html_url?: string;
  pull_request?: string;
  submitted_at?: string;
  user?: {
    login?: string;
    html_url?: string;
  };
}

interface CreateReviewConfiguration {
  repository?: string;
  pullNumber?: string;
  event?: string;
  body?: string;
}

function getCreateReviewMetadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as CreateReviewConfiguration | undefined;
  const nodeMetadata = node.metadata as { repository?: { name?: string } } | undefined;

  if (nodeMetadata?.repository?.name) {
    metadata.push({ icon: "book", label: nodeMetadata.repository.name });
  }

  if (configuration?.pullNumber) {
    metadata.push({ icon: "hash", label: `PR #${configuration.pullNumber}` });
  }

  if (configuration?.event) {
    const eventLabels: Record<string, string> = {
      APPROVE: "Approve",
      REQUEST_CHANGES: "Request Changes",
      COMMENT: "Comment",
    };
    metadata.push({ icon: "tag", label: eventLabels[configuration.event] || configuration.event });
  }

  return metadata;
}

export const createReviewMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const base = baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);

    return {
      ...base,
      metadata: getCreateReviewMetadataList(context.node),
    };
  },

  subtitle(context: SubtitleContext): string {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    if (outputs?.default && outputs.default.length > 0) {
      const reviews = outputs.default[0].data as ReviewOutput | ReviewOutput[];
      const review = Array.isArray(reviews) ? reviews[0] : reviews;
      if (review?.state) {
        return buildGithubExecutionSubtitle(context.execution, review.state);
      }
    }
    return buildGithubExecutionSubtitle(context.execution);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {};

    Object.assign(details, {
      "Submitted At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
    });

    if (outputs?.default && outputs.default.length > 0) {
      const reviews = outputs.default[0].data as ReviewOutput | ReviewOutput[];
      const review = Array.isArray(reviews) ? reviews[0] : reviews;

      if (review) {
        if (review.state) {
          details["State"] = review.state;
        }

        if (review.html_url) {
          details["Review URL"] = review.html_url;
        }

        if (review.user?.login) {
          details["Reviewer"] = review.user.html_url || review.user.login;
        }

        if (review.body) {
          const truncated = review.body.length > 100 ? review.body.substring(0, 100) + "..." : review.body;
          details["Body"] = truncated;
        }

        if (review.id) {
          details["Review ID"] = review.id.toString();
        }
      }
    }

    return details;
  },
};
