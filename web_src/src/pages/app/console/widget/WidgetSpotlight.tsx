import { useState } from "react";
import { AlertTriangle, Check, Clock, Loader2, Minus, Rocket, ShieldCheck, Timer, User, X } from "lucide-react";

import { formatTimestampInUserTimezone } from "@/lib/timezone";
import { cn } from "@/lib/utils";

import { WidgetEmptyState } from "../WidgetEmptyState";

import { formatValue } from "./widgetFormat";

/**
 * Prototype `spotlight` panel renderer. Takes a single record — the top row of
 * a data source — and blows it up into an attention-grabbing hero banner
 * instead of a table row. Built for the "what's currently in production?"
 * question: who shipped it, what (the PR), when + how long, who approved it,
 * and are the checks green.
 *
 * Pure and presentational — every slot is passed in already resolved, so a
 * future data-bound panel could feed this from `useWidgetData` (picking
 * `rows[0]`) without touching the renderer.
 */
export type SpotlightStatus = "success" | "running" | "failed" | "warning" | "neutral";

export interface SpotlightActor {
  name: string;
  avatarUrl?: string;
}

export interface SpotlightCheck {
  name: string;
  status: SpotlightStatus;
}

interface WidgetSpotlightProps {
  /** Small eyebrow above the headline, e.g. "Currently in production". */
  kicker?: string;
  /** Overall status; drives the accent bar and header pill color. */
  status?: SpotlightStatus;
  /** Text shown in the header status pill, e.g. "Live" / "Deploying". */
  statusLabel?: string;
  /** Who — the person the banner is about (large avatar + name). */
  actor?: SpotlightActor;
  /** What — the headline, e.g. the PR title. */
  title?: string;
  /** Optional link for the title (opens in a new tab). */
  href?: string;
  /** Secondary line under the title, e.g. repo + branch. */
  subtitle?: string;
  /** When — a timestamp (ISO string or epoch ms/seconds). Rendered relative. */
  timestamp?: string | number;
  /** Duration in milliseconds. */
  duration?: number;
  /** A secondary person, e.g. approver / reviewer / commander. */
  approver?: SpotlightActor;
  /** Label shown before the secondary person. Defaults to "Approved by". */
  approverLabel?: string;
  /** Checks from the PR, rendered as a strip of status pills. */
  checks?: SpotlightCheck[];
  isLoading?: boolean;
}

const STATUS_ACCENT_CLASS: Record<SpotlightStatus, string> = {
  success: "bg-emerald-500",
  running: "bg-blue-500",
  failed: "bg-red-500",
  warning: "bg-amber-500",
  neutral: "bg-slate-300 dark:bg-gray-600",
};

const STATUS_PILL_CLASS: Record<SpotlightStatus, string> = {
  success: "bg-emerald-50 text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-300",
  running: "bg-blue-50 text-blue-700 dark:bg-blue-950/40 dark:text-blue-300",
  failed: "bg-red-50 text-red-700 dark:bg-red-950/40 dark:text-red-300",
  warning: "bg-amber-50 text-amber-700 dark:bg-amber-950/40 dark:text-amber-300",
  neutral: "bg-slate-100 text-slate-600 dark:bg-gray-800 dark:text-gray-300",
};

const STATUS_DOT_CLASS: Record<SpotlightStatus, string> = {
  success: "bg-emerald-500",
  running: "bg-blue-500",
  failed: "bg-red-500",
  warning: "bg-amber-500",
  neutral: "bg-slate-400 dark:bg-gray-500",
};

const CHECK_TONE_CLASS: Record<SpotlightStatus, string> = {
  success: "text-emerald-600 dark:text-emerald-400",
  running: "text-blue-600 dark:text-blue-400",
  failed: "text-red-600 dark:text-red-400",
  warning: "text-amber-600 dark:text-amber-400",
  neutral: "text-slate-500 dark:text-gray-400",
};

export function WidgetSpotlight({
  kicker,
  status = "neutral",
  statusLabel,
  actor,
  title,
  href,
  subtitle,
  timestamp,
  duration,
  approver,
  approverLabel = "Approved by",
  checks,
  isLoading = false,
}: WidgetSpotlightProps) {
  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center p-4">
        <Loader2 className="size-4 animate-spin text-slate-400 dark:text-gray-500" />
      </div>
    );
  }

  const hasContent = Boolean(title || actor || subtitle);
  if (!hasContent) {
    return <WidgetEmptyState icon={Rocket} message="No record to spotlight." testId="widget-spotlight-empty" />;
  }

  return (
    <div className="flex h-full overflow-hidden" data-testid="widget-spotlight">
      <div className={cn("w-1.5 shrink-0", STATUS_ACCENT_CLASS[status])} aria-hidden />
      <div className="flex min-w-0 flex-1 flex-col justify-center gap-3 p-5">
        <SpotlightHeader kicker={kicker} status={status} statusLabel={statusLabel} />
        <SpotlightHeadline title={title} href={href} subtitle={subtitle} />
        <SpotlightMeta
          actor={actor}
          timestamp={timestamp}
          duration={duration}
          approver={approver}
          approverLabel={approverLabel}
        />
        {checks && checks.length > 0 ? <ChecksStrip checks={checks} /> : null}
      </div>
    </div>
  );
}

function SpotlightHeader({
  kicker,
  status,
  statusLabel,
}: {
  kicker?: string;
  status: SpotlightStatus;
  statusLabel?: string;
}) {
  if (!kicker && !statusLabel) return null;
  return (
    <div className="flex items-center justify-between gap-2">
      {kicker ? (
        <span className="truncate text-[11px] font-semibold uppercase tracking-wider text-slate-400 dark:text-gray-500">
          {kicker}
        </span>
      ) : (
        <span />
      )}
      {statusLabel ? (
        <span
          className={cn(
            "inline-flex shrink-0 items-center gap-1.5 rounded-full px-2 py-0.5 text-[11px] font-medium",
            STATUS_PILL_CLASS[status],
          )}
        >
          <span className={cn("size-1.5 rounded-full", STATUS_DOT_CLASS[status])} aria-hidden />
          {statusLabel}
        </span>
      ) : null}
    </div>
  );
}

function SpotlightHeadline({ title, href, subtitle }: { title?: string; href?: string; subtitle?: string }) {
  if (!title && !subtitle) return null;
  return (
    <div className="flex flex-col gap-0.5">
      {title ? (
        href ? (
          <a
            href={href}
            target="_blank"
            rel="noopener noreferrer"
            className="truncate text-xl font-semibold leading-tight text-sky-600 no-underline hover:!underline underline-offset-2 dark:text-sky-400"
            title={title}
          >
            {title}
          </a>
        ) : (
          <span
            className="truncate text-xl font-semibold leading-tight text-slate-900 dark:text-gray-100"
            title={title}
          >
            {title}
          </span>
        )
      ) : null}
      {subtitle ? <span className="truncate text-xs text-slate-500 dark:text-gray-400">{subtitle}</span> : null}
    </div>
  );
}

function SpotlightMeta({
  actor,
  timestamp,
  duration,
  approver,
  approverLabel,
}: {
  actor?: SpotlightActor;
  timestamp?: string | number;
  duration?: number;
  approver?: SpotlightActor;
  approverLabel?: string;
}) {
  const hasTimestamp = timestamp != null && timestamp !== "";
  const hasDuration = duration != null && Number.isFinite(duration);
  if (!actor && !hasTimestamp && !hasDuration && !approver) return null;

  return (
    <div className="flex flex-wrap items-center gap-x-4 gap-y-2 text-xs text-slate-500 dark:text-gray-400">
      {actor ? (
        <span className="flex items-center gap-2">
          <SpotlightAvatar actor={actor} className="size-8" iconClassName="size-4" />
          <span className="font-medium text-slate-800 dark:text-gray-200">{actor.name}</span>
        </span>
      ) : null}
      {hasTimestamp ? (
        <span className="inline-flex items-center gap-1" title={formatAbsoluteTitle(timestamp)}>
          <Clock className="size-3.5" aria-hidden />
          {formatValue(timestamp, "relative")}
        </span>
      ) : null}
      {hasDuration ? (
        <span className="inline-flex items-center gap-1">
          <Timer className="size-3.5" aria-hidden />
          {formatValue(duration, "duration")}
        </span>
      ) : null}
      {approver ? (
        <span className="inline-flex items-center gap-1.5">
          <ShieldCheck className="size-3.5 text-emerald-500" aria-hidden />
          <span>{approverLabel || "Approved by"}</span>
          <SpotlightAvatar actor={approver} className="size-5" iconClassName="size-3" />
          <span className="font-medium text-slate-700 dark:text-gray-300">{approver.name}</span>
        </span>
      ) : null}
    </div>
  );
}

function ChecksStrip({ checks }: { checks: SpotlightCheck[] }) {
  const passed = checks.filter((check) => check.status === "success").length;
  const anyFailed = checks.some((check) => check.status === "failed");
  const allPassed = passed === checks.length;
  const aggregateTone = anyFailed
    ? "text-red-600 dark:text-red-400"
    : allPassed
      ? "text-emerald-600 dark:text-emerald-400"
      : "text-slate-500 dark:text-gray-400";

  return (
    <div className="flex flex-col gap-1.5 border-t border-slate-950/5 pt-3 dark:border-gray-700/60">
      <div className="flex items-center gap-1.5">
        <span className="text-[11px] font-semibold uppercase tracking-wider text-slate-400 dark:text-gray-500">
          Checks
        </span>
        <span className={cn("text-[11px] font-medium tabular-nums", aggregateTone)}>
          {passed}/{checks.length} passed
        </span>
      </div>
      <div className="flex flex-wrap gap-1.5">
        {checks.map((check, index) => (
          <CheckPill key={`${check.name}-${index}`} check={check} />
        ))}
      </div>
    </div>
  );
}

function CheckPill({ check }: { check: SpotlightCheck }) {
  const Icon = CHECK_ICON[check.status];
  return (
    <span
      className="inline-flex items-center gap-1 rounded-md bg-slate-100 px-1.5 py-0.5 text-[11px] font-medium text-slate-700 dark:bg-gray-800 dark:text-gray-300"
      data-testid="widget-spotlight-check"
    >
      <Icon
        className={cn("size-3", CHECK_TONE_CLASS[check.status], check.status === "running" && "animate-spin")}
        aria-hidden
      />
      {check.name}
    </span>
  );
}

const CHECK_ICON: Record<SpotlightStatus, typeof Check> = {
  success: Check,
  failed: X,
  running: Loader2,
  warning: AlertTriangle,
  neutral: Minus,
};

/** Circular avatar with a `User`-icon fallback for missing/broken image URLs. */
function SpotlightAvatar({
  actor,
  className,
  iconClassName,
}: {
  actor: SpotlightActor;
  className: string;
  iconClassName: string;
}) {
  const [errored, setErrored] = useState(false);
  const src = actor.avatarUrl?.trim() ?? "";

  if (src === "" || errored) {
    return (
      <span
        className={cn(
          "inline-flex shrink-0 items-center justify-center rounded-full bg-slate-100 text-slate-400 dark:bg-gray-800 dark:text-gray-500",
          className,
        )}
        aria-label={actor.name}
        data-testid="widget-spotlight-avatar"
      >
        <User className={iconClassName} aria-hidden />
      </span>
    );
  }

  return (
    <img
      src={src}
      alt={actor.name}
      loading="lazy"
      onError={() => setErrored(true)}
      className={cn("shrink-0 rounded-full object-cover", className)}
      data-testid="widget-spotlight-avatar"
    />
  );
}

function formatAbsoluteTitle(value: unknown): string | undefined {
  if (value == null) return undefined;
  if (typeof value === "string" && value.trim() !== "") {
    const parsed = Date.parse(value);
    if (Number.isFinite(parsed)) return formatTimestampInUserTimezone(new Date(parsed).toISOString());
  }
  const n = typeof value === "number" ? value : Number(value);
  if (!Number.isFinite(n)) return undefined;
  const ms = n > 1e12 ? n : n * 1000;
  return formatTimestampInUserTimezone(new Date(ms).toISOString());
}
