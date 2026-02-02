import { ComponentsNode, TriggersTrigger, CanvasesCanvasEvent } from "@/api-client";
import { getBackgroundColorClass } from "@/utils/colors";
import { TriggerRenderer } from "../../types";
import { TriggerProps } from "@/ui/trigger";
import awsEcrIcon from "@/assets/icons/integrations/aws.ecr.svg";
import { EcrImageScanEvent, EcrTriggerConfiguration, EcrTriggerMetadata } from "./types";
import {
  buildRepositoryMetadataItems,
  formatTagLabel,
  formatTags,
  getRepositoryLabel,
  numberOrZero,
  stringOrDash,
} from "./utils";
import { formatTimeAgo } from "@/utils/date";
import { EcrImageScanDetail } from "./types";

/**
 * Renderer for the "aws.ecr.onImageScan" trigger
 */
export const onImageScanTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (event: CanvasesCanvasEvent): { title: string; subtitle: string } => {
    const eventData = event.data?.data as EcrImageScanEvent;
    const detail = eventData?.detail;
    const repository = getRepositoryLabel(undefined, undefined, detail?.["repository-name"]);
    const tagLabel = formatTagLabel(detail?.["image-tags"]);

    const title = repository ? `${repository}${tagLabel ? `:${tagLabel}` : ""}` : "ECR image scan";
    const subtitle = event.createdAt ? formatTimeAgo(new Date(event.createdAt)) : "";

    return { title, subtitle };
  },

  getRootEventValues: (event: CanvasesCanvasEvent): Record<string, string> => {
    const eventData = event.data?.data as EcrImageScanEvent;
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

  getTriggerProps: (node: ComponentsNode, trigger: TriggersTrigger, lastEvent: CanvasesCanvasEvent) => {
    const metadata = node.metadata as EcrTriggerMetadata | undefined;
    const configuration = node.configuration as EcrTriggerConfiguration | undefined;
    const metadataItems = buildRepositoryMetadataItems(metadata, configuration);

    const props: TriggerProps = {
      title: node.name!,
      iconSrc: awsEcrIcon,
      collapsedBackground: getBackgroundColorClass(trigger.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const { title, subtitle } = onImageScanTriggerRenderer.getTitleAndSubtitle(lastEvent);
      props.lastEventData = {
        title,
        subtitle,
        receivedAt: new Date(lastEvent.createdAt!),
        state: "triggered",
        eventId: lastEvent.id!,
      };
    }

    return props;
  },
};
