import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import bitbucketIcon from "@/assets/icons/integrations/bitbucket.svg";
import { TriggerProps } from "@/ui/trigger";
import { BitbucketNodeMetadata, BitbucketPush, BitbucketPushConfiguration } from "./types";
import { Predicate, formatPredicate } from "../utils";
import { formatTimeAgo } from "@/utils/date";

function buildBitbucketSubtitle(shortSha: string, createdAt?: string): string {
  const trimmedSha = shortSha.trim();
  const timeAgo = createdAt ? formatTimeAgo(new Date(createdAt)) : "";

  if (trimmedSha && timeAgo) {
    return `${trimmedSha} · ${timeAgo}`;
  }
  return trimmedSha || timeAgo;
}

/**
 * Renderer for the "bitbucket.onPush" trigger
 */
export const onPushTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as BitbucketPush;
    const firstChange = eventData?.push?.changes?.[0];
    const commitMessage = firstChange?.new?.target?.message?.trim() || "";
    const shortSha = firstChange?.new?.target?.hash?.slice(0, 7) || "";

    return {
      title: commitMessage,
      subtitle: buildBitbucketSubtitle(shortSha, context.event?.createdAt),
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as BitbucketPush;
    const firstChange = eventData?.push?.changes?.[0];

    return {
      Branch: firstChange?.new?.name || "",
      Commit: firstChange?.new?.target?.message?.trim() || "",
      SHA: firstChange?.new?.target?.hash || "",
      Author: eventData?.actor?.display_name || "",
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as unknown as BitbucketNodeMetadata;
    const configuration = node.configuration as unknown as BitbucketPushConfiguration;
    const metadataItems = [];

    if (metadata?.repository) {
      metadataItems.push({
        icon: "book",
        label: metadata.repository.full_name || metadata.repository.name || "",
      });
    }

    if (configuration?.refs && configuration.refs.length > 0) {
      metadataItems.push({
        icon: "funnel",
        label: (configuration.refs as unknown as Predicate[]).map(formatPredicate).join(", "),
      });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: bitbucketIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data as BitbucketPush;
      const firstChange = eventData?.push?.changes?.[0];
      const shortSha = firstChange?.new?.target?.hash?.slice(0, 7) || "";

      props.lastEventData = {
        title: firstChange?.new?.target?.message?.trim() || "",
        subtitle: buildBitbucketSubtitle(shortSha, lastEvent.createdAt),
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id!,
      };
    }

    return props;
  },
};
