import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  OutputPayload,
} from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import type { ListDataSourcesOutput } from "./types";
import { formatTimestamp } from "../utils";
import { grafanaComponentBaseProps, grafanaCreatedAtSubtitle } from "./base";

export const listDataSourcesMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext) {
    return grafanaComponentBaseProps(context, metadataList(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {
      "Listed At": formatTimestamp(context.execution.createdAt),
    };

    if (!outputs || !outputs.default || outputs.default.length === 0) {
      details["Data Source Count"] = "0";
      return details;
    }

    const payload = outputs.default[0];
    const payloadTimestamp = formatTimestamp(payload?.timestamp);
    if (payloadTimestamp !== "-") {
      details["Listed At"] = payloadTimestamp;
    }

    const output = payload?.data as ListDataSourcesOutput | undefined;
    const sources = output?.dataSources ?? [];

    details["Data Source Count"] = String(sources.length);

    const types = [...new Set(sources.map((s) => s.type).filter(Boolean))];
    if (types.length > 0) {
      details["Types"] = types.slice(0, 5).join(", ");
    }

    return details;
  },

  subtitle: grafanaCreatedAtSubtitle,
};

function metadataList(_node: NodeInfo): MetadataItem[] {
  return [];
}
