import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  OutputPayload,
} from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import type { CreateSilenceConfiguration, CreateSilenceOutput, SilenceMatcher } from "./types";
import { formatTimestamp } from "../utils";
import { grafanaComponentBaseProps, grafanaCreatedAtSubtitle } from "./base";

export const createSilenceMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext) {
    return grafanaComponentBaseProps(context, metadataList(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {
      "Created At": formatTimestamp(context.execution.createdAt),
    };

    const configuration = context.node.configuration as CreateSilenceConfiguration | undefined;
    if (configuration) {
      const matchersPreview = formatMatchersPreview(configuration.matchers, { maxItems: 10 });
      if (matchersPreview) {
        details.Matchers = matchersPreview;
      }
      if (configuration.startsAt) {
        details["Starts At"] = configuration.startsAt;
      }
      if (configuration.endsAt) {
        details["Ends At"] = configuration.endsAt;
      }
      if (configuration.comment) {
        details.Comment = configuration.comment;
      }
    }

    if (!outputs || !outputs.default || outputs.default.length === 0) {
      return details;
    }

    const payload = outputs.default[0];
    const payloadTimestamp = formatTimestamp(payload?.timestamp);
    if (payloadTimestamp !== "-") {
      details["Created At"] = payloadTimestamp;
    }

    const output = payload?.data as CreateSilenceOutput | undefined;

    if (output?.silenceUrl) {
      details["Silence URL"] = output.silenceUrl;
    }

    return details;
  },

  subtitle: grafanaCreatedAtSubtitle,
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const configuration = node.configuration as CreateSilenceConfiguration | undefined;

  return [
    buildMatchersMetadata(configuration),
    ...buildSilenceTimeWindowMetadata(configuration),
    buildCommentMetadata(configuration),
  ].filter((item): item is MetadataItem => Boolean(item));
}

function buildMatchersMetadata(configuration: CreateSilenceConfiguration | undefined): MetadataItem | undefined {
  const matchersPreview = formatMatchersPreview(configuration?.matchers, { maxItems: 2 });
  if (!matchersPreview) {
    return undefined;
  }

  return { icon: "filter", label: `Matchers: ${matchersPreview}` };
}

function buildSilenceTimeWindowMetadata(configuration: CreateSilenceConfiguration | undefined): MetadataItem[] {
  if (!configuration?.startsAt && !configuration?.endsAt) {
    return [];
  }

  if (configuration?.startsAt && configuration?.endsAt) {
    return [{ icon: "schedule", label: `${configuration.startsAt} → ${configuration.endsAt}` }];
  }

  if (configuration?.startsAt) {
    return [{ icon: "schedule", label: `Starts: ${configuration.startsAt}` }];
  }

  return [{ icon: "schedule", label: `Ends: ${configuration?.endsAt}` }];
}

function buildCommentMetadata(configuration: CreateSilenceConfiguration | undefined): MetadataItem | undefined {
  if (!configuration?.comment) {
    return undefined;
  }

  const preview =
    configuration.comment.length > 60
      ? configuration.comment.substring(0, 60).trimEnd() + "..."
      : configuration.comment;

  return { icon: "sticky-note", label: preview };
}

function formatMatchersPreview(
  matchers: SilenceMatcher[] | undefined,
  options: { maxItems: number },
): string | undefined {
  if (!matchers || !Array.isArray(matchers) || matchers.length === 0) {
    return undefined;
  }

  const formatted = matchers
    .map((m) => formatMatcher(m))
    .filter((m): m is string => typeof m === "string" && m.length > 0);

  if (formatted.length === 0) {
    return undefined;
  }

  const maxItems = Math.max(1, options.maxItems);
  const head = formatted.slice(0, maxItems);
  const remaining = formatted.length - head.length;
  const suffix = remaining > 0 ? ` +${remaining}` : "";

  return head.join(", ") + suffix;
}

function formatMatcher(matcher: SilenceMatcher | undefined): string | undefined {
  if (!matcher || typeof matcher !== "object") {
    return undefined;
  }

  const name = typeof matcher.name === "string" ? matcher.name.trim() : "";
  const value = typeof matcher.value === "string" ? matcher.value.trim() : "";
  if (!name || !value) {
    return undefined;
  }

  const operator =
    typeof matcher.operator === "string" && matcher.operator.trim().length > 0
      ? matcher.operator.trim()
      : matcher.isRegex
        ? "=~"
        : "=";

  return `${name}${operator}${value}`;
}
