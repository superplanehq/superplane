import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  OutputPayload,
} from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import type { QueryTracesConfiguration } from "./types";
import { formatTimestamp } from "../utils";
import { countGrafanaQueryResponseRows } from "./queryResponse";
import { grafanaComponentBaseProps, grafanaCreatedAtSubtitle } from "./base";

export const queryTracesMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext) {
    return grafanaComponentBaseProps(context, metadataList(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const configuration = context.node.configuration as QueryTracesConfiguration | undefined;
    const details: Record<string, string> = {
      "Queried At": formatTimestamp(context.execution.createdAt),
    };

    if (configuration?.dataSource) {
      details["Data Source"] = configuration.dataSource;
    }

    if (configuration?.query) {
      details["Query"] =
        configuration.query.length > 80 ? configuration.query.substring(0, 80) + "..." : configuration.query;
    }

    if (!outputs || !outputs.default || outputs.default.length === 0) {
      details["Traces"] = "0";
      return details;
    }

    const payload = outputs.default[0];
    const payloadTimestamp = formatTimestamp(payload?.timestamp);
    if (payloadTimestamp !== "-") {
      details["Queried At"] = payloadTimestamp;
    }

    const responseData = payload?.data as Record<string, unknown> | undefined;
    if (!responseData) {
      details["Traces"] = "No data returned";
      return details;
    }

    details["Traces"] = String(countGrafanaQueryResponseRows(responseData));

    return details;
  },

  subtitle: grafanaCreatedAtSubtitle,
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as QueryTracesConfiguration | undefined;

  if (configuration?.dataSource) {
    metadata.push({ icon: "database", label: `Data Source: ${configuration.dataSource}` });
  }

  if (configuration?.query) {
    const preview =
      configuration.query.length > 50 ? configuration.query.substring(0, 50) + "..." : configuration.query;
    metadata.push({ icon: "code", label: preview });
  }

  return metadata;
}
