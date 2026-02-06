import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../types";
import gitlabIcon from "@/assets/icons/integrations/gitlab.svg";
import { TriggerProps } from "@/ui/trigger";
import { GitLabNodeMetadata } from "./types";
import { buildGitlabSubtitle } from "./utils";

interface OnIssueConfiguration {
  actions: string[];
  project: string;
}

interface GitLabIssue {
  id: number;
  iid: number;
  title: string;
  description?: string;
  state: string;
  action: string;
  url: string;
}

interface OnIssueEventData {
  object_kind?: string;
  event_type?: string;
  object_attributes?: GitLabIssue;
  user?: {
    id: number;
    name: string;
    username: string;
  };
  project?: {
    id: number;
    name: string;
    path_with_namespace: string;
    web_url: string;
  };
}

function getDetailsForIssue(issue: GitLabIssue | undefined): Record<string, string> {
  if (!issue) {
    return {};
  }

  return {
    URL: issue.url || "",
    Title: issue.title || "",
    Action: issue.action || "",
    State: issue.state || "",
    IID: issue.iid?.toString() || "",
  };
}

export const onIssueTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as OnIssueEventData;
    const issue = eventData?.object_attributes;

    return {
      title: `#${issue?.iid} - ${issue?.title || "Issue"}`,
      subtitle: buildGitlabSubtitle(issue?.action || "", context.event?.createdAt),
    };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as OnIssueEventData;
    const issue = eventData?.object_attributes;
    const values = getDetailsForIssue(issue);

    if (eventData?.user?.username) {
      values["Author"] = eventData.user.username;
    }

    if (eventData?.project?.path_with_namespace) {
      values["Project"] = eventData.project.path_with_namespace;
    }

    return values;
  },

  getTriggerProps: (context: TriggerRendererContext): TriggerProps => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as unknown as GitLabNodeMetadata;
    const configuration = node.configuration as unknown as OnIssueConfiguration;
    const metadataItems = [];

    if (metadata?.repository?.name) {
      metadataItems.push({
        icon: "book",
        label: metadata.repository.name,
      });
    }

    if (configuration?.actions) {
      metadataItems.push({
        icon: "funnel",
        label: configuration.actions.join(", "),
      });
    }

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: gitlabIcon,
      iconColor: getColorClass(definition.color),
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const eventData = lastEvent.data as OnIssueEventData;
      const issue = eventData?.object_attributes;

      props.lastEventData = {
        title: `#${issue?.iid} - ${issue?.title || "Issue"}`,
        subtitle: buildGitlabSubtitle(issue?.action || "", lastEvent.createdAt),
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id!,
      };
    }

    return props;
  },
};
