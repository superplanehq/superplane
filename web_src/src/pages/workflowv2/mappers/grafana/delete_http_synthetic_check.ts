import type { ComponentBaseContext, ComponentBaseMapper, ExecutionDetailsContext, OutputPayload } from "../types";
import { grafanaComponentBaseProps, grafanaCreatedAtSubtitle } from "./base";
import type { DeleteHttpSyntheticCheckConfiguration, SyntheticCheckNodeMetadata } from "./types";
import { buildDeleteHttpSyntheticCheckDetails, buildSyntheticCheckSelectionMetadata } from "./synthetic_check_shared";

export const deleteHttpSyntheticCheckMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext) {
    const configuration = context.node.configuration as DeleteHttpSyntheticCheckConfiguration | undefined;
    const nodeMetadata = context.node.metadata as SyntheticCheckNodeMetadata | undefined;
    return grafanaComponentBaseProps(
      context,
      buildSyntheticCheckSelectionMetadata(nodeMetadata, configuration?.syntheticCheck),
    );
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    return buildDeleteHttpSyntheticCheckDetails(outputs?.default?.[0]);
  },

  subtitle: grafanaCreatedAtSubtitle,
};
