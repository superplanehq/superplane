import { getBackgroundColorClass } from "@/utils/colors";
import { TriggerEventContext, TriggerRenderer, TriggerRendererContext } from "../../types";
import { TriggerProps } from "@/ui/trigger";
import awsCodeArtifactIcon from "@/assets/icons/integrations/aws.codeartifact.svg";
import { PackageVersionDetail, PackageVersionEvent, Repository } from "./types";
import { formatPackageLabel, formatPackageName } from "./utils";
import { formatTimeAgo } from "@/utils/date";
import { formatPredicate, numberOrZero, Predicate, stringOrDash } from "../../utils";
import { MetadataItem } from "@/ui/metadataList";

export interface Configuration {
  region?: string;
  domain?: string;
  repository?: string;
  packages?: Predicate[];
  versions?: Predicate[];
}

export interface Metadata {
  region?: string;
  repository?: Repository;
}

/**
 * Renderer for the "aws.codeArtifact.onPackageVersion" trigger
 */
export const onPackageVersionTriggerRenderer: TriggerRenderer = {
  getTitleAndSubtitle: (context: TriggerEventContext): { title: string; subtitle: string } => {
    const eventData = context.event?.data as PackageVersionEvent;
    const detail = eventData?.detail;
    const packageLabel = formatPackageLabel(detail);
    const title = packageLabel || "CodeArtifact package version";
    const subtitle = formatTimeAgo(new Date(context.event?.createdAt || ""));

    return { title, subtitle };
  },

  getRootEventValues: (context: TriggerEventContext): Record<string, string> => {
    const eventData = context.event?.data as PackageVersionEvent;
    const detail = eventData?.detail as PackageVersionDetail;

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
    const configuration = node.configuration as Configuration | undefined;
    const metadataItems = buildMetadataItems(configuration);

    const props: TriggerProps = {
      title: node.name || definition.label || "Unnamed trigger",
      iconSrc: awsCodeArtifactIcon,
      collapsedBackground: getBackgroundColorClass(definition.color),
      metadata: metadataItems,
    };

    if (lastEvent) {
      const { title, subtitle } = onPackageVersionTriggerRenderer.getTitleAndSubtitle({ event: lastEvent });
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

function buildMetadataItems(configuration?: Configuration): MetadataItem[] {
  const items: MetadataItem[] = [];

  if (configuration?.region) {
    items.push({
      icon: "globe",
      label: configuration.region,
    });
  }

  if (configuration?.domain) {
    items.push({
      icon: "database",
      label: `Domain: ${configuration?.domain}`,
    });
  }

  if (configuration?.repository) {
    items.push({
      icon: "boxes",
      label: `Repository: ${configuration?.repository}`,
    });
  }

  if (configuration?.packages) {
    items.push({
      icon: "package",
      label: `${configuration.packages.map((predicate) => formatPredicate(predicate)).join(", ")}`,
    });
  }

  if (configuration?.versions) {
    items.push({
      icon: "tag",
      label: `${configuration.versions.map((predicate) => formatPredicate(predicate)).join(", ")}`,
    });
  }

  return items;
}
