import type { ComponentBaseMapper, ExecutionDetailsContext, SubtitleContext } from "../../types";
import type React from "react";
import {
  buildPrometheusComponentProps,
  firstOutputData,
  type PrometheusQueryPayload,
  prometheusSubtitle,
  queryDetails,
  queryMetadataList,
} from "./common";

export const queryRangeMapper: ComponentBaseMapper = {
  props(context) {
    return buildPrometheusComponentProps(context, queryMetadataList(context.node, true));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const payload = firstOutputData<PrometheusQueryPayload>(context.execution.outputs);
    return queryDetails(context.execution, context.node, payload);
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    return prometheusSubtitle(context);
  },
};
