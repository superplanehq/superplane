import React from "react";

import { cn, resolveIcon } from "@/lib/utils";
import { formatDurationSeconds } from "@/lib/duration";
import { DEFAULT_EVENT_STATE_MAP, type EventState } from "@/ui/componentBase";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/ui/hoverCard";
import { TimeAgo } from "@/components/TimeAgo";
import {
  getStatusBadgeProps,
  resolveEventState,
} from "@/pages/workflowv2/lib/canvas-runs";
import { AlertTriangle } from "lucide-react";

//
// One step as seen by the ribbon. `status` is the raw label (can be a
// component-specific string like "created" or "pushed through"),
// `eventState` is the canonical state used for semantics (counts, running
// detection), and `badgeColor` is the exact tailwind bg class the canvas
// node uses for this step. We keep both because some components
// (e.g. wait) define custom colors that don't exist in the canonical
// EventState palette -- pushing the color through from the component's
// own EventStateMap is the only way to stay consistent with the canvas.
// When `badgeColor` is absent the ribbon falls back to the canonical map.
//
export interface RibbonStep {
  key: string;
  name: string;
  status: string;
  eventState?: EventState;
  badgeColor?: string;
  isTrigger: boolean;
  durationMs: number;
  finished: boolean;
  componentName?: string;
  iconSrc?: string;
  iconSlug?: string;
  startedAt?: string;
  finishedAt?: string;
  elapsedMs?: number;
  error?: string;
}

interface StepRibbonProps {
  steps: RibbonStep[];
  totalDurationMs: number;
  onStepClick?: (key: string) => void;
}

//
// Canonical state for the ribbon. Triggers always render as "triggered"
// (violet) regardless of the underlying execution status so they stay
// visually distinct. For everything else we prefer the explicit `eventState`
// the caller passes (which comes from the component's own state resolver,
// matching the canvas), falling back to the global status -> state map.
//
function stateFor(step: RibbonStep): EventState {
  if (step.isTrigger) return "triggered";
  return step.eventState ?? resolveEventState(step.status);
}

//
// Prefer the caller-provided `badgeColor` (resolved via the component's
// own EventStateMap) so per-component custom colors land correctly. Fall
// back to the canonical palette for cases where no explicit color is
// known (triggers, synthetic queued items, old call sites).
//
function barColorClass(step: RibbonStep): string {
  if (step.badgeColor) return step.badgeColor;
  const state = stateFor(step);
  return (DEFAULT_EVENT_STATE_MAP[state] || DEFAULT_EVENT_STATE_MAP.neutral).badgeColor;
}

function formatMs(ms: number): string {
  if (ms <= 0) return "0s";
  return formatDurationSeconds(ms);
}

function buildCaption(steps: RibbonStep[], totalDurationMs: number): string {
  const execSteps = steps.filter((s) => !s.isTrigger);
  if (execSteps.length === 0) return "No steps executed yet";

  //
  // Bucket steps by canonical event state so the caption counters match
  // the ribbon colors (and the rest of the UI). "running" and "queued"
  // both indicate in-flight work, so we collapse them into one counter.
  //
  const buckets = { running: 0, success: 0, failed: 0, error: 0, cancelled: 0 };
  for (const step of execSteps) {
    const state = stateFor(step);
    if (state === "running" || state === "queued") buckets.running += 1;
    else if (state === "success") buckets.success += 1;
    else if (state === "failed") buckets.failed += 1;
    else if (state === "error") buckets.error += 1;
    else if (state === "cancelled") buckets.cancelled += 1;
  }

  const durationPart = totalDurationMs > 0 ? formatDurationSeconds(totalDurationMs) : null;

  if (buckets.running > 0) {
    const base = `${buckets.running} of ${execSteps.length} running`;
    return durationPart ? `${base} · elapsed ${durationPart}` : base;
  }

  const parts: string[] = [];
  parts.push(`${execSteps.length} ${execSteps.length === 1 ? "step" : "steps"}`);
  if (buckets.success > 0) parts.push(`${buckets.success} passed`);
  const failedTotal = buckets.failed + buckets.error;
  if (failedTotal > 0) parts.push(`${failedTotal} failed`);
  if (buckets.cancelled > 0) parts.push(`${buckets.cancelled} cancelled`);
  const left = parts.join(", ");
  return durationPart ? `${left} · ${durationPart}` : left;
}

//
// Ribbon step icon. Mirrors the icon-resolution logic in ActivityRow so the
// hover card shows the same glyph the rest of Run View uses. Falls back to
// the lucide "bolt" icon (matches ActivityRow's fallback).
//
function StepIcon({
  iconSrc,
  iconSlug,
  alt,
}: {
  iconSrc: string | undefined;
  iconSlug: string | undefined;
  alt: string;
}) {
  if (iconSrc) {
    return <img src={iconSrc} alt={alt} className="h-4 w-4 shrink-0 object-contain" />;
  }
  return React.createElement(resolveIcon(iconSlug || "bolt"), {
    size: 16,
    className: "shrink-0 text-gray-500",
  });
}

function formatAbsolute(value: string): string {
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return value;
  return d.toLocaleString();
}

function StepHoverCardContent({ step }: { step: RibbonStep }) {
  const eventState = stateFor(step);
  const badge = getStatusBadgeProps(
    step.isTrigger ? "triggered" : step.status,
    eventState,
    step.badgeColor,
  );
  const isRunning = !step.isTrigger && !step.finished && eventState !== "queued";
  const finishedAt = step.finished ? step.finishedAt : undefined;
  const elapsedDisplay =
    !step.isTrigger && !step.finished && step.elapsedMs && step.elapsedMs > 0
      ? formatMs(step.elapsedMs)
      : null;
  const durationDisplay = step.finished && step.durationMs > 0 ? formatMs(step.durationMs) : null;
  const componentLabel = step.isTrigger ? "Trigger" : step.componentName;

  return (
    <div className="flex w-64 flex-col gap-2 text-sm">
      <div className="flex items-start gap-2">
        <StepIcon iconSrc={step.iconSrc} iconSlug={step.iconSlug} alt={step.name} />
        <div className="flex min-w-0 flex-1 flex-col leading-tight">
          <span className="truncate font-medium text-gray-900">{step.name}</span>
          {componentLabel ? (
            <span className="truncate text-[11px] text-gray-500">{componentLabel}</span>
          ) : null}
        </div>
        <span
          className={cn(
            "shrink-0 rounded px-1.5 py-[1px] text-[10px] font-semibold uppercase tracking-wide text-white",
            badge.badgeColor,
          )}
        >
          {badge.label}
        </span>
      </div>

      <div className="flex flex-col gap-1 border-t border-gray-100 pt-2 text-[11px] text-gray-600">
        {step.isTrigger && step.startedAt ? (
          <div className="flex items-center justify-between gap-2">
            <span className="text-gray-500">Triggered</span>
            <span className="flex min-w-0 items-center gap-1 text-gray-700">
              <TimeAgo date={step.startedAt} />
              <span className="shrink-0 text-gray-400">·</span>
              <span className="truncate">{formatAbsolute(step.startedAt)}</span>
            </span>
          </div>
        ) : null}

        {!step.isTrigger && step.startedAt ? (
          <div className="flex items-center justify-between gap-2">
            <span className="text-gray-500">Started</span>
            <span className="flex min-w-0 items-center gap-1 text-gray-700">
              <TimeAgo date={step.startedAt} />
              <span className="shrink-0 text-gray-400">·</span>
              <span className="truncate">{formatAbsolute(step.startedAt)}</span>
            </span>
          </div>
        ) : null}

        {!step.isTrigger && finishedAt ? (
          <div className="flex items-center justify-between gap-2">
            <span className="text-gray-500">Finished</span>
            <span className="flex min-w-0 items-center gap-1 text-gray-700">
              <TimeAgo date={finishedAt} />
              <span className="shrink-0 text-gray-400">·</span>
              <span className="truncate">{formatAbsolute(finishedAt)}</span>
            </span>
          </div>
        ) : null}

        {durationDisplay ? (
          <div className="flex items-center justify-between gap-2">
            <span className="text-gray-500">Duration</span>
            <span className="tabular-nums text-gray-700">{durationDisplay}</span>
          </div>
        ) : null}

        {isRunning && elapsedDisplay ? (
          <div className="flex items-center justify-between gap-2">
            <span className="text-gray-500">Elapsed</span>
            <span className="tabular-nums text-gray-700">{elapsedDisplay}</span>
          </div>
        ) : null}

      </div>

      {step.error ? (
        <div className="flex items-start gap-1.5 rounded border border-red-200 bg-red-50 px-2 py-1.5 text-[11px] text-red-700">
          <AlertTriangle className="mt-[1px] h-3 w-3 shrink-0" />
          <span className="line-clamp-3 break-words">{step.error}</span>
        </div>
      ) : null}
    </div>
  );
}

export function StepRibbon({ steps, totalDurationMs, onStepClick }: StepRibbonProps) {
  if (steps.length === 0) return null;

  const caption = buildCaption(steps, totalDurationMs);

  return (
    <div className="flex flex-col gap-1.5">
      <div className="flex h-2 w-full items-stretch gap-[2px]">
        {steps.map((step) => {
          const eventState = stateFor(step);
          const isActive = !step.isTrigger && eventState === "running";
          const badge = getStatusBadgeProps(
            step.isTrigger ? "triggered" : step.status,
            eventState,
            step.badgeColor,
          );
          return (
            <HoverCard key={step.key} openDelay={150} closeDelay={80}>
              <HoverCardTrigger asChild>
                <button
                  type="button"
                  onClick={() => onStepClick?.(step.key)}
                  aria-label={`${step.name}: ${badge.label}`}
                  className={cn(
                    "group relative h-full flex-1 overflow-hidden rounded-[2px] transition-transform",
                    barColorClass(step),
                    step.isTrigger && "max-w-[10px]",
                    "hover:scale-y-[1.4]",
                  )}
                >
                  {isActive ? (
                    <span className="absolute inset-0 animate-pulse bg-white/30" aria-hidden />
                  ) : null}
                </button>
              </HoverCardTrigger>
              <HoverCardContent align="center" sideOffset={8} className="w-auto p-3">
                <StepHoverCardContent step={step} />
              </HoverCardContent>
            </HoverCard>
          );
        })}
      </div>
      <div className="text-xs text-gray-500">{caption}</div>
    </div>
  );
}
