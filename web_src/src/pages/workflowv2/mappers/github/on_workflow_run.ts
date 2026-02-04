import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import githubIcon from "@/assets/icons/integrations/github.svg";
import { TriggerProps } from "@/ui/trigger";
import { BaseNodeMetadata } from "./types";
import { buildGithubSubtitle } from "./utils";

interface OnWorkflowRunConfiguration {
  conclusions: string[];
  workflowFiles: string[];
}

interface WorkflowRun {
  id?: number;
  name?: string;
  display_title?: string;
  head_branch?: string;
  head_sha?: string;
  path?: string;
  run_number?: number;
  event?: string;
  status?: string;
  conclusion?: string;
  html_url?: string;
  created_at?: string;
  head_commit?: {
    id?: string;
    message?: string;
    author?: {
      name?: string;
      email?: string;
    };
  };
  actor?: {
    login?: string;
  };
}

interface Workflow {
  id?: number;
  name?: string;
  path?: string;
}

interface OnWorkflowRunEventData {
  action?: string;
  workflow_run?: WorkflowRun;
  workflow?: Workflow;
}

/**
 * Renderer for the "github.onWorkflowRun" trigger
 */
export const onWorkflowRunTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as OnWorkflowRunEventData;
    const workflowName =
      eventData?.workflow_run?.display_title ||
      eventData?.workflow_run?.name ||
      eventData?.workflow?.name ||
      "Workflow";
    const conclusion = eventData?.workflow_run?.conclusion || "";

    return {
      title: workflowName,
      subtitle: buildGithubSubtitle(conclusion, context.event?.createdAt),
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnWorkflowRunEventData;
    const receivedAt = context.event?.createdAt ? new Date(context.event?.createdAt || "").toLocaleString() : "";

    return {
      "Received at": receivedAt,
      Conclusion: eventData?.workflow_run?.conclusion || "",
      "Triggered by": eventData?.workflow_run?.event || "",
      "Workflow link": eventData?.workflow_run?.html_url || "",
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as unknown as BaseNodeMetadata;
    const configuration = node.configuration as unknown as OnWorkflowRunConfiguration;
    const metadataItems = [];

    if (metadata?.repository?.name) {
      metadataItems.push({
        icon: "book",
        label: metadata.repository.name,
      });
    }

    if (configuration?.conclusions && configuration.conclusions.length > 0) {
      metadataItems.push({
        icon: "funnel",
        label: configuration.conclusions.join(", "),
      });
    }

    // Build specs for workflow files (shown as expandable tooltip like filter/approval components)
    const specs =
      configuration?.workflowFiles && configuration.workflowFiles.length > 0
        ? [
            {
              title: "workflow file",
              tooltipTitle: "workflow files",
              iconSlug: "file-code",
              values: configuration.workflowFiles.map((file) => ({
                badges: [{ label: file, bgColor: "bg-gray-100", textColor: "text-gray-700" }],
              })),
            },
          ]
        : undefined;

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: githubIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
      specs,
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnWorkflowRunEventData;
      const workflowName =
        eventData?.workflow_run?.display_title ||
        eventData?.workflow_run?.name ||
        eventData?.workflow?.name ||
        "Workflow";
      const conclusion = eventData?.workflow_run?.conclusion || "";

      props.lastEventData = {
        title: workflowName,
        subtitle: buildGithubSubtitle(conclusion, lastEvent.createdAt),
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
