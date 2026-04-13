import type { ComponentBaseContext, ComponentBaseMapper, ExecutionDetailsContext, OutputPayload } from "../types";
import { formatTimestamp } from "../utils";
import { grafanaComponentBaseProps, grafanaCreatedAtSubtitle } from "./base";
import { buildSilenceSelectionMetadata, type SilenceSelectionNodeMetadata } from "./silenceMetadata";

interface DeleteSilenceConfiguration {
  silence?: string;
  silenceId?: string;
}

export const deleteSilenceMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext) {
    const configuration = context.node.configuration as DeleteSilenceConfiguration | undefined;
    const nodeMetadata = context.node.metadata as SilenceSelectionNodeMetadata | undefined;
    const silence = configuration?.silence ?? configuration?.silenceId;
    return grafanaComponentBaseProps(context, buildSilenceSelectionMetadata(nodeMetadata, silence));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {
      "Deleted At": formatTimestamp(context.execution.createdAt),
    };

    if (!outputs || !outputs.default || outputs.default.length === 0) {
      return details;
    }

    const payload = outputs.default[0];
    const payloadTimestamp = formatTimestamp(payload?.timestamp);
    if (payloadTimestamp !== "-") {
      details["Deleted At"] = payloadTimestamp;
    }

    return details;
  },

  subtitle: grafanaCreatedAtSubtitle,
};
