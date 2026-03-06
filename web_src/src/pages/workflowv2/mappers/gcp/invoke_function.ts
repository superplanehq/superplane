import { ComponentBaseProps } from "@/ui/componentBase";
import { formatTimeAgo } from "@/utils/date";
import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { baseMapper } from "./base";

export const invokeFunctionMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return baseMapper.props(context);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const payload = outputs?.default?.[0];
    const data = payload?.data as Record<string, any> | undefined;

    const details: Record<string, string> = {};

    if (payload?.timestamp) {
      details["Invoked At"] = new Date(payload.timestamp).toLocaleString();
    }

    if (data?.functionName) {
      const parts = String(data.functionName).split("/");
      details["Function"] = parts[parts.length - 1] ?? data.functionName;
    }

    if (data?.executionId) {
      details["Execution ID"] = String(data.executionId);
    }

    if (data?.resultRaw !== undefined) {
      details["Result"] = String(data.resultRaw);
    } else if (data?.result !== undefined) {
      details["Result"] = typeof data.result === "string" ? data.result : JSON.stringify(data.result);
    }

    return details;
  },

  subtitle(context: SubtitleContext): string {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? formatTimeAgo(new Date(timestamp)) : "";
  },
};
