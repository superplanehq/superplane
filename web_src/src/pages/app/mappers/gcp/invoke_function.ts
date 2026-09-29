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

function invokeFunctionResultDisplay(data: InvokeFunctionData | undefined): string | undefined {
  if (data?.resultRaw !== undefined) {
    return String(data.resultRaw);
  }
  if (data?.result !== undefined) {
    return typeof data.result === "string" ? data.result : JSON.stringify(data.result);
  }
  return undefined;
}

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

    const resultDisplay = invokeFunctionResultDisplay(data);
    if (resultDisplay !== undefined) {
      details["Result"] = resultDisplay;
    }

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
  },
};
