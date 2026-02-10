import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import { TriggerProps } from "@/ui/trigger";
import BuildkiteLogo from "@/assets/buildkite-logo.svg";
import { formatTimeAgo } from "@/utils/date";

interface OnBuildFinishedMetadata {
  organization?: string;
  pipeline?: string;
  branch?: string;
  appSubscriptionID?: string;
}

interface OnBuildFinishedEventData {
  build?: {
    id: string;
    state: string;
    result?: string;
    web_url?: string;
    number?: number;
    commit?: string;
    branch?: string;
    message?: string;
    blocked?: boolean;
    done_at?: string;
  };
  pipeline?: {
    id: string;
    slug: string;
    name: string;
  };
  organization?: {
    id: string;
    slug: string;
    name: string;
  };
  sender?: {
    id: string;
    name: string;
    email: string;
  };
}

/**
 * Renderer for the "buildkite.onBuildFinished" trigger type
 */
export const onBuildFinishedTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as OnBuildFinishedEventData;
    const build = eventData?.build;
    const state = build?.state || "";
    const result = build?.blocked ? "blocked" : state;
    const timeAgo = context.event?.createdAt ? formatTimeAgo(new Date(context.event?.createdAt)) : "";
    const subtitle = result && timeAgo ? `${result} · ${timeAgo}` : result || timeAgo;

    return {
      title: eventData?.pipeline?.name || eventData?.build?.web_url?.split("/").pop() || "Unknown Pipeline",
      subtitle,
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnBuildFinishedEventData;
    const build = eventData?.build;
    const pipeline = eventData?.pipeline;
    const organization = eventData?.organization;
    const sender = eventData?.sender;

    const doneAt = build?.done_at ? new Date(build.done_at).toLocaleString() : "";
    const buildUrl = build?.web_url || "";

    return {
      "Done At": doneAt,
      "Build State": build?.state || "",
      "Build Number": build?.number?.toString() || "",
      Result: build?.blocked ? "blocked" : build?.state || "",
      Project: organization?.name || "",
      Organization: organization?.slug || "",
      Pipeline: pipeline?.name || "",
      "Pipeline URL": buildUrl,
      Branch: build?.branch || "",
      Commit: build?.commit || "",
      Message: build?.message || "",
      "Triggered By": sender?.name || "",
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as unknown as OnBuildFinishedMetadata;
    const metadataItems = [];

    if (metadata?.organization) {
      metadataItems.push({
        icon: "users",
        label: metadata.organization,
      });
    }

    if (metadata?.pipeline) {
      metadataItems.push({
        icon: "layers",
        label: metadata.pipeline,
      });
    }

    if (metadata?.branch) {
      metadataItems.push({
        icon: "git-branch",
        label: metadata.branch,
      });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: BuildkiteLogo,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnBuildFinishedEventData;
      const build = eventData?.build;
      const state = build?.state || "";
      const result = build?.blocked ? "blocked" : state;
      const timeAgo = lastEvent.createdAt ? formatTimeAgo(new Date(lastEvent.createdAt)) : "";
      const subtitle = result && timeAgo ? `${result} · ${timeAgo}` : result || timeAgo;

      props.lastEventData = {
        title: eventData?.pipeline?.name || "Unknown Pipeline",
        subtitle,
        receivedAt: new Date(lastEvent.createdAt),
        state: "triggered",
        eventId: lastEvent.id,
      };
    }

    return props;
  },
};
