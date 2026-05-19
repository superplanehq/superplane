import type { ComponentBaseProps } from "@/ui/componentBase";
import type React from "react";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import { renderTimeAgo } from "@/components/TimeAgo";
import { jiraComponentBaseProps } from "./base";
import { addDetail, addProjectMetadata } from "./utils";
import type { CreateWorkflowConfiguration, JiraNodeMetadata, JiraWorkflow } from "./types";

export const createWorkflowMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return jiraComponentBaseProps(context, metadataList(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {
      "Executed At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
    };

    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const workflow = outputs?.default?.[0]?.data as JiraWorkflow | undefined;
    if (workflow) {
      addDetail(details, "Workflow ID", workflow.id);
      addDetail(details, "Name", workflow.name);
      if (workflow.version?.versionNumber !== undefined) {
        details["Version"] = String(workflow.version.versionNumber);
      }
    }

    const configuration = context.node.configuration as CreateWorkflowConfiguration | undefined;
    if (configuration?.scope) {
      details["Scope"] = configuration.scope;
    }
    if (configuration?.statuses?.length) {
      details["Statuses"] = String(configuration.statuses.length);
    }
    if (configuration?.transitions?.length) {
      details["Transitions"] = String(configuration.transitions.length);
    }

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const workflow = outputs?.default?.[0]?.data as JiraWorkflow | undefined;
    if (workflow?.name) return workflow.name;
    if (context.execution.createdAt) {
      return renderTimeAgo(new Date(context.execution.createdAt));
    }
    return "";
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as JiraNodeMetadata | undefined;
  const configuration = node.configuration as CreateWorkflowConfiguration | undefined;

  const name = nodeMetadata?.workflowName || configuration?.name;
  if (name) {
    metadata.push({ icon: "workflow", label: name });
  }

  const scope = configuration?.scope || "GLOBAL";
  metadata.push({ icon: "globe", label: scope === "PROJECT" ? "Project scoped" : "Global" });

  addProjectMetadata(metadata, nodeMetadata?.project, configuration?.project);

  if (configuration?.statuses?.length) {
    metadata.push({ icon: "list", label: `${configuration.statuses.length} statuses` });
  }

  return metadata.slice(0, 4);
}
