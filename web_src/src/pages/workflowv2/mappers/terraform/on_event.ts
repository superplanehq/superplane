import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import terraformIcon from "@/assets/icons/integrations/terraform.svg";
import { TriggerProps } from "@/ui/trigger";
import { TerraformEventData } from "./types";

export const onRunEventTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as TerraformEventData;

    return {
      title: eventData?.runMessage || "Terraform Run Event",
      subtitle: `Action: ${eventData?.action || "Unknown"} | Workspace: ${eventData?.workspaceName || "Unknown"}`,
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as TerraformEventData;

    return {
      "Run ID": eventData?.runId || "",
      Workspace: eventData?.workspaceName || "",
      Action: eventData?.action || "",
      Status: eventData?.runStatus || "",
      "Created By": eventData?.runCreatedBy || "",
      URL: eventData?.runUrl || "",
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const props: TriggerProps = {
      title: node.name || definition.label || "On Terraform Run Event",
      iconSrc: terraformIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: [],
    };

    if (lastEvent) {
      const eventData = lastEvent.data as TerraformEventData;
      props.lastEventData = {
        title: eventData?.runMessage || "Terraform Run",
        subtitle: `Action: ${eventData?.action || "Unknown"} | Workspace: ${eventData?.workspaceName || "Unknown"}`,
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id!,
      };
    }

    return props;
  },
};

export const onNeedsAttentionTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as TerraformEventData;

    return {
      title: eventData?.runMessage || "Run Needs Attention",
      subtitle: `Workspace: ${eventData?.workspaceName || "Unknown"}`,
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as TerraformEventData;

    return {
      "Run ID": eventData?.runId || "",
      Workspace: eventData?.workspaceName || "",
      Action: eventData?.action || "",
      Status: eventData?.runStatus || "",
      "Created By": eventData?.runCreatedBy || "",
      URL: eventData?.runUrl || "",
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;

    const props: TriggerProps = {
      title: node.name || definition.label || "On Run Needs Attention",
      iconSrc: terraformIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: [],
    };

    if (lastEvent) {
      const eventData = lastEvent.data as TerraformEventData;
      props.lastEventData = {
        title: eventData?.runMessage || "Run Needs Attention",
        subtitle: `Workspace: ${eventData?.workspaceName || "Unknown"}`,
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id!,
      };
    }

    return props;
  },
};
