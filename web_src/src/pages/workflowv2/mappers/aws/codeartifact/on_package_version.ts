import { ComponentsNode, TriggersTrigger, CanvasesCanvasEvent } from "@/api-client";
import { getBackgroundColorClass } from "@/utils/colors";
import { TriggerRenderer } from "../../types";
import { TriggerProps } from "@/ui/trigger";
import awsIcon from "@/assets/icons/integrations/aws.svg";
import {
  CodeArtifactPackageVersionDetail,
  CodeArtifactPackageVersionEvent,
  CodeArtifactTriggerConfiguration,
  CodeArtifactTriggerMetadata,
} from "./types";
import { buildCodeArtifactMetadataItems, formatPackageLabel, formatPackageName } from "./utils";
import { formatTimeAgo } from "@/utils/date";
import { numberOrZero, stringOrDash } from "../../utils";

/**
 * Renderer for the "aws.codeArtifact.onPackageVersion" trigger
 */
export const onPackageVersionTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (event: CanvasesCanvasEvent): { title: string; subtitle: string } => {
    const eventData = event.data?.data as CodeArtifactPackageVersionEvent;
    const detail = eventData?.detail;
    const packageLabel = formatPackageLabel(undefined, undefined, detail);

    const title = packageLabel || "CodeArtifact package version";
    const subtitle = event.createdAt ? formatTimeAgo(new Date(event.createdAt)) : "";

    return { title, subtitle };
  },

  getRootEventValues: (event: CanvasesCanvasEvent): Record<string, string> => {
    const eventData = event.data?.data as CodeArtifactPackageVersionEvent;
    const detail = eventData?.detail as CodeArtifactPackageVersionDetail;

    const values: Record<string, string> = {
      Domain: stringOrDash(detail?.domainName),
      Repository: stringOrDash(detail?.repositoryName),
      "Package Format": stringOrDash(detail?.packageFormat),
      Namespace: stringOrDash(detail?.packageNamespace ?? undefined),
      Package: stringOrDash(formatPackageName(detail?.packageNamespace, detail?.packageName)),
      Version: stringOrDash(detail?.packageVersion),
      State: stringOrDash(detail?.packageVersionState),
      Operation: stringOrDash(detail?.operationType),
      Region: stringOrDash(eventData?.region),
      Account: stringOrDash(eventData?.account),
    };

    const changes = detail?.changes;
    if (changes) {
      values["Assets Added"] = numberOrZero(changes.assetsAdded).toString();
      values["Assets Removed"] = numberOrZero(changes.assetsRemoved).toString();
      values["Assets Updated"] = numberOrZero(changes.assetsUpdated).toString();
      values["Metadata Updated"] = stringOrDash(changes.metadataUpdated);
      values["Status Changed"] = stringOrDash(changes.statusChanged);
    }

    return values;
  },

  getTriggerProps: (node: ComponentsNode, trigger: TriggersTrigger, lastEvent: CanvasesCanvasEvent) => {
    const metadata = node.metadata as CodeArtifactTriggerMetadata | undefined;
    const configuration = node.configuration as CodeArtifactTriggerConfiguration | undefined;
    const metadataItems = buildCodeArtifactMetadataItems(metadata, configuration);

    const props: TriggerProps = {
      title: node.name!,
      iconSrc: awsIcon,
      collapsedBackground: getBackgroundColorClass(trigger.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const { title, subtitle } = onPackageVersionTriggerRenderer.getTitleAndSubtitle(lastEvent);
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
