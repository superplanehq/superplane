import { getBackgroundColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../../types";
import { TriggerProps } from "@/ui/trigger";
import awsCodePipelineIcon from "@/assets/icons/integrations/aws.codepipeline.svg";
import { formatTimeAgo } from "@/utils/date";
import { MetadataItem } from "@/ui/metadataList";
import { formatPredicate, Predicate, stringOrDash } from "../../utils";

interface OnPipelineConfiguration {
  region?: string;
  pipelines?: Predicate[];
  states?: string[];
}

interface PipelineExecutionEvent {
  account?: string;
  region?: string;
  detail?: {
    pipeline?: string;
    state?: string;
    "execution-id"?: string;
  };
}

function buildMetadataItems(configuration?: OnPipelineConfiguration): MetadataItem[] {
  const items: MetadataItem[] = [];

  if (configuration?.region) {
    items.push({ icon: "globe", label: configuration.region });
  }

  if (configuration?.pipelines && configuration.pipelines.length > 0) {
    items.push({
      icon: "funnel",
      label: configuration.pipelines.map(formatPredicate).join(", "),
    });
  }

  if (configuration?.states && configuration.states.length > 0) {
    items.push({ icon: "tag", label: configuration.states.join(", ") });
  }

  return items;
}

export const onPipelineTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as PipelineExecutionEvent;
    const pipeline = eventData?.detail?.pipeline;
    const state = eventData?.detail?.state;

    let title = "CodePipeline execution";
    if (pipeline && state) {
      title = `${pipeline} - ${state}`;
    } else if (pipeline) {
      title = pipeline;
    }

    const subtitle = context.event?.createdAt ? formatTimeAgo(new Date(context.event.createdAt)) : "";
    return { title, subtitle };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as PipelineExecutionEvent;
    const detail = eventData?.detail;

    return {
      Pipeline: stringOrDash(detail?.pipeline),
      State: stringOrDash(detail?.state),
      "Execution ID": stringOrDash(detail?.["execution-id"]),
      Region: stringOrDash(eventData?.region),
      Account: stringOrDash(eventData?.account),
    };
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const configuration = node.configuration as OnPipelineConfiguration | undefined;

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: awsCodePipelineIcon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: buildMetadataItems(configuration),
    };

    if (lastEvent) {
      const { title, subtitle } = onPipelineTriggerRenderer.getTitleAndSubtitle({ event: lastEvent });
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
