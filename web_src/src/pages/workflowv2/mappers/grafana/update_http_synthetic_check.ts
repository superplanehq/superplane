import type { ComponentBaseContext, ComponentBaseMapper, ExecutionDetailsContext, OutputPayload } from "../types";
import { grafanaComponentBaseProps, grafanaCreatedAtSubtitle } from "./base";
import type { UpdateHttpSyntheticCheckConfiguration } from "./types";
import { buildMutationDetails, buildSyntheticCheckMutationMetadata } from "./synthetic_check_shared";

export const updateHttpSyntheticCheckMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext) {
    return grafanaComponentBaseProps(context, buildSyntheticCheckMutationMetadata(context.node, "update"));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const configuration = context.node.configuration as UpdateHttpSyntheticCheckConfiguration | undefined;
    return buildMutationDetails("Updated", outputs?.default?.[0], configuration);
  },

  subtitle: grafanaCreatedAtSubtitle,
};
