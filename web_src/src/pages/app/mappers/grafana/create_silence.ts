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
    const configuration = context.node.configuration as CreateSilenceConfiguration | undefined;
    const details = buildCreateSilenceDetails(context.execution.createdAt, configuration);
    const payload = outputs?.default?.[0];
    if (!payload) {
      return details;
    }

    applyCreateSilenceTimestamp(details, payload.timestamp);
    const output = payload?.data as CreateSilenceOutput | undefined;
    applyCreateSilenceOutput(details, output, configuration);
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

function buildCreateSilenceDetails(
  executionCreatedAt: string | undefined,
  configuration: CreateSilenceConfiguration | undefined,
): Record<string, string> {
  const details: Record<string, string> = {
    "Created At": formatTimestamp(executionCreatedAt),
  };

  const matchersPreview = formatMatchersPreview(configuration?.matchers, { maxItems: 10 });
  if (matchersPreview) {
    details.Matchers = matchersPreview;
  }

  if (configuration?.startsAt) {
    details["Starts At"] = configuration.startsAt;
  }

  if (configuration?.endsAt) {
    details["Ends At"] = configuration.endsAt;
  }

  if (configuration?.comment) {
    details.Comment = configuration.comment;
  }

  return details;
}

function applyCreateSilenceTimestamp(details: Record<string, string>, payloadTimestamp: string | undefined): void {
  const formattedTimestamp = formatTimestamp(payloadTimestamp);
  if (formattedTimestamp !== "-") {
    details["Created At"] = formattedTimestamp;
  }
}

function applyCreateSilenceOutput(
  details: Record<string, string>,
  output: CreateSilenceOutput | undefined,
  configuration: CreateSilenceConfiguration | undefined,
): void {
  assignFormattedOrConfiguredValue(details, "Starts At", output?.startsAt, configuration?.startsAt);
  assignFormattedOrConfiguredValue(details, "Ends At", output?.endsAt, configuration?.endsAt);

  if (output?.silenceUrl) {
    details["Silence URL"] = output.silenceUrl;
  }
}

function assignFormattedOrConfiguredValue(
  details: Record<string, string>,
  key: string,
  outputValue: string | undefined,
  configuredValue: string | undefined,
): void {
  if (outputValue) {
    details[key] = formatTimestamp(outputValue);
    return;
  }

  if (configuredValue) {
    details[key] = configuredValue;
  }
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
