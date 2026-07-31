import { getBackgroundColorClass } from "@/lib/colors";
import React from "react";
import type {
  CustomFieldRenderer,
  NodeInfo,
  TriggerEventContext,
  TriggerRenderer,
  TriggerRendererContext,
} from "../types";
import type { TriggerProps } from "@/ui/trigger";
import dockerIcon from "@/assets/icons/integrations/docker.svg";
import type { Repository, RepositoryMetadata } from "./types";
import { renderTimeAgo } from "@/components/TimeAgo";
import { formatTimestampInUserTimezone } from "@/lib/timezone";
import type { Predicate } from "../utils";
import { formatPredicate, stringOrDash } from "../utils";
import type { MetadataItem } from "@/ui/metadataList";

export interface OnImagePushMetadata {
  repository?: RepositoryMetadata;
  webhookUrl?: string;
}

export interface OnImagePushConfiguration {
  repository?: string;
  tags?: Predicate[];
}

interface PushData {
  tag?: string;
  pushed_at?: number;
  pusher?: string;
}

interface ImagePushEvent {
  callback_url?: string;
  push_data?: PushData;
  repository?: Repository;
}

/**
 * Renderer for the "dockerhub.onImagePush" trigger
 */
export const onImagePushTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string | React.ReactNode } => {
    const eventData = context.event?.data as ImagePushEvent;
    const repository = eventData?.repository?.repo_name;
    const tag = eventData?.push_data?.tag;

    const title = repository ? `${repository}${tag ? `:${tag}` : ""}` : "Image push";
    const subtitle = context.event?.createdAt ? renderTimeAgo(new Date(context.event?.createdAt || "")) : "";

    return { title, subtitle };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as ImagePushEvent;
    const repository = eventData?.repository;
    const pushData = eventData?.push_data;
    const pushedAt = pushData?.pushed_at ? new Date(pushData.pushed_at * 1000).toISOString() : undefined;

    const visibility = repository?.is_private === undefined ? "-" : repository.is_private ? "Private" : "Public";

    return {
      Repository: stringOrDash(repository?.repo_name),
      Tag: stringOrDash(pushData?.tag),
      Pusher: stringOrDash(pushData?.pusher),
      "Pushed At": pushedAt ? formatTimestampInUserTimezone(pushedAt) : "-",
      "Repository URL": stringOrDash(repository?.repo_url),
      Visibility: visibility,
      Stars: stringOrDash(repository?.star_count),
      Pulls: stringOrDash(repository?.pull_count),
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as OnImagePushMetadata | undefined;
    const configuration = node.configuration as OnImagePushConfiguration | undefined;
    const metadataItems: MetadataItem[] = [];

    if (metadata?.repository) {
      metadataItems.push({
        icon: "package",
        label: getRepositoryLabel(metadata),
      });
    }

    if (configuration?.tags?.length) {
      metadataItems.push({
        icon: "tag",
        label: configuration.tags.map(formatPredicate).join(", "),
      });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: dockerIcon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const { title, subtitle } = onImagePushTriggerRenderer.getTitleAndSubtitle({ event: lastEvent });
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

export const onImagePushCustomFieldRenderer: CustomFieldRenderer = {
  render: (node: NodeInfo) => {
    const metadata = node.metadata as OnImagePushMetadata | undefined;
    const repositoryLabel = getRepositoryLabel(metadata);
    const repositoryUrl = `https://hub.docker.com/repository/docker/${repositoryLabel}/webhooks`;
    const webhookUrl = metadata?.webhookUrl || "[URL GENERATED ONCE THE CANVAS IS SAVED]";

    return (
      <div className="border-t-1 border-edge-default pt-4">
        <div className="space-y-3">
          <div>
            <span className="text-sm font-medium text-content-secondary">DockerHub Webhook Setup</span>
            <div className="mt-2 rounded-md border-1 border-edge-default bg-surface-subtle px-2.5 py-2 text-xs text-content-primary">
              <ol className="list-decimal ml-4 space-y-1">
                <li>
                  Go to the{" "}
                  <a href={repositoryUrl} target="_blank" rel="noopener noreferrer">
                    {repositoryLabel}
                  </a>{" "}
                  webhooks page
                </li>
                <li>Add webhook</li>
                <li>Set the webhook URL below and save</li>
              </ol>
              <div className="mt-3">
                <span className="text-xs font-medium text-content-secondary">Webhook URL</span>
                <div className="relative group mt-1">
                  <pre className="rounded-md border-1 border-edge-default bg-surface-raised px-2.5 py-2 font-mono text-xs text-content-primary whitespace-pre-wrap break-all">
                    {webhookUrl}
                  </pre>
                </div>
              </div>
              <p className="mt-3">DockerHub will send tag push events to SuperPlane once the webhook is configured.</p>
            </div>
          </div>
        </div>
      </div>
    );
  },
};

function getRepositoryLabel(metadata?: OnImagePushMetadata): string | undefined {
  return metadata?.repository?.namespace
    ? `${metadata.repository.namespace}/${metadata.repository.name}`
    : metadata?.repository?.name;
}
