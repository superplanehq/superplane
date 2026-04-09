import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  OutputPayload,
} from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import type { QueryDataSourceConfiguration } from "./types";
import { truncate } from "../safeMappers";
import { formatTimestamp } from "../utils";
import { grafanaComponentBaseProps, grafanaCreatedAtSubtitle } from "./base";

export const queryDataSourceMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext) {
    return grafanaComponentBaseProps(context, metadataList(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const configuration = context.node.configuration as QueryDataSourceConfiguration | undefined;
    const details: Record<string, string> = {
      "Queried At": formatTimestamp(context.execution.createdAt),
    };

    if (configuration?.dataSourceUid) {
      details["Data Source"] = configuration.dataSourceUid;
    }

    if (configuration?.query) {
      details.Query = configuration.query;
    }

    if (configuration?.format) {
      details.Format = configuration.format;
    }

    if (!outputs || !outputs.default || outputs.default.length === 0) {
      details.Response = "No data returned";
      return details;
    }

    const payload = outputs.default[0];
    const responseData = payload?.data as Record<string, unknown> | undefined;
    const payloadTimestamp = formatTimestamp(payload?.timestamp);
    if (payloadTimestamp !== "-") {
      details["Queried At"] = payloadTimestamp;
    }

    if (!responseData) {
      details.Response = "No data returned";
      return details;
    }

    return {
      ...details,
      ...buildQueryResultSummary(responseData),
    };
  },

  subtitle: grafanaCreatedAtSubtitle,
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as QueryDataSourceConfiguration;

  if (configuration?.dataSourceUid) {
    metadata.push({ icon: "database", label: `Data Source: ${configuration.dataSourceUid}` });
  }

  if (configuration?.query) {
    const preview = truncate(configuration.query, 50);
    metadata.push({ icon: "code", label: preview });
  }

  if (configuration?.format) {
    metadata.push({ icon: "funnel", label: `Format: ${configuration.format}` });
  }

  if (configuration?.timezone) {
    metadata.push({ icon: "schedule", label: `Timezone: ${configuration.timezone}` });
  }

  return metadata;
}

function buildQueryResultSummary(responseData: Record<string, unknown>): Record<string, string> {
  const details: Record<string, string> = {};
  const results = responseData.results;

  if (!results || typeof results !== "object" || Array.isArray(results)) {
    details["Response Keys"] = Object.keys(responseData).join(", ") || "-";
    return details;
  }

  const refIds = Object.keys(results);
  if (refIds.length === 0) {
    details.Results = "No results";
    return details;
  }

  details["Result Ref IDs"] = refIds.join(", ");
  details.Results = String(refIds.length);

  const fieldNames = new Set<string>();
  const summary = summarizeQueryResults(results, refIds, fieldNames);

  details["Frame Count"] = String(summary.frameCount);

  if (summary.rowCount > 0) {
    details["Row Count"] = String(summary.rowCount);
  }

  if (fieldNames.size > 0) {
    details.Fields = Array.from(fieldNames).slice(0, 5).join(", ");
  }

  return details;
}

function summarizeQueryResults(
  results: object,
  refIds: string[],
  fieldNames: Set<string>,
): { frameCount: number; rowCount: number } {
  let frameCount = 0;
  let rowCount = 0;

  for (const refId of refIds) {
    const result = asRecord((results as Record<string, unknown>)[refId]);
    if (!result) {
      continue;
    }

    const frames = Array.isArray(result.frames) ? result.frames : [];
    frameCount += frames.length;

    for (const frameValue of frames) {
      const frame = asRecord(frameValue);
      if (!frame) {
        continue;
      }

      rowCount += getFrameRowCount(frame);
      collectFieldNames(frame, fieldNames);
    }
  }

  return { frameCount, rowCount };
}

function collectFieldNames(frame: Record<string, unknown>, fieldNames: Set<string>): void {
  const schema = asRecord(frame.schema);
  const schemaFields = Array.isArray(schema?.fields) ? schema.fields : [];

  for (const fieldValue of schemaFields) {
    const field = asRecord(fieldValue);
    if (typeof field?.name === "string") {
      fieldNames.add(field.name);
    }
  }
}

function getFrameRowCount(frame: Record<string, unknown>): number {
  const data = asRecord(frame.data);
  const values = data?.values;
  if (!Array.isArray(values)) {
    return 0;
  }

  let maxLength = 0;
  for (const column of values) {
    if (Array.isArray(column) && column.length > maxLength) {
      maxLength = column.length;
    }
  }

  return maxLength;
}

function asRecord(value: unknown): Record<string, unknown> | undefined {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return undefined;
  }

  return value as Record<string, unknown>;
}
