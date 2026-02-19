import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import TerraformCloudLogo from "@/assets/icons/integrations/terraformcloud.svg";
import { formatTimeAgo } from "@/utils/date";

interface OnRunCompletedMetadata {
  workspaceId?: string;
  workspaceName?: string;
  organization?: string;
}

interface OnRunCompletedEventData {
  run_id?: string;
  run_url?: string;
  run_message?: string;
  run_status?: string;
  workspace_id?: string;
  workspace_name?: string;
  organization_name?: string;
  notifications?: Array<{
    trigger?: string;
    run_status?: string;
    message?: string;
  }>;
}

export const onRunCompletedTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as OnRunCompletedEventData;
    const runStatus = eventData?.notifications?.[0]?.run_status || "";
    const workspaceName = eventData?.workspace_name || "Run";
    const timeAgo = context.event?.createdAt ? formatTimeAgo(new Date(context.event?.createdAt)) : "";
    const subtitle = runStatus && timeAgo ? `${runStatus} · ${timeAgo}` : runStatus || timeAgo;

    return {
      title: workspaceName,
      subtitle,
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnRunCompletedEventData;
    const notification = eventData?.notifications?.[0];

    return {
      Workspace: eventData?.workspace_name || "",
      Organization: eventData?.organization_name || "",
      "Run ID": eventData?.run_id || "",
      "Run Status": notification?.run_status || "",
      "Run URL": eventData?.run_url || "",
      Message: eventData?.run_message || "",
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as unknown as OnRunCompletedMetadata;
    const configuration = node.configuration as any;
    const metadataItems = [];

    const workspaceLabel = metadata?.workspaceName || configuration?.workspaceId;
    if (workspaceLabel) {
      metadataItems.push({
        icon: "layout-grid",
        label: workspaceLabel,
      });
    }

    const orgLabel = metadata?.organization || configuration?.organization;
    if (orgLabel) {
      metadataItems.push({
        icon: "building-2",
        label: orgLabel,
      });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: TerraformCloudLogo,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnRunCompletedEventData;
      const workspaceName = eventData?.workspace_name || "Run";
      const runStatus = eventData?.notifications?.[0]?.run_status || "";
      const timeAgo = lastEvent.createdAt ? formatTimeAgo(new Date(lastEvent.createdAt)) : "";
      const subtitle = runStatus && timeAgo ? `${runStatus} · ${timeAgo}` : runStatus || timeAgo;

      props.lastEventData = {
        title: workspaceName,
        subtitle,
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
