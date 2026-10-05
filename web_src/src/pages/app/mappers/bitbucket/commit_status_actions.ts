import type React from "react";
import type { ComponentBaseProps } from "@/ui/componentBase";
import type { ComponentBaseContext, ComponentBaseMapper, ExecutionDetailsContext, SubtitleContext } from "../types";
import { formatTimestamp } from "../utils";
import { baseProps } from "./base";
import type { CombinedCommitStatus, CommitStatus } from "./types";
import { addDetailIfPresent, buildBitbucketExecutionSubtitle, defaultOutput, shortHash } from "./utils";

export const publishCommitStatusMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const output = defaultOutput<CommitStatus>(context.execution.outputs);

    if (output) {
      return `${output.data.key || ""} ${output.data.state || ""}`.trim();
    }

    return buildBitbucketExecutionSubtitle(context.execution, "Status Published");
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const output = defaultOutput<CommitStatus>(context.execution.outputs);

    if (!output) {
      return {};
    }

    const status = output.data;
    const details: Record<string, string> = {
      "Published At": formatTimestamp(status.updated_on || status.created_on, output.timestamp),
    };

    addDetailIfPresent(details, "Key", status.key);
    addDetailIfPresent(details, "Name", status.name);
    addDetailIfPresent(details, "State", status.state);
    addDetailIfPresent(details, "Ref", status.refname);
    addDetailIfPresent(details, "Commit", shortHash(status.commit?.hash));
    addDetailIfPresent(details, "URL", status.url);

    return details;
  },
};

export const getCommitStatusMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const output = defaultOutput<CombinedCommitStatus>(context.execution.outputs);

    if (output) {
      const total = output.data.total_count ?? 0;
      return `${output.data.state || ""} (${total} ${total === 1 ? "check" : "checks"})`.trim();
    }

    return buildBitbucketExecutionSubtitle(context.execution, "Statuses Retrieved");
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const output = defaultOutput<CombinedCommitStatus>(context.execution.outputs);

    if (!output) {
      return {};
    }

    const combined = output.data;
    const details: Record<string, string> = {};

    addDetailIfPresent(details, "Commit", shortHash(combined.commit));
    addDetailIfPresent(details, "Combined State", combined.state);
    addDetailIfPresent(details, "Checks", combined.total_count);

    // List the failing checks by name, since that is what a human reading a blocked
    // deploy actually needs to see.
    const failing = (combined.statuses || []).filter(
      (status) => status.state === "FAILED" || status.state === "STOPPED",
    );
    if (failing.length > 0) {
      details["Not Passing"] = failing.map((status) => status.key || status.name || "unknown").join(", ");
    }

    return details;
  },
};
