import { getColorClass, getBackgroundColorClass } from "@/lib/colors";
import type { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import githubIcon from "@/assets/icons/integrations/github.svg";
import type { TriggerProps } from "@/ui/trigger";
import type { BaseNodeMetadata } from "./types";
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
  getTitleAndSubtitle: (context: TriggerEventContext) => {
    const eventData = context.event?.data as OnWorkflowRunEventData;

    return {
      title: workflowRunTitle(eventData),
      subtitle: buildGithubSubtitle(workflowRunConclusion(eventData), context.event?.createdAt),
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnWorkflowRunEventData;

    return {
      "Received at": receivedAtLabel(context.event?.createdAt),
      Conclusion: workflowRunConclusion(eventData),
      "Triggered by": workflowRunEvent(eventData),
      "Workflow link": workflowRunUrl(eventData),
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as unknown as BaseNodeMetadata;
    const configuration = node.configuration as unknown as OnWorkflowRunConfiguration;

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: githubIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: workflowRunMetadataItems(metadata?.repository?.name, configuration),
      specs: workflowFileSpecs(configuration),
    };

    if (lastEvent) {
      const { title, subtitle } = onWorkflowRunTriggerRenderer.getTitleAndSubtitle({ event: lastEvent });
      props.lastEventData = {
        title,
        subtitle,
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};

function workflowRunTitle(eventData?: OnWorkflowRunEventData): string {
  return (
    eventData?.workflow_run?.display_title || eventData?.workflow_run?.name || eventData?.workflow?.name || "Workflow"
  );
}

function workflowRunConclusion(eventData?: OnWorkflowRunEventData): string {
  return eventData?.workflow_run?.conclusion || "";
}

function workflowRunEvent(eventData?: OnWorkflowRunEventData): string {
  return eventData?.workflow_run?.event || "";
}

function workflowRunUrl(eventData?: OnWorkflowRunEventData): string {
  return eventData?.workflow_run?.html_url || "";
}

function receivedAtLabel(createdAt?: string): string {
  if (!createdAt) {
    return "";
  }

  return new Date(createdAt).toLocaleString();
}

function workflowRunMetadataItems(repositoryName: string | undefined, configuration?: OnWorkflowRunConfiguration) {
  const metadataItems = [];

  if (repositoryName) {
    metadataItems.push({
      icon: "book",
      label: repositoryName,
    });
  }

  if (configuration?.conclusions && configuration.conclusions.length > 0) {
    metadataItems.push({
      icon: "funnel",
      label: configuration.conclusions.join(", "),
    });
  }

  return metadataItems;
}

function workflowFileSpecs(configuration?: OnWorkflowRunConfiguration) {
  if (!configuration?.workflowFiles || configuration.workflowFiles.length === 0) {
    return undefined;
  }

  return [
    {
      title: "workflow file",
      tooltipTitle: "workflow files",
      iconSlug: "file-code",
      values: configuration.workflowFiles.map((file) => ({
        badges: [{ label: file, bgColor: "bg-gray-100", textColor: "text-gray-700" }],
      })),
    },
  ];
}
