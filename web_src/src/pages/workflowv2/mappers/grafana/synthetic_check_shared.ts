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

/** Resolves grouped (request/schedule/validation) or legacy flat configuration for UI mappers. */
export function getGrafanaSyntheticCheckFlatView(c: CreateHttpSyntheticCheckConfiguration | undefined): {
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
} {
  if (!c) {
    return {};
  }
  const req = c.request;
  const sch = c.schedule;
  const val = c.validation;
  return {
    target: req?.target ?? c.target,
    method: req?.method ?? c.method,
    headers: req?.headers ?? c.headers,
    body: req?.body ?? c.body,
    noFollowRedirects: req?.noFollowRedirects ?? c.noFollowRedirects,
    basicAuth: req?.basicAuth ?? c.basicAuth,
    bearerToken: req?.bearerToken ?? c.bearerToken,
    enabled: sch?.enabled ?? c.enabled,
    frequency: sch?.frequency ?? c.frequency,
    timeout: sch?.timeout ?? c.timeout,
    probes: sch?.probes ?? c.probes,
    failIfSSL: val?.failIfSSL ?? c.failIfSSL,
    failIfNotSSL: val?.failIfNotSSL ?? c.failIfNotSSL,
    validStatusCodes: val?.validStatusCodes ?? c.validStatusCodes,
    failIfBodyMatchesRegexp: val?.failIfBodyMatchesRegexp ?? c.failIfBodyMatchesRegexp,
    failIfBodyNotMatchesRegexp: val?.failIfBodyNotMatchesRegexp ?? c.failIfBodyNotMatchesRegexp,
    failIfHeaderMatchesRegexp: val?.failIfHeaderMatchesRegexp ?? c.failIfHeaderMatchesRegexp,
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

  if (variant === "update") {
    const idFallback = updateConfiguration?.syntheticCheck?.trim();
    const headline = nodeMetadata?.checkLabel?.trim() || (idFallback ? idFallback : undefined);
    if (headline) {
      metadata.push({ icon: "activity", label: truncate(headline, 48) });
    } else if (flat.target) {
      metadata.push({ icon: "globe", label: truncate(flat.target, 48) });
    }
  } else if (flat.target) {
    metadata.push({ icon: "globe", label: truncate(flat.target, 48) });
  }

  if (flat.method) {
    metadata.push({ icon: "arrow-right", label: flat.method.toUpperCase() });
  }

  const probeText =
    flat.probes && flat.probes.length > 0
      ? nodeMetadata?.probeSummary?.trim()
        ? truncate(nodeMetadata.probeSummary, 48)
        : probeSummary(flat.probes)
      : "";

  const scheduleParts: string[] = [];
  if (probeText) {
    scheduleParts.push(probeText);
  }
  if (flat.frequency) {
    scheduleParts.push(`Every ${formatConfiguredFrequency(flat.frequency)}`);
  }
  if (scheduleParts.length > 0) {
    metadata.push({ icon: "map-pin", label: scheduleParts.join(" · ") });
  }

  return metadata.slice(0, 3);
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
  const details: Record<string, string> = {};

  if (!configuration) {
    if (payload?.timestamp) {
      details["Fetched At"] = formatTimestamp(payload.timestamp);
    }
    return details;
  }

  const metrics = output?.metrics;
  const http = configuration.settings?.http;

  if (metrics?.lastOutcome) {
    details["Last Outcome"] = metrics.lastOutcome;
  }

  if (configuration.job) {
    details.Job = configuration.job;
  }

  if (configuration.target) {
    const method = (http?.method || "GET").toUpperCase();
    details.Target = `${method} ${configuration.target}`;
  }

  const scheduleLine = buildGetCheckScheduleLine(configuration.frequency, configuration.timeout);
  if (scheduleLine) {
    details.Schedule = scheduleLine;
  }

  if (metrics && syntheticCheckHasRunCounts(metrics)) {
    details["Runs (24h)"] = formatSyntheticCheckRuns24hLine(metrics);
  }

  if (metrics?.averageLatencySeconds24h != null) {
    details["Avg Latency (24h)"] = `${metrics.averageLatencySeconds24h.toFixed(3)}s`;
  } else if (metrics?.lastExecutionAt) {
    details["Last Probe"] = formatTimestamp(metrics.lastExecutionAt);
  } else if (payload?.timestamp) {
    details["Fetched At"] = formatTimestamp(payload.timestamp);
  }

  return details;
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
    parts.push(`${metrics.successRuns24h} succeeded`);
  }
  if (metrics.failureRuns24h != null) {
    parts.push(`${metrics.failureRuns24h} failed`);
  }
  if (metrics.totalRuns24h != null) {
    parts.push(`${metrics.totalRuns24h} total`);
  }
  return parts.join(" · ");
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
  if (value >= 1000 && value % 1000 === 0) {
    return formatMilliseconds(value);
  }

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
