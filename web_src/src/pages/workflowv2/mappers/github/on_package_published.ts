import { ComponentsNode, TriggersTrigger, WorkflowsWorkflowEvent } from "@/api-client";
import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerRenderer } from "../types";
import githubIcon from "@/assets/icons/integrations/github.svg";
import { TriggerProps } from "@/ui/trigger";
import { BaseNodeMetadata } from "./types";
import { Predicate, formatPredicate } from "./utils";

interface OnPackagePublishedConfiguration {
  packageNames: Predicate[];
  packageTypes?: string[];
}

interface PackagePublishedEventData {
  action?: string;
  package?: {
    name?: string;
    package_type?: string;
    package_version?: {
      version?: string;
      summary?: string;
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
 * Renderer for the "github.onPackagePublished" trigger
 */
export const onPackagePublishedTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (event: WorkflowsWorkflowEvent): { title: string; subtitle: string } => {
    const eventData = event.data?.data as PackagePublishedEventData;
    const packageName = eventData?.package?.name || "Package";
    const packageType = eventData?.package?.package_type || "";
    const version = eventData?.package?.package_version?.version || "";

    return {
      title: `${packageName}${version ? `@${version}` : ""}`,
      subtitle: packageType ? `${packageType} package` : "Package published",
    };
  },

  getRootEventValues: (lastEvent: WorkflowsWorkflowEvent): Record<string, string> => {
    const eventData = lastEvent.data?.data as PackagePublishedEventData;

    return {
      Package: eventData?.package?.name || "",
      Version: eventData?.package?.package_version?.version || "",
      Type: eventData?.package?.package_type || "",
      Repository: eventData?.repository?.full_name || "",
      Sender: eventData?.sender?.login || "",
    };
  },

  getTriggerProps: (node: ComponentsNode, trigger: TriggersTrigger, lastEvent: WorkflowsWorkflowEvent) => {
    const metadata = node.metadata as unknown as BaseNodeMetadata;
    const configuration = node.configuration as unknown as OnPackagePublishedConfiguration;
    const metadataItems = [];

    if (metadata?.repository?.name) {
      metadataItems.push({
        icon: "book",
        label: metadata.repository.name,
      });
    }

    if (configuration?.packageNames && configuration.packageNames.length > 0) {
      metadataItems.push({
        icon: "funnel",
        label: configuration.packageNames.map(formatPredicate).join(", "),
      });
    }

    if (configuration?.packageTypes && configuration.packageTypes.length > 0) {
      metadataItems.push({
        icon: "tag",
        label: configuration.packageTypes.join(", "),
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
      const eventData = lastEvent.data?.data as PackagePublishedEventData;
      const packageName = eventData?.package?.name || "Package";
      const packageType = eventData?.package?.package_type || "";
      const version = eventData?.package?.package_version?.version || "";

      props.lastEventData = {
        title: `${packageName}${version ? `@${version}` : ""}`,
        subtitle: packageType ? `${packageType} package` : "Package published",
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
