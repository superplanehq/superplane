import type { ComponentBaseProps } from "@/ui/componentBase";
import type React from "react";
import { renderTimeAgo } from "@/components/TimeAgo";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { baseMapper } from "./base";
import gcpCloudRunIcon from "@/assets/icons/integrations/gcp.cloudrun.svg";

type InvokeFunctionData = {
  functionName?: string;
  executionId?: string;
  resultRaw?: unknown;
  result?: unknown;
};

type InvokeFunctionOutputPayload = OutputPayload & {
  data?: InvokeFunctionData;
};

export const invokeFunctionMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return {
      ...baseMapper.props(context),
      iconSrc: gcpCloudRunIcon,
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: InvokeFunctionOutputPayload[] } | undefined;
    const payload = outputs?.default?.[0];
    const data = payload?.data;

    const details: Record<string, string> = {};

    if (payload?.timestamp) {
      details["Invoked At"] = new Date(payload.timestamp).toLocaleString();
    }

    if (data?.functionName) {
      const parts = data.functionName.split("/");
      details["Function"] = parts[parts.length - 1] ?? data.functionName;
    }

    if (data?.executionId) {
      details["Execution ID"] = data.executionId;
    }

    if (data?.resultRaw !== undefined) {
      details["Result"] = String(data.resultRaw);
    } else if (data?.result !== undefined) {
      details["Result"] = typeof data.result === "string" ? data.result : JSON.stringify(data.result);
    }

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
  },
};
