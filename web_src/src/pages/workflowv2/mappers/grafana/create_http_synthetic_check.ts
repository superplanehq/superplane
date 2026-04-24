import type { ComponentBaseContext, ComponentBaseMapper, ExecutionDetailsContext, OutputPayload } from "../types";
import { grafanaComponentBaseProps, grafanaCreatedAtSubtitle } from "./base";
import type { CreateHttpSyntheticCheckConfiguration } from "./types";
import { buildMutationDetails, buildSyntheticCheckMutationMetadata } from "./synthetic_check_shared";

export const createHttpSyntheticCheckMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext) {
    return grafanaComponentBaseProps(context, buildSyntheticCheckMutationMetadata(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const configuration = context.node.configuration as CreateHttpSyntheticCheckConfiguration | undefined;
    return buildMutationDetails("Created", outputs?.default?.[0], configuration);
  },

  subtitle: grafanaCreatedAtSubtitle,
};
