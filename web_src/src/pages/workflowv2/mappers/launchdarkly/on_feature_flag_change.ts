import { getBackgroundColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import launchdarklyIcon from "@/assets/icons/integrations/launchdarkly.svg";

const eventLabels: Record<string, string> = {
  flag: "Feature flag change",
};

function formatEventLabel(event: string): string {
  return eventLabels[event] ?? event;
}

interface OnFeatureFlagChangeEventData {
  kind?: string;
  name?: string;
  title?: string;
  titleVerb?: string;
}

export const onFeatureFlagChangeTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as OnFeatureFlagChangeEventData;
    const title = eventData?.title || eventData?.name || "Feature Flag";
    const verb = eventData?.titleVerb;
    const kind = eventData?.kind ? formatEventLabel(eventData.kind) : "";
    const contentParts = [verb || kind].filter(Boolean).join(" · ");
    const subtitle = buildSubtitle(contentParts, context.event?.createdAt);
    return { title, subtitle };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnFeatureFlagChangeEventData;
    const details: Record<string, string> = {};
    if (eventData?.kind) details["Event"] = formatEventLabel(eventData.kind);
    if (eventData?.name) details["Flag"] = eventData.name;
    if (eventData?.titleVerb) details["Action"] = eventData.titleVerb;
    return details;
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as { events?: string[] };
    const metadataItems: { icon: string; label: string }[] = [];
    if (configuration?.events?.length) {
      const formattedEvents = configuration.events.map(formatEventLabel).join(", ");
      metadataItems.push({ icon: "funnel", label: "Events: " + formattedEvents });
    }
    const props: TriggerProps = {
      title: node.name!,
      iconSrc: launchdarklyIcon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };
    if (lastEvent) {
      const eventData = lastEvent.data as OnFeatureFlagChangeEventData;
      const title = eventData?.title || eventData?.name || "Feature Flag";
      const verb = eventData?.titleVerb;
      const kind = eventData?.kind ? formatEventLabel(eventData.kind) : "";
      const contentParts = [verb || kind].filter(Boolean).join(" · ");
      const subtitle = buildSubtitle(contentParts, lastEvent.createdAt);
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

function buildSubtitle(content: string, createdAt?: string): string {
  const timeAgo = createdAt ? formatTimeAgo(new Date(createdAt)) : "";
  return content && timeAgo ? content + " · " + timeAgo : content || timeAgo;
}
