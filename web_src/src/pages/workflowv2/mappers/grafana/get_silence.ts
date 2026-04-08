import type { ComponentBaseContext, ComponentBaseMapper, ExecutionDetailsContext, OutputPayload } from "../types";
import type { Silence } from "./types";
import { formatTimestamp } from "../utils";
import { grafanaComponentBaseProps, grafanaCreatedAtSubtitle } from "./base";
import { buildSilenceSelectionMetadata, type SilenceSelectionNodeMetadata } from "./silenceMetadata";

interface GetSilenceConfiguration {
  silenceId?: string;
}

export const getSilenceMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext) {
    const configuration = context.node.configuration as GetSilenceConfiguration | undefined;
    const nodeMetadata = context.node.metadata as SilenceSelectionNodeMetadata | undefined;
    return grafanaComponentBaseProps(context, buildSilenceSelectionMetadata(nodeMetadata, configuration?.silenceId));
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
    const payloadTimestamp = formatTimestamp(payload.timestamp);
    if (payloadTimestamp !== "-") {
      details["Fetched At"] = payloadTimestamp;
    }

    const silence = payload.data as Silence | undefined;

    if (silence?.status?.state) {
      details.State = silence.status.state;
    }

    if (silence?.comment) {
      details.Comment = silence.comment;
    }

    if (silence?.endsAt) {
      details["Ends At"] = formatTimestamp(silence.endsAt);
    }

    if (silence?.url) {
      details["Silence URL"] = silence.url;
    }

    return details;
  },

  subtitle: grafanaCreatedAtSubtitle,
};
