import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import bitbucketIcon from "@/assets/icons/integrations/bitbucket.svg";
import { TriggerProps } from "@/ui/trigger";
import { NodeMetadata } from "./types";
import { Predicate, formatPredicate } from "../utils";
import { formatTimeAgo } from "@/utils/date";

export interface OnPushConfiguration {
  repository?: string;
  refs: Predicate[];
}

export interface BitbucketPush {
  actor?: {
    display_name?: string;
    uuid?: string;
    nickname?: string;
  };
  repository?: {
    full_name?: string;
    name?: string;
    uuid?: string;
    links?: {
      html?: {
        href?: string;
      };
    };
  };
  push?: {
    changes?: BitbucketChange[];
  };
}

export interface BitbucketChange {
  new?: {
    type?: string;
    name?: string;
    target?: {
      hash?: string;
      message?: string;
      date?: string;
      author?: {
        raw?: string;
        user?: {
          display_name?: string;
          uuid?: string;
        };
      };
      links?: {
        html?: {
          href?: string;
        };
      };
    };
  };
  old?: {
    type?: string;
    name?: string;
  };
  created?: boolean;
  forced?: boolean;
  closed?: boolean;
  commits?: BitbucketCommit[];
  truncated?: boolean;
}

export interface BitbucketCommit {
  hash?: string;
  message?: string;
  author?: {
    raw?: string;
  };
  links?: {
    html?: {
      href?: string;
    };
  };
}

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
    const metadata = node.metadata as unknown as NodeMetadata;
    const configuration = node.configuration as unknown as OnPushConfiguration;
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
        label: configuration.refs.map(formatPredicate).join(", "),
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
