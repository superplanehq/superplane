import { TriggerRenderer, TriggerEventContext, TriggerRendererContext } from "../types";
import { getColorClass } from "@/utils/colors";
import snykIcon from "@/assets/icons/integrations/snyk.svg";
import { TriggerProps } from "@/ui/trigger";

interface OnNewIssueDetectedConfiguration {
  projectId?: string;
  severity?: string[];
}

interface OnNewIssueDetectedEventData {
  issue?: {
    id: string;
    title: string;
    severity: string;
    description: string;
    packageName: string;
    packageVersion: string;
  };
  project?: {
    id: string;
    name: string;
  };
  timestamp?: string;
}

export const onNewIssueDetectedTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as OnNewIssueDetectedEventData;

    if (eventData?.issue) {
      const issueId = eventData.issue.id || "unknown";
      const severity = eventData.issue.severity || "unknown";

      return {
        title: `New issue detected: ${issueId}`,
        subtitle: `${severity.charAt(0).toUpperCase() + severity.slice(1)} severity`,
      };
    }

    return { title: "On New Issue Detected", subtitle: "Event received" };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, unknown> => {
    const eventData = context.event?.data as OnNewIssueDetectedEventData;
    const values: Record<string, unknown> = {};

    if (eventData?.issue) {
      values["Issue ID"] = eventData.issue.id;
      values["Issue Title"] = eventData.issue.title;
      values["Severity"] = eventData.issue.severity;
      values["Package"] = `${eventData.issue.packageName}@${eventData.issue.packageVersion}`;
    }

    if (eventData?.project) {
      values["Project"] = eventData.project.name;
      values["Project ID"] = eventData.project.id;
    }

    return values;
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as OnNewIssueDetectedConfiguration;
    const metadata = node.metadata as { projectId?: { id: string; name: string } } | undefined;

    const metadataItems = [];

    if (metadata?.projectId?.name) {
      metadataItems.push({
        icon: "book",
        label: metadata.projectId.name,
      });
    }

    if (configuration?.severity && configuration.severity.length > 0) {
      metadataItems.push({
        icon: "alert-circle",
        label: configuration.severity.join(", "),
      });
    }

    const props: TriggerProps = {
      title: node.name || "On New Issue Detected",
      iconSrc: snykIcon,
      iconColor: getColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnNewIssueDetectedEventData;
      const issueId = eventData?.issue?.id || "unknown";
      const severity = eventData?.issue?.severity || "";

      props.lastEventData = {
        title: `New issue detected: ${issueId}`,
        subtitle: severity ? `${severity.charAt(0).toUpperCase() + severity.slice(1)} severity` : "",
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id!,
      };
    }

    return props;
  },
};

export default onNewIssueDetectedTriggerRenderer;
