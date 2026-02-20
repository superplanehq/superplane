import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../../types";
import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "../..";
import awsCodeArtifactIcon from "@/assets/icons/integrations/aws.codeartifact.svg";
import { formatTimeAgo } from "@/utils/date";
import { formatTimestampInUserTimezone } from "@/utils/timezone";
import { MetadataItem } from "@/ui/metadataList";
import { PackageVersionDescription, PackageLicense, PackageVersionPayload } from "./types";
import { formatPackageName } from "./utils";
import { stringOrDash } from "../../utils";

export interface GetPackageVersionConfiguration {
  region?: string;
  domain?: string;
  repository?: string;
  package?: string;
  format?: string;
  namespace?: string;
  version?: string;
}

export const getPackageVersionMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;

    return {
      title:
        context.node.name ||
        context.componentDefinition.label ||
        context.componentDefinition.name ||
        "Unnamed component",
      iconSrc: awsCodeArtifactIcon,
      iconColor: getColorClass(context.componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      eventSections: lastExecution
        ? getPackageVersionEventSections(context.nodes, lastExecution, context.componentDefinition.name)
        : undefined,
      includeEmptyState: !lastExecution,
      metadata: getPackageVersionMetadataList(context.node),
      eventStateMap: getStateMap(context.componentDefinition.name),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const payload = outputs?.default?.[0]?.data as PackageVersionPayload | undefined;
    const result = payload?.package as PackageVersionDescription | undefined;

    if (!result) {
      return {};
    }

    return {
      Package: stringOrDash(formatPackageName(result.namespace, result.packageName)),
      Version: stringOrDash(result.version),
      Format: stringOrDash(result.format),
      Status: stringOrDash(result.status),
      Revision: stringOrDash(result.revision),
      "Display Name": stringOrDash(result.displayName),
      "Published At": result.publishedTime ? formatTimestampInUserTimezone(result.publishedTime) : "-",
      Summary: stringOrDash(result.summary),
      Licenses: formatLicenses(result.licenses),
      "Home Page": stringOrDash(result.homePage),
      "Source Code": stringOrDash(result.sourceCodeRepository),
    };
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) {
      return "";
    }
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function getPackageVersionMetadataList(node: NodeInfo): MetadataItem[] {
  const configuration = node.configuration as GetPackageVersionConfiguration | undefined;
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

  if (configuration?.package) {
    items.push({
      icon: "package",
      label: configuration.package,
    });
  }

  if (configuration?.version) {
    items.push({
      icon: "tag",
      label: configuration.version,
    });
  }

  return items;
}

function getPackageVersionEventSections(
  nodes: NodeInfo[],
  execution: ExecutionInfo,
  componentName: string,
): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: formatTimeAgo(new Date(execution.createdAt!)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent!.id!,
    },
  ];
}

function formatLicenses(licenses?: PackageLicense[]): string {
  if (!licenses || licenses.length === 0) {
    return "-";
  }

  const formatted = licenses
    .map((license) => {
      if (!license) {
        return "";
      }

      const name = license.name?.trim();
      const url = license.url?.trim();

      if (name && url) {
        return `${name} (${url})`;
      }

      return name || url || "";
    })
    .filter((entry) => entry.length > 0)
    .join(", ");

  return formatted.length > 0 ? formatted : "-";
}
