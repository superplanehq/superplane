import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  OutputPayload,
} from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import type { DataSource, GetDataSourceNodeMetadata } from "./types";
import { formatTimestamp } from "../utils";
import { grafanaComponentBaseProps, grafanaCreatedAtSubtitle } from "./base";

export const getDataSourceMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext) {
    return grafanaComponentBaseProps(context, metadataList(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {
      "Fetched At": formatTimestamp(context.execution.createdAt),
    };

    if (!outputs || !outputs.default || outputs.default.length === 0) {
      return details;
    }

    const payload = outputs.default[0];
    const payloadTimestamp = formatTimestamp(payload?.timestamp);
    if (payloadTimestamp !== "-") {
      details["Fetched At"] = payloadTimestamp;
    }

    const source = payload?.data as DataSource | undefined;
    appendDataSourceDetails(details, source);

    return details;
  },

  subtitle: grafanaCreatedAtSubtitle,
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as GetDataSourceNodeMetadata | undefined;
  const configuration = node.configuration as { dataSource?: string } | undefined;

  if (nodeMetadata?.dataSourceName) {
    metadata.push({ icon: "database", label: nodeMetadata.dataSourceName });
  } else if (configuration?.dataSource) {
    metadata.push({ icon: "database", label: `Data Source: ${configuration.dataSource}` });
  }

  if (nodeMetadata?.dataSourceType) {
    metadata.push({ icon: "server", label: `Type: ${nodeMetadata.dataSourceType}` });
  }

  return metadata;
}

function appendDataSourceDetails(details: Record<string, string>, source: DataSource | undefined): void {
  if (!source) {
    return;
  }

  appendDetail(details, "Name", source.name);
  appendDetail(details, "Type", source.type);
  appendDetail(details, "UID", source.uid);
  appendDetail(details, "URL", source.url);

  if (source.isDefault !== undefined) {
    details["Default"] = source.isDefault ? "Yes" : "No";
  }
}

function appendDetail(details: Record<string, string>, label: string, value: string | undefined): void {
  if (!value) {
    return;
  }

  details[label] = value;
}
