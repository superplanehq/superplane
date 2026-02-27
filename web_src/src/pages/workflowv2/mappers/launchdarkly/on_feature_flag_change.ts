import { getBackgroundColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import { Predicate, formatPredicate, buildSubtitle } from "../utils";
import launchdarklyIcon from "@/assets/icons/integrations/launchdarkly.svg";

const eventLabels: Record<string, string> = {
  flag: "Feature flag change",
};

function formatEventLabel(event: string): string {
  return eventLabels[event] ?? event;
}

const actionLabels: Record<string, string> = {
  updateOn: "Turned on / off",
  updateTargets: "Targeting changed",
  updateRules: "Rules changed",
  updateFallthrough: "Default rule changed",
  updateOffVariation: "Off variation changed",
  createFlag: "Flag created",
  deleteFlag: "Flag deleted",
};

function formatActionLabel(action: string): string {
  return actionLabels[action] ?? action;
}

interface OnFeatureFlagChangeConfiguration {
  projectKey?: string;
  environments?: string[];
  flags?: Predicate[];
  actions?: string[];
}

interface OnFeatureFlagChangeEventData {
  kind?: string;
  name?: string;
  title?: string;
  titleVerb?: string;
  projectKey?: string;
  environmentKey?: string;
  flagKey?: string;
}

function getEventTitleAndSubtitle(
  eventData: OnFeatureFlagChangeEventData | undefined,
  createdAt?: string,
): { title: string; subtitle: string } {
  const title = eventData?.name || eventData?.flagKey || "Feature Flag";
  const verb = eventData?.titleVerb;
  const kind = eventData?.kind ? formatEventLabel(eventData.kind) : "";
  const contentParts = [verb || kind].filter(Boolean).join(" · ");
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
    if (eventData?.projectKey) details["Project"] = eventData.projectKey;
    if (eventData?.environmentKey) details["Environment"] = eventData.environmentKey;
    if (eventData?.flagKey) details["Flag Key"] = eventData.flagKey;
    if (eventData?.name) details["Flag Name"] = eventData.name;
    if (eventData?.titleVerb) details["Action"] = eventData.titleVerb;
    if (eventData?.projectKey && eventData?.flagKey) {
      details["URL"] = `https://app.launchdarkly.com/projects/${eventData.projectKey}/flags/${eventData.flagKey}`;
    }
    return details;
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as OnFeatureFlagChangeConfiguration;
    const metadataItems: { icon: string; label: string }[] = [];

    if (configuration?.projectKey) {
      metadataItems.push({ icon: "folder", label: configuration.projectKey });
    }

    if (configuration?.environments?.length) {
      metadataItems.push({
        icon: "globe",
        label: configuration.environments.join(", "),
      });
    }

    if (configuration?.flags?.length) {
      metadataItems.push({
        icon: "flag",
        label: configuration.flags.map(formatPredicate).join(", "),
      });
    }

    if (configuration?.actions?.length) {
      const formattedActions = configuration.actions.map(formatActionLabel).join(", ");
      metadataItems.push({ icon: "funnel", label: "Actions: " + formattedActions });
    }

    const props: TriggerProps = {
      title: node.name!,
      iconSrc: launchdarklyIcon,
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
