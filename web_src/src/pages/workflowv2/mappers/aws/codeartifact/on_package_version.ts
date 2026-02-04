import { getBackgroundColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../../types";
import { TriggerProps } from "@/ui/trigger";
import awsCodeArtifactIcon from "@/assets/icons/integrations/aws.codeartifact.svg";
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
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data?.data as CodeArtifactPackageVersionEvent;
    const detail = eventData?.detail;
    const packageLabel = formatPackageLabel(detail);
    const title = packageLabel || "CodeArtifact package version";
    const subtitle = formatTimeAgo(new Date(context.event?.createdAt || ""));

    return { title, subtitle };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data?.data as CodeArtifactPackageVersionEvent;
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

  getTriggerProps: (context: TriggerRendererContext) => {
    const { node, definition, lastEvent } = context;
    const metadata = node.metadata as CodeArtifactTriggerMetadata | undefined;
    const configuration = node.configuration as CodeArtifactTriggerConfiguration | undefined;
    const metadataItems = buildCodeArtifactMetadataItems(metadata, configuration);

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: awsCodeArtifactIcon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const { title, subtitle } = onPackageVersionTriggerRenderer.getTitleAndSubtitle({
        event: {
          nodeId: node.id!,
          id: lastEvent.id!,
          createdAt: lastEvent.createdAt!,
          data: lastEvent.data || {},
        },
      });
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
