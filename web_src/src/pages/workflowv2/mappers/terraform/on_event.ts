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

    const values: Record<string, string> = {
      "Run ID": eventData?.runId || "",
      Workspace: eventData?.workspaceName || "",
      Action: eventData?.action || "",
      Status: eventData?.runStatus || "",
      "Created By": eventData?.runCreatedBy || "",
      URL: eventData?.runUrl || "",
    };

    if (
      eventData?.additions !== undefined ||
      eventData?.changes !== undefined ||
      eventData?.destructions !== undefined
    ) {
      values["Resources Added"] = String(eventData?.additions ?? 0);
      values["Resources Changed"] = String(eventData?.changes ?? 0);
      values["Resources Destroyed"] = String(eventData?.destructions ?? 0);
    }

    return values;
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

    const config = node.configuration as Record<string, any>;
    const nodeMetadata = node.metadata as Record<string, any>;
    if (nodeMetadata?.workspace?.name) {
      props.metadata!.push({ icon: "box", label: nodeMetadata.workspace.name });
    } else if (config?.workspaceId) {
      props.metadata!.push({ icon: "box", label: config.workspaceId });
    }

    if (lastEvent) {
      const eventData = lastEvent.data as TerraformEventData;
      const hasDiff =
        eventData?.additions !== undefined || eventData?.changes !== undefined || eventData?.destructions !== undefined;
      const diffSummary = hasDiff
        ? ` | +${eventData?.additions ?? 0} ~${eventData?.changes ?? 0} -${eventData?.destructions ?? 0}`
        : "";
      props.lastEventData = {
        title: eventData?.runMessage || "Terraform Run",
        subtitle: `Action: ${eventData?.action || "Unknown"} | Workspace: ${eventData?.workspaceName || "Unknown"}${diffSummary}`,
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id!,
      };
    }

    return props;
  },
};
