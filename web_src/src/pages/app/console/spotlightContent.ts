import type { ComponentProps } from "react";
import * as yaml from "js-yaml";

import { compileFieldResolver } from "./widget/resolveCellValue";
import { getValueAtPath } from "./widget/fieldPath";
import type { SpotlightActor, SpotlightCheck, SpotlightStatus, WidgetSpotlight } from "./widget/WidgetSpotlight";
import type { WidgetDataSource } from "./widget/types";

const KNOWN_STATUSES: SpotlightStatus[] = ["success", "running", "failed", "warning", "neutral"];

const STATUS_SYNONYMS: Record<string, SpotlightStatus> = {
  success: "success",
  passed: "success",
  pass: "success",
  ok: "success",
  green: "success",
  completed: "success",
  succeeded: "success",
  healthy: "success",
  running: "running",
  in_progress: "running",
  "in-progress": "running",
  pending: "running",
  queued: "running",
  started: "running",
  deploying: "running",
  failed: "failed",
  failure: "failed",
  error: "failed",
  red: "failed",
  broken: "failed",
  warning: "warning",
  warn: "warning",
  degraded: "warning",
  flaky: "warning",
  // SuperPlane run/execution enum tokens (RESULT_* / STATE_*). Result is
  // authoritative for a finished stage; the STATE_* tokens only carry signal
  // while a stage is still in flight, so a bare STATE_FINISHED stays neutral
  // and lets the result decide.
  result_passed: "success",
  result_failed: "failed",
  result_none: "neutral",
  result_cancelled: "neutral",
  state_started: "running",
  state_pending: "running",
  state_queued: "running",
  state_finished: "neutral",
};

/** Coerce an arbitrary status string into one of the known spotlight statuses. */
export function normalizeStatus(raw: unknown): SpotlightStatus {
  if (typeof raw !== "string") return "neutral";
  const key = raw.trim().toLowerCase();
  if ((KNOWN_STATUSES as string[]).includes(key)) return key as SpotlightStatus;
  return STATUS_SYNONYMS[key] ?? "neutral";
}

/**
 * Editor-facing content model for the prototype spotlight panel. It mirrors how
 * real console panels are structured — a `dataSource` (memory / executions /
 * runs) plus render config — but instead of aggregating many rows it resolves a
 * set of display "slots" off a single record. A data-bound panel would run
 * `useWidgetData(dataSource)` and feed `rows[0]` into `spotlightPropsFromContent`.
 *
 * Every `*Field` is a literal dot path or `{{ cel }}` expression resolved with
 * the same `compileFieldResolver` the table/chart cells use, so authors get the
 * exact same field vocabulary they already know.
 */
export interface SpotlightPanelContent {
  /** Panel chrome title. */
  title?: string;
  /** Where the record comes from (same shape as table/chart/number panels). */
  dataSource: WidgetDataSource;
  /** Static eyebrow shown above the headline, e.g. "Currently in production". */
  kicker?: string;
  /** Field holding the overall status (normalized to success/running/failed/...). */
  statusField?: string;
  /** Field holding the header pill text, e.g. "Live". */
  statusLabelField?: string;
  /** Who — name + avatar. */
  actorNameField?: string;
  actorAvatarField?: string;
  /** What — headline title + optional link. */
  titleField?: string;
  hrefField?: string;
  subtitleField?: string;
  /** When + how long. */
  timestampField?: string;
  durationField?: string;
  /** A secondary person (approver / reviewer / commander / owner). */
  approverNameField?: string;
  approverAvatarField?: string;
  /** Static label shown before that person, e.g. "Approved by", "Reviewed by". */
  approverLabel?: string;
  /** Checks — a field resolving to an array, plus item sub-paths. */
  checksField?: string;
  checkNameField?: string;
  checkStatusField?: string;
}

export const DEFAULT_CHECK_NAME_FIELD = "name";
export const DEFAULT_CHECK_STATUS_FIELD = "status";

/**
 * Per-source slot defaults. Runs/executions map onto the derived row fields the
 * console already exposes (`status`, `nodeName`, `createdAt`, `durationMs`) and
 * read stages from the run's `executions` array; memory is left mostly blank
 * because its shape is discovered per namespace. `applySourceDefaults` swaps
 * these in when the author changes the data source kind, so a fresh source
 * renders something immediately instead of a blank banner.
 */
export type SlotDefaults = Pick<
  SpotlightPanelContent,
  | "statusField"
  | "statusLabelField"
  | "actorNameField"
  | "actorAvatarField"
  | "titleField"
  | "hrefField"
  | "subtitleField"
  | "timestampField"
  | "durationField"
  | "checksField"
  | "checkNameField"
  | "checkStatusField"
>;

const RUN_SLOT_DEFAULTS: SlotDefaults = {
  statusField: "status",
  statusLabelField: "status",
  actorNameField: "",
  actorAvatarField: "",
  titleField: "nodeName",
  hrefField: "",
  subtitleField: "",
  timestampField: "createdAt",
  durationField: "durationMs",
  checksField: "executions",
  checkNameField: "nodeName",
  checkStatusField: "result",
};

const EXECUTION_SLOT_DEFAULTS: SlotDefaults = {
  ...RUN_SLOT_DEFAULTS,
  // A single execution is one stage — there is no nested array to spotlight.
  checksField: "",
  checkNameField: DEFAULT_CHECK_NAME_FIELD,
  checkStatusField: DEFAULT_CHECK_STATUS_FIELD,
};

const MEMORY_SLOT_DEFAULTS: SlotDefaults = {
  statusField: "status",
  statusLabelField: "",
  actorNameField: "",
  actorAvatarField: "",
  titleField: "",
  hrefField: "",
  subtitleField: "",
  timestampField: "",
  durationField: "",
  checksField: "checks",
  checkNameField: DEFAULT_CHECK_NAME_FIELD,
  checkStatusField: DEFAULT_CHECK_STATUS_FIELD,
};

/** Slot defaults for a given data source kind. */
export function slotDefaultsForSource(kind: WidgetDataSource["kind"]): SlotDefaults {
  if (kind === "runs") return RUN_SLOT_DEFAULTS;
  if (kind === "executions") return EXECUTION_SLOT_DEFAULTS;
  return MEMORY_SLOT_DEFAULTS;
}

/** Whether the source exposes a per-record array of stages (runs) vs a checks payload (memory). */
export function checksAreStages(kind: WidgetDataSource["kind"]): boolean {
  return kind === "runs";
}

/**
 * Swap in the slot defaults for a newly-selected source while preserving the
 * author's kicker/title/approver-label choices, which are source-agnostic.
 */
export function applySourceDefaults(content: SpotlightPanelContent, next: WidgetDataSource): SpotlightPanelContent {
  return { ...content, ...slotDefaultsForSource(next.kind), dataSource: next };
}

/**
 * A valid, immediately-rendering default so the editor is never empty. Runs are
 * the primary source: the banner spotlights the latest run of a canvas, with
 * its pipeline stages as the checks strip.
 */
export const DEFAULT_SPOTLIGHT_CONTENT: SpotlightPanelContent = {
  title: "",
  dataSource: { kind: "runs" },
  kicker: "Latest run",
  approverNameField: "",
  approverAvatarField: "",
  approverLabel: "Approved by",
  ...RUN_SLOT_DEFAULTS,
};

type SpotlightRenderProps = ComponentProps<typeof WidgetSpotlight>;

function asString(value: unknown): string | undefined {
  if (value == null) return undefined;
  const text = String(value).trim();
  return text === "" ? undefined : text;
}

function asNumber(value: unknown): number | undefined {
  const n = typeof value === "number" ? value : Number(value);
  return Number.isFinite(n) ? n : undefined;
}

function resolveActor(row: unknown, nameField?: string, avatarField?: string): SpotlightActor | undefined {
  const name = nameField ? asString(compileFieldResolver(nameField).resolve(row)) : undefined;
  const avatarUrl = avatarField ? asString(compileFieldResolver(avatarField).resolve(row)) : undefined;
  if (!name && !avatarUrl) return undefined;
  return { name: name ?? "Unknown", avatarUrl };
}

/**
 * Resolve the checks/stages array off the record, mapping each item to a name +
 * normalized status via the configured item sub-paths.
 *
 * Works for both shapes: a memory `checks: [{ name, status }]` payload and a
 * run's `executions: [{ nodeId, state, result }]` array. The name falls back
 * from the configured path to `nodeId`/`name`, and the status falls back to the
 * item's `state` when the primary path (e.g. `result`) is inconclusive — so a
 * still-running stage (`RESULT_NONE` + `STATE_STARTED`) reads as running.
 */
function resolveChecks(row: unknown, content: SpotlightPanelContent): SpotlightCheck[] | undefined {
  if (!content.checksField?.trim()) return undefined;
  const raw = compileFieldResolver(content.checksField).resolve(row);
  if (!Array.isArray(raw)) return undefined;
  const nameField = content.checkNameField?.trim() || DEFAULT_CHECK_NAME_FIELD;
  const statusField = content.checkStatusField?.trim() || DEFAULT_CHECK_STATUS_FIELD;
  return raw.map((item) => ({
    name: resolveCheckName(item, nameField),
    status: resolveCheckStatus(item, statusField),
  }));
}

function resolveCheckName(item: unknown, nameField: string): string {
  return (
    asString(getValueAtPath(item, nameField)) ??
    asString(getValueAtPath(item, "nodeName")) ??
    asString(getValueAtPath(item, "nodeId")) ??
    "Stage"
  );
}

function resolveCheckStatus(item: unknown, statusField: string): SpotlightStatus {
  const status = normalizeStatus(getValueAtPath(item, statusField));
  if (status !== "neutral") return status;
  const state = getValueAtPath(item, "state");
  return state == null ? status : normalizeStatus(state);
}

/**
 * Map the editor content to `WidgetSpotlight` render props by resolving each
 * slot off a single sample record (`rows[0]` in a data-bound panel).
 */
export function spotlightPropsFromContent(content: SpotlightPanelContent, row: unknown): SpotlightRenderProps {
  const resolve = (field?: string): unknown => (field?.trim() ? compileFieldResolver(field).resolve(row) : undefined);
  return {
    kicker: content.kicker?.trim() || undefined,
    status: normalizeStatus(resolve(content.statusField)),
    statusLabel: asString(resolve(content.statusLabelField)),
    actor: resolveActor(row, content.actorNameField, content.actorAvatarField),
    title: asString(resolve(content.titleField)),
    href: asString(resolve(content.hrefField)),
    subtitle: asString(resolve(content.subtitleField)),
    timestamp: asString(resolve(content.timestampField)) ?? asNumber(resolve(content.timestampField)),
    duration: asNumber(resolve(content.durationField)),
    approver: resolveActor(row, content.approverNameField, content.approverAvatarField),
    approverLabel: content.approverLabel?.trim() || undefined,
    checks: resolveChecks(row, content),
  };
}

/**
 * Return a human-readable problem with the content, or `null` when valid.
 * Powers both the inline field hints and the footer summary strip.
 */
export function validateSpotlightContent(content: SpotlightPanelContent): string | null {
  const dataSource = content.dataSource;
  if (dataSource.kind === "memory" && !dataSource.namespace.trim()) {
    return "Choose a memory namespace for the data source.";
  }
  const hasHeadline = Boolean(content.titleField?.trim() || content.actorNameField?.trim());
  if (!hasHeadline) {
    return "Map at least a title or an actor name so the banner has a headline.";
  }
  return null;
}

/**
 * Serialize the content to YAML for the editor's YAML tab, in the same
 * `dataSource` + `render` shape the real panels persist.
 */
export function spotlightContentToYaml(content: SpotlightPanelContent): string {
  const render: Record<string, unknown> = {
    kicker: content.kicker || undefined,
    status: content.statusField || undefined,
    statusLabel: content.statusLabelField || undefined,
    actor: pruneEmpty({ name: content.actorNameField, avatar: content.actorAvatarField }),
    title: content.titleField || undefined,
    href: content.hrefField || undefined,
    subtitle: content.subtitleField || undefined,
    timestamp: content.timestampField || undefined,
    duration: content.durationField || undefined,
    approver: pruneEmpty({
      name: content.approverNameField,
      avatar: content.approverAvatarField,
      label: content.approverLabel,
    }),
    checks: content.checksField
      ? pruneEmpty({
          field: content.checksField,
          name: content.checkNameField,
          status: content.checkStatusField,
        })
      : undefined,
  };

  return yaml.dump(
    { type: "spotlight", dataSource: content.dataSource, render },
    {
      noRefs: true,
      lineWidth: 100,
      sortKeys: false,
    },
  );
}

/** Drop empty/undefined entries; return undefined when nothing remains. */
function pruneEmpty(input: Record<string, string | undefined>): Record<string, string> | undefined {
  const entries = Object.entries(input).filter(([, v]) => v && v.trim() !== "") as [string, string][];
  if (entries.length === 0) return undefined;
  return Object.fromEntries(entries);
}
