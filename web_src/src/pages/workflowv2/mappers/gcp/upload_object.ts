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

export const uploadObjectMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return baseMapper.props(context);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const payload = outputs?.default?.[0];
    const data = payload?.data as Record<string, string | undefined> | undefined;

    const details: Record<string, string> = {};

    if (payload?.timestamp) {
      details["Uploaded At"] = new Date(payload.timestamp).toLocaleString();
    }

    if (data?.name) {
      details["Object"] = data.name;
    }

    if (data?.bucket) {
      details["Bucket"] = data.bucket;
    }

    if (data?.size) {
      details["Size"] = `${data.size} bytes`;
    }

    if (data?.contentType) {
      details["Content Type"] = data.contentType;
    }

    if (data?.storageClass) {
      details["Storage Class"] = data.storageClass;
    }

    return details;
  },

  subtitle(context: SubtitleContext): string {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? formatTimeAgo(new Date(timestamp)) : "";
  },
};
