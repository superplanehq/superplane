import { getBackgroundColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../../types";
import { TriggerProps } from "@/ui/trigger";
import awsEcrIcon from "@/assets/icons/integrations/aws.ecr.svg";
import { EcrImageScanEvent, EcrTriggerConfiguration, EcrTriggerMetadata } from "./types";
import { buildRepositoryMetadataItems, formatTagLabel, formatTags, getRepositoryLabel } from "./utils";
import { formatTimeAgo } from "@/utils/date";
import { EcrImageScanDetail } from "./types";
import { numberOrZero, stringOrDash } from "../../utils";

/**
 * Renderer for the "aws.ecr.onImageScan" trigger
 */
export const onImageScanTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event.data as EcrImageScanEvent;
    const detail = eventData?.detail;
    const repository = getRepositoryLabel(undefined, undefined, detail?.["repository-name"]);
    const tagLabel = formatTagLabel(detail?.["image-tags"]);

    const title = repository ? `${repository}${tagLabel ? `:${tagLabel}` : ""}` : "ECR image scan";
    const subtitle = context.event.createdAt ? formatTimeAgo(new Date(context.event.createdAt)) : "";

    return { title, subtitle };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event.data as EcrImageScanEvent;
    const detail = eventData?.detail as EcrImageScanDetail;

    let values: Record<string, string> = {
      Repository: stringOrDash(getRepositoryLabel(undefined, undefined, detail?.["repository-name"])),
      "Image Tags": formatTags(detail?.["image-tags"]),
      "Image Digest": stringOrDash(detail?.["image-digest"]),
      "Scan Status": stringOrDash(detail?.["scan-status"]),
      Region: stringOrDash(eventData?.region),
      Account: stringOrDash(eventData?.account),
    };

    const severityCounts = detail["finding-severity-counts"];
    if (severityCounts) {
      values["Critical"] = numberOrZero(severityCounts.CRITICAL).toString();
      values["High"] = numberOrZero(severityCounts.HIGH).toString();
      values["Medium"] = numberOrZero(severityCounts.MEDIUM).toString();
      values["Low"] = numberOrZero(severityCounts.LOW).toString();
    }

    return values;
  },

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as EcrTriggerMetadata | undefined;
    const configuration = node.configuration as EcrTriggerConfiguration | undefined;
    const metadataItems = buildRepositoryMetadataItems(metadata, configuration);

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: awsEcrIcon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const { title, subtitle } = onImageScanTriggerRenderer.getTitleAndSubtitle({ event: lastEvent });
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
