import { ComponentsNode, TriggersTrigger, WorkflowsWorkflowEvent } from "@/api-client";
import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerRenderer } from "../types";
import githubIcon from "@/assets/icons/integrations/github.svg";
import { TriggerProps } from "@/ui/trigger";

interface OnIssueMetadata {
  repository: {
    id: string;
    name: string;
    url: string;
  };
}

interface OnIssueConfiguration {
  actions: string[];
}

interface OnIssueEventData {
  action?: string;
  issue?: {
    id?: number;
    number?: number;
    title?: string;
    html_url?: string;
    state?: string;
    user?: {
      id: number;
      login: string;
    };
  };
}

/**
 * Renderer for the "github.onIssue" trigger
 */
export const onIssueTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (event: WorkflowsWorkflowEvent): { title: string; subtitle: string } => {
    const eventData = event.data as OnIssueEventData;

    return {
      title: `#${eventData?.issue?.number} - ${eventData?.issue?.title}`,
      subtitle: eventData?.action || "",
    };
  },

  getRootEventValues: (lastEvent: WorkflowsWorkflowEvent): Record<string, string> => {
    const eventData = lastEvent.data as OnIssueEventData;

    return {
      URL: eventData?.issue?.html_url || "",
      Title: eventData?.issue?.title || "",
      Action: eventData?.action || "",
      Author: eventData?.issue?.user?.login || "",
      State: eventData?.issue?.state || "",
    };
  },

  getTriggerProps: (node: ComponentsNode, trigger: TriggersTrigger, lastEvent: WorkflowsWorkflowEvent) => {
    const metadata = node.metadata as unknown as OnIssueMetadata;
    const configuration = node.configuration as unknown as OnIssueConfiguration;
    const metadataItems = [];

    if (metadata?.repository?.name) {
      metadataItems.push({
        icon: "book",
        label: metadata.repository.name,
      });
    }

    if (configuration?.actions) {
      metadataItems.push({
        icon: "funnel",
        label: configuration.actions.join(", "),
      });
    }

    const props: TriggerProps = {
      title: node.name!,
      iconSrc: githubIcon,
      iconBackground: "bg-white",
      iconColor: getColorClass(trigger.color),
      headerColor: getBackgroundColorClass(trigger.color),
      collapsedBackground: getBackgroundColorClass(trigger.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnIssueEventData;

      props.lastEventData = {
        title: `#${eventData?.issue?.number} - ${eventData?.issue?.title}`,
        subtitle: eventData?.action || "",
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
