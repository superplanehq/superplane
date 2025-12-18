import { ComponentsNode, TriggersTrigger, WorkflowsWorkflowEvent } from "@/api-client";
import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerRenderer } from "../types";
import githubIcon from "@/assets/icons/integrations/github.svg";
import { TriggerProps } from "@/ui/trigger";

interface OnPackageMetadata {
  repository: {
    id: string;
    name: string;
    url: string;
  };
}

interface OnPackageConfiguration {
  actions: string[];
}

interface OnPackageEventData {
  action?: string;
  package?: {
    id?: number;
    name?: string;
    package_type?: string;
    package_version?: {
      id?: number;
      html_url?: string;
      version?: string;
      package_url?: string;
    };
  };
  repository?: {
    name?: string;
    full_name?: string;
  };
  sender?: {
    login?: string;
  };
}

/**
 * Renderer for the "github.onPackage" trigger
 */
export const onPackageTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (event: WorkflowsWorkflowEvent): { title: string; subtitle: string } => {
    const eventData = event.data as OnPackageEventData;

    return {
      title: eventData?.package?.package_version?.package_url || eventData?.package?.name || "",
      subtitle: eventData?.package?.package_version?.version || "",
    };
  },

  getRootEventValues: (lastEvent: WorkflowsWorkflowEvent): Record<string, string> => {
    const eventData = lastEvent.data as OnPackageEventData;

    return {
      Action: eventData?.action || "",
      Type: eventData?.package?.package_type || "",
      Package: eventData?.package?.name || "",
      "Package URL": eventData?.package?.package_version?.package_url || "",
      Version: eventData?.package?.package_version?.version || "",
      URL: eventData?.package?.package_version?.html_url || "",
    };
  },

  getTriggerProps: (node: ComponentsNode, trigger: TriggersTrigger, lastEvent: WorkflowsWorkflowEvent) => {
    const metadata = node.metadata as unknown as OnPackageMetadata;
    const configuration = node.configuration as unknown as OnPackageConfiguration;
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
      const eventData = lastEvent.data as OnPackageEventData;

      props.lastEventData = {
        title: eventData?.package?.package_version?.package_url || eventData?.package?.name || "",
        subtitle: eventData?.package?.package_version?.version || "",
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
