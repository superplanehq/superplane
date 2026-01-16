import { ComponentsNode, TriggersTrigger, WorkflowsWorkflowEvent } from "@/api-client";
import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerRenderer } from "../types";
import { TriggerProps } from "@/ui/trigger";
import SemaphoreLogo from "@/assets/semaphore-logo-sign-black.svg";

interface OnPipelineDoneMetadata {
  project?: {
    id: string;
    name: string;
    url: string;
  };
}

interface OnPipelineDoneEventData {
  project?: {
    name: string;
  };
  repository?: {
    slug: string;
  };
  pipeline?: {
    name: string;
    state: string;
    result: string;
    done_at: string;
  };
}

/**
 * Renderer for the "semaphore.onPipelineDone" trigger type
 */
export const onPipelineDoneTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (event: WorkflowsWorkflowEvent): { title: string; subtitle: string } => {
    const eventData = event.data?.data as OnPipelineDoneEventData;

    return {
      title: eventData?.pipeline?.name || "",
      subtitle: eventData?.pipeline?.result || "",
    };
  },

  getRootEventValues: (lastEvent: WorkflowsWorkflowEvent): Record<string, string> => {
    const eventData = lastEvent.data?.data as OnPipelineDoneEventData;

    return {
      Project: eventData?.project?.name || "",
      Repository: eventData?.repository?.slug || "",
      Pipeline: eventData?.pipeline?.name || "",
      Result: eventData?.pipeline?.result || "",
      "Done At": eventData?.pipeline?.done_at || "",
    };
  },

  getTriggerProps: (node: ComponentsNode, trigger: TriggersTrigger, lastEvent: WorkflowsWorkflowEvent) => {
    const metadata = node.metadata as unknown as OnPipelineDoneMetadata;
    const metadataItems = [];

    if (metadata?.project?.name) {
      metadataItems.push({
        icon: "book",
        label: metadata.project.name,
      });
    }

    const props: TriggerProps = {
      title: node.name!,
      iconSrc: SemaphoreLogo,
      iconBackground: "bg-white",
      iconColor: getColorClass(trigger.color),
      headerColor: "",
      collapsedBackground: getBackgroundColorClass(trigger.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data?.data as OnPipelineDoneEventData;
      props.lastEventData = {
        title: eventData.pipeline?.name || "",
        subtitle: eventData.pipeline?.result || "",
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
