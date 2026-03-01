import { getBackgroundColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import { Predicate, formatPredicate, buildSubtitle } from "../utils";
import splitioIcon from "@/assets/icons/integrations/splitio.svg";

interface OnFeatureFlagChangeConfiguration {
  environments?: Predicate[];
  flags?: Predicate[];
}

interface OnFeatureFlagChangeEventData {
  name?: string;
  type?: string;
  environmentName?: string;
  description?: string;
  editor?: string;
  time?: number;
}

function getEventTitleAndSubtitle(
  eventData: OnFeatureFlagChangeEventData | undefined,
  createdAt?: string,
): { title: string; subtitle: string } {
  const title = eventData?.name || "Feature Flag";
  const contentParts = [eventData?.description].filter(Boolean).join(" · ");
  const subtitle = buildSubtitle(contentParts, createdAt);
  return { title, subtitle };
}

export const onFeatureFlagChangeTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as OnFeatureFlagChangeEventData;
    return getEventTitleAndSubtitle(eventData, context.event?.createdAt);
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnFeatureFlagChangeEventData;
    const details: Record<string, string> = {};
    if (eventData?.name) details["Flag Name"] = eventData.name;
    if (eventData?.environmentName) details["Environment"] = eventData.environmentName;
    if (eventData?.description) details["Description"] = eventData.description;
    if (eventData?.editor) details["Changed By"] = eventData.editor;
    return details;
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as OnFeatureFlagChangeConfiguration;
    const metadataItems: { icon: string; label: string }[] = [];

    if (configuration?.environments?.length) {
      metadataItems.push({
        icon: "globe",
        label: configuration.environments.map(formatPredicate).join(", "),
      });
    }

    if (configuration?.flags?.length) {
      metadataItems.push({
        icon: "flag",
        label: configuration.flags.map(formatPredicate).join(", "),
      });
    }

    const props: TriggerProps = {
      title: node.name!,
      iconSrc: splitioIcon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnFeatureFlagChangeEventData;
      const { title, subtitle } = getEventTitleAndSubtitle(eventData, lastEvent.createdAt);
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
