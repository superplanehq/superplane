import type { MetadataItem } from "@/ui/metadataList";
import { truncate } from "../safeMappers";
import type {
  CreateHttpSyntheticCheckConfiguration,
  GetHttpSyntheticCheckOutput,
  SyntheticCheckMutationOutput,
  SyntheticCheckNodeMetadata,
  UpdateHttpSyntheticCheckConfiguration,
} from "./types";
import type { NodeInfo, OutputPayload } from "../types";
import { formatTimestamp } from "../utils";

type SyntheticCheckFlatView = {
  target?: string;
  method?: string;
  headers?: CreateHttpSyntheticCheckConfiguration["headers"];
  body?: string;
  noFollowRedirects?: boolean;
  basicAuth?: CreateHttpSyntheticCheckConfiguration["basicAuth"];
  bearerToken?: string;
  enabled?: boolean;
  frequency?: number;
  timeout?: number;
  probes?: string[];
  failIfSSL?: boolean;
  failIfNotSSL?: boolean;
  validStatusCodes?: number[];
  failIfBodyMatchesRegexp?: string[];
  failIfBodyNotMatchesRegexp?: string[];
  failIfHeaderMatchesRegexp?: CreateHttpSyntheticCheckConfiguration["failIfHeaderMatchesRegexp"];
};

/** Resolves grouped (request/schedule/validation) or legacy flat configuration for UI mappers. */
export function getGrafanaSyntheticCheckFlatView(
  c: CreateHttpSyntheticCheckConfiguration | undefined,
): SyntheticCheckFlatView {
  if (!c) {
    return {};
  }

  return {
    ...getSyntheticCheckRequestFlatView(c),
    ...getSyntheticCheckScheduleFlatView(c),
    ...getSyntheticCheckValidationFlatView(c),
  };
}

function getSyntheticCheckRequestFlatView(c: CreateHttpSyntheticCheckConfiguration): SyntheticCheckFlatView {
  const request = c.request;
  return {
    target: request?.target ?? c.target,
    method: request?.method ?? c.method,
    headers: request?.headers ?? c.headers,
    body: request?.body ?? c.body,
    noFollowRedirects: request?.noFollowRedirects ?? c.noFollowRedirects,
    basicAuth: request?.basicAuth ?? c.basicAuth,
    bearerToken: request?.bearerToken ?? c.bearerToken,
  };
}

function getSyntheticCheckScheduleFlatView(c: CreateHttpSyntheticCheckConfiguration): SyntheticCheckFlatView {
  const schedule = c.schedule;
  return {
    enabled: schedule?.enabled ?? c.enabled,
    frequency: normalizeConfiguredScheduleFrequency(schedule?.frequency, c.frequency),
    timeout: schedule?.timeout ?? c.timeout,
    probes: schedule?.probes ?? c.probes,
  };
}

function normalizeConfiguredScheduleFrequency(
  scheduleFrequency: number | undefined,
  legacyFrequency: number | undefined,
): number | undefined {
  if (scheduleFrequency != null) {
    return scheduleFrequency;
  }
  if (legacyFrequency == null) {
    return undefined;
  }
  return legacyFrequency / 1000;
}

function getSyntheticCheckValidationFlatView(c: CreateHttpSyntheticCheckConfiguration): SyntheticCheckFlatView {
  const validation = c.validation;
  return {
    failIfSSL: validation?.failIfSSL ?? c.failIfSSL,
    failIfNotSSL: validation?.failIfNotSSL ?? c.failIfNotSSL,
    validStatusCodes: validation?.validStatusCodes ?? c.validStatusCodes,
    failIfBodyMatchesRegexp: validation?.failIfBodyMatchesRegexp ?? c.failIfBodyMatchesRegexp,
    failIfBodyNotMatchesRegexp: validation?.failIfBodyNotMatchesRegexp ?? c.failIfBodyNotMatchesRegexp,
    failIfHeaderMatchesRegexp: validation?.failIfHeaderMatchesRegexp ?? c.failIfHeaderMatchesRegexp,
  };
}

export type SyntheticCheckCanvasVariant = "create" | "update";

/**
 * Canvas card metadata: at most three rows (target or check, method, locations + schedule).
 * Update variant uses the resolved check label as the first row and does not repeat the URL from the Request group.
 */
export function buildSyntheticCheckMutationMetadata(
  node: NodeInfo,
  variant: SyntheticCheckCanvasVariant = "create",
): MetadataItem[] {
  const configuration = node.configuration as CreateHttpSyntheticCheckConfiguration | undefined;
  const updateConfiguration = configuration as UpdateHttpSyntheticCheckConfiguration | undefined;
  const nodeMetadata = node.metadata as SyntheticCheckNodeMetadata | undefined;
  const flat = getGrafanaSyntheticCheckFlatView(configuration);
  const metadata: MetadataItem[] = [];
  const headline = buildSyntheticCheckHeadlineMetadata(flat, nodeMetadata, updateConfiguration, variant);

  if (headline) {
    metadata.push(headline);
  }

  if (flat.method) {
    metadata.push({ icon: "arrow-right", label: flat.method.toUpperCase() });
  }

  const schedule = buildSyntheticCheckScheduleMetadata(flat, nodeMetadata);
  if (schedule) {
    metadata.push(schedule);
  }

  return metadata.slice(0, 3);
}

function buildSyntheticCheckHeadlineMetadata(
  flat: SyntheticCheckFlatView,
  nodeMetadata: SyntheticCheckNodeMetadata | undefined,
  updateConfiguration: UpdateHttpSyntheticCheckConfiguration | undefined,
  variant: SyntheticCheckCanvasVariant,
): MetadataItem | undefined {
  if (variant === "update") {
    const idFallback = updateConfiguration?.syntheticCheck?.trim();
    const headline = nodeMetadata?.checkLabel?.trim() || idFallback;
    if (headline) {
      return { icon: "activity", label: truncate(headline, 48) };
    }
  }

  if (!flat.target) {
    return undefined;
  }

  return { icon: "globe", label: truncate(flat.target, 48) };
}

function buildSyntheticCheckScheduleMetadata(
  flat: SyntheticCheckFlatView,
  nodeMetadata: SyntheticCheckNodeMetadata | undefined,
): MetadataItem | undefined {
  const probeText = formatSyntheticCheckProbeText(flat.probes, nodeMetadata);
  const scheduleParts: string[] = [];

  if (probeText) {
    scheduleParts.push(probeText);
  }
  if (flat.frequency) {
    scheduleParts.push(`Every ${formatConfiguredFrequency(flat.frequency)}`);
  }
  if (scheduleParts.length === 0) {
    return undefined;
  }

  return { icon: "map-pin", label: scheduleParts.join(" · ") };
}

function formatSyntheticCheckProbeText(
  probes: string[] | undefined,
  nodeMetadata: SyntheticCheckNodeMetadata | undefined,
): string {
  if (!probes || probes.length === 0) {
    return "";
  }

  const probeMetadata = nodeMetadata?.probeSummary?.trim();
  if (probeMetadata) {
    return truncate(probeMetadata, 48);
  }

  return probeSummary(probes);
}

export function buildSyntheticCheckSelectionMetadata(
  nodeMetadata: SyntheticCheckNodeMetadata | undefined,
  syntheticCheck: string | undefined,
): MetadataItem[] {
  const label = nodeMetadata?.checkLabel || syntheticCheck;
  if (!label) {
    return [];
  }

  return [{ icon: "activity", label: truncate(label, 48) }];
}

export function buildMutationDetails(
  verb: "Created" | "Updated",
  payload: OutputPayload | undefined,
  fallbackConfiguration: CreateHttpSyntheticCheckConfiguration | undefined,
): Record<string, string> {
  const context = buildMutationContext(payload, fallbackConfiguration);
  const details: Record<string, string> = {
    [`${verb} At`]: formatTimestamp(payload?.timestamp),
  };

  addCheckReference(details, context.checkUrl, context.checkID);
  addTargetDetails(details, context.method, context.target, context.fallback);
  addScheduleDetails(details, context.frequency, context.timeout, context.fallback);
  addProbeSummary(details, context.probes);
  addEnabledDetail(details, context.enabled, context.fallback.enabled);

  return details;
}

export function buildGetSyntheticCheckDetails(payload: OutputPayload | undefined): Record<string, string> {
  const output = payload?.data as GetHttpSyntheticCheckOutput | undefined;
  const configuration = output?.configuration;

  if (!configuration) {
    return buildEmptyGetSyntheticCheckDetails(payload);
  }

  const details: Record<string, string> = {};
  const metrics = output?.metrics;

  if (payload?.timestamp) {
    details["Fetched At"] = formatTimestamp(payload.timestamp);
  }

  addGetSyntheticCheckSummaryDetails(details, configuration, metrics);
  addGetSyntheticCheckScheduleDetails(details, configuration);
  addGetSyntheticCheckMetricDetails(details, metrics);

  return details;
}

function buildEmptyGetSyntheticCheckDetails(payload: OutputPayload | undefined): Record<string, string> {
  if (!payload?.timestamp) {
    return {};
  }

  return { "Fetched At": formatTimestamp(payload.timestamp) };
}

function addGetSyntheticCheckSummaryDetails(
  details: Record<string, string>,
  configuration: NonNullable<GetHttpSyntheticCheckOutput["configuration"]>,
  metrics: GetHttpSyntheticCheckOutput["metrics"] | undefined,
): void {
  if (metrics?.lastOutcome) {
    details["Last Outcome"] = metrics.lastOutcome;
  }

  if (configuration.job) {
    details.Job = configuration.job;
  }

  if (!configuration.target) {
    return;
  }

  const method = (configuration.settings?.http?.method || "GET").toUpperCase();
  details.Target = `${method} ${configuration.target}`;
}

function addGetSyntheticCheckScheduleDetails(
  details: Record<string, string>,
  configuration: NonNullable<GetHttpSyntheticCheckOutput["configuration"]>,
): void {
  const scheduleLine = buildGetCheckScheduleLine(configuration.frequency, configuration.timeout);
  if (scheduleLine) {
    details.Schedule = scheduleLine;
  }
}

function addGetSyntheticCheckMetricDetails(
  details: Record<string, string>,
  metrics: GetHttpSyntheticCheckOutput["metrics"] | undefined,
): void {
  const healthSummary = formatSyntheticCheckHealthSummary(metrics);
  if (healthSummary) {
    details["Health (24h)"] = healthSummary;
  }

  if (metrics && syntheticCheckHasRunCounts(metrics)) {
    details["Runs (24h)"] = formatSyntheticCheckRuns24hLine(metrics);
  }

  if (metrics?.sslEarliestExpiryAt) {
    details["SSL Expiry"] = formatSyntheticCheckSSLExpiry(metrics);
  }

  const probeActivity = formatSyntheticCheckProbeActivity(metrics);
  if (probeActivity) {
    details[probeActivity.label] = probeActivity.value;
  }
}

function formatSyntheticCheckProbeActivity(
  metrics: GetHttpSyntheticCheckOutput["metrics"] | undefined,
): { label: string; value: string } | undefined {
  if (metrics?.averageLatencySeconds24h != null) {
    return { label: "Avg Latency (24h)", value: `${metrics.averageLatencySeconds24h.toFixed(3)}s` };
  }

  if (metrics?.lastExecutionAt) {
    return { label: "Last Probe", value: formatTimestamp(metrics.lastExecutionAt) };
  }
  return undefined;
}

function formatSyntheticCheckHealthSummary(
  metrics: GetHttpSyntheticCheckOutput["metrics"] | undefined,
): string | undefined {
  const parts: string[] = [];

  if (metrics?.uptimePercent24h != null) {
    parts.push(`${formatPercent(metrics.uptimePercent24h)} uptime`);
  }

  if (metrics?.reachabilityPercent24h != null) {
    parts.push(`${formatPercent(metrics.reachabilityPercent24h)} reachability`);
  }

  return parts.length > 0 ? parts.join(" · ") : undefined;
}

function formatSyntheticCheckSSLExpiry(metrics: NonNullable<GetHttpSyntheticCheckOutput["metrics"]>): string {
  const formattedDate = formatTimestamp(metrics.sslEarliestExpiryAt);
  if (metrics.sslEarliestExpiryDays == null) {
    return formattedDate;
  }

  return `${formattedDate} (${formatDays(metrics.sslEarliestExpiryDays)})`;
}

function formatPercent(value: number): string {
  return `${value.toFixed(value % 1 === 0 ? 0 : 2)}%`;
}

function formatDays(value: number): string {
  const rounded = Math.round(value * 10) / 10;
  return `${rounded.toFixed(rounded % 1 === 0 ? 0 : 1)}d`;
}

export function buildDeleteHttpSyntheticCheckDetails(payload: OutputPayload | undefined): Record<string, string> {
  const output = payload?.data as
    | { syntheticCheck?: string; job?: string; target?: string; deleted?: boolean }
    | undefined;
  const details: Record<string, string> = {
    "Deleted At": formatTimestamp(payload?.timestamp),
  };

  if (output?.syntheticCheck) {
    details["Check ID"] = output.syntheticCheck;
  }
  if (output?.job) {
    details.Job = output.job;
  }
  if (output?.target) {
    details.Target = output.target;
  }
  if (output?.deleted) {
    details.Status = "Deleted";
  }

  return details;
}

export function formatMilliseconds(value: number): string {
  if (!value) {
    return "0ms";
  }
  if (value % 60000 === 0) {
    return `${value / 60000}m`;
  }
  if (value % 1000 === 0) {
    return `${value / 1000}s`;
  }
  return `${value}ms`;
}

function probeSummary(probes: string[]): string {
  if (probes.length === 1) {
    return probes[0];
  }
  if (probes.length <= 3) {
    return probes.join(", ");
  }
  return `${probes.slice(0, 3).join(", ")} +${probes.length - 3}`;
}

function addCheckReference(
  details: Record<string, string>,
  checkUrl: string | undefined,
  checkID: number | undefined,
): void {
  if (checkUrl) {
    details.Check = checkUrl;
    return;
  }

  if (checkID != null) {
    details["Check ID"] = String(checkID);
  }
}

function addTargetDetails(
  details: Record<string, string>,
  method: string | undefined,
  target: string | undefined,
  fallbackConfiguration?: MutationFallback,
): void {
  const resolvedTarget = target || fallbackConfiguration?.target;
  if (!resolvedTarget) {
    return;
  }

  details.Target = `${(method || fallbackConfiguration?.method || "GET").toUpperCase()} ${resolvedTarget}`;
}

function addScheduleDetails(
  details: Record<string, string>,
  frequency: number | undefined,
  timeout: number | undefined,
  fallbackConfiguration?: MutationFallback,
): void {
  if (frequency) {
    details.Schedule = `Every ${formatMilliseconds(frequency)}`;
  } else if (fallbackConfiguration?.frequency) {
    details.Schedule = `Every ${formatConfiguredFrequency(fallbackConfiguration.frequency)}`;
  }

  const resolvedTimeout = timeout || fallbackConfiguration?.timeout;
  if (resolvedTimeout) {
    details.Timeout = formatMilliseconds(resolvedTimeout);
  }
}

function addProbeSummary(details: Record<string, string>, probes: string[] | undefined): void {
  if (!probes || probes.length === 0) {
    return;
  }

  details.Probes = probeSummary(probes);
}

function addEnabledDetail(
  details: Record<string, string>,
  enabled: boolean | undefined,
  fallbackEnabled?: boolean,
): void {
  const resolvedEnabled = enabled ?? fallbackEnabled;
  if (resolvedEnabled == null) {
    return;
  }

  details.Enabled = resolvedEnabled ? "Yes" : "No";
}

function buildGetCheckScheduleLine(frequencyMs: number | undefined, timeoutMs: number | undefined): string | undefined {
  const parts: string[] = [];
  if (frequencyMs) {
    parts.push(`Every ${formatMilliseconds(frequencyMs)}`);
  }
  if (timeoutMs) {
    parts.push(`${formatMilliseconds(timeoutMs)} timeout`);
  }
  return parts.length > 0 ? parts.join(" · ") : undefined;
}

function syntheticCheckHasRunCounts(metrics: NonNullable<GetHttpSyntheticCheckOutput["metrics"]>): boolean {
  return metrics.totalRuns24h != null || metrics.successRuns24h != null || metrics.failureRuns24h != null;
}

function formatSyntheticCheckRuns24hLine(metrics: NonNullable<GetHttpSyntheticCheckOutput["metrics"]>): string {
  const parts: string[] = [];
  if (metrics.successRuns24h != null) {
    parts.push(`${formatRunCount(metrics.successRuns24h)} succeeded`);
  }
  if (metrics.failureRuns24h != null) {
    parts.push(`${formatRunCount(metrics.failureRuns24h)} failed`);
  }
  if (metrics.totalRuns24h != null) {
    parts.push(`${formatRunCount(metrics.totalRuns24h)} total`);
  }
  return parts.join(" · ");
}

function formatRunCount(value: number): string {
  return String(Math.max(0, Math.round(value)));
}

type MutationFallback = {
  enabled?: boolean;
  frequency?: number;
  method?: string;
  probes?: string[];
  target?: string;
  timeout?: number;
};

type MutationDetailContext = {
  checkID?: number;
  checkUrl?: string;
  enabled?: boolean;
  fallback: MutationFallback;
  frequency?: number;
  method?: string;
  probes?: string[];
  target?: string;
  timeout?: number;
};

function normalizeMutationFallback(configuration: CreateHttpSyntheticCheckConfiguration | undefined): MutationFallback {
  const flat = getGrafanaSyntheticCheckFlatView(configuration);
  return {
    enabled: flat.enabled,
    frequency: flat.frequency,
    method: flat.method,
    probes: flat.probes,
    target: flat.target,
    timeout: flat.timeout,
  };
}

function formatConfiguredFrequency(value: number): string {
  return formatMilliseconds(value * 1000);
}

function buildMutationContext(
  payload: OutputPayload | undefined,
  fallbackConfiguration: CreateHttpSyntheticCheckConfiguration | undefined,
): MutationDetailContext {
  const output = payload?.data as SyntheticCheckMutationOutput | undefined;
  const fallback = normalizeMutationFallback(fallbackConfiguration);
  const check = output?.check;

  return {
    checkID: check?.id,
    checkUrl: output?.checkUrl,
    enabled: check?.enabled,
    fallback,
    frequency: check?.frequency,
    method: check?.settings?.http?.method,
    probes: check?.probes?.map(String) || fallback.probes,
    target: check?.target,
    timeout: check?.timeout,
  };
}
