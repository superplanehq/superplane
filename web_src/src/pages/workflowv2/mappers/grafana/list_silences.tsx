import React from "react";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  OutputPayload,
  SubtitleContext,
} from "../types";
import type { ListSilencesOutput } from "./types";
import { renderTimeAgo } from "@/components/TimeAgo";
import { formatTimestamp } from "../utils";
import { grafanaComponentBaseProps } from "./base";

export const listSilencesMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext) {
    return grafanaComponentBaseProps(context, []);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {
      "Listed At": formatTimestamp(context.execution.createdAt),
    };

    if (!outputs || !outputs.default || outputs.default.length === 0) {
      details["Silence Count"] = "0";
      return details;
    }

    const payload = outputs.default[0];
    const payloadTimestamp = formatTimestamp(payload?.timestamp);
    if (payloadTimestamp !== "-") {
      details["Listed At"] = payloadTimestamp;
    }

    const output = payload?.data as ListSilencesOutput | undefined;
    const silences = output?.silences ?? [];
    const count = silences.length;

    details["Silence Count"] = String(count);

    if (count > 0 && silences[0].comment) {
      details["First Comment"] = silences[0].comment;
    }

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const payload = outputs?.default?.[0];
    const output = payload?.data as ListSilencesOutput | undefined;
    const count = output?.silences?.length ?? 0;

    const timeAgo = context.execution.createdAt ? renderTimeAgo(new Date(context.execution.createdAt)) : "-";

    return (
      <span>
        {count} silence{count === 1 ? "" : "s"} · {timeAgo}
      </span>
    );
  },
};
