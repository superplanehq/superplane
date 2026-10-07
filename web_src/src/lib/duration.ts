const MS_PER_SECOND = 1_000;
const MS_PER_MINUTE = 60_000;
const MS_PER_HOUR = 3_600_000;
const MS_PER_DAY = 86_400_000;
const MS_PER_WEEK = 7 * MS_PER_DAY;

type DurationParts = {
  days?: number;
  hours?: number;
  minutes?: number;
  seconds?: number;
  milliseconds?: number;
};

type DurationFormatConstructor = new (
  locales: string | string[] | undefined,
  options: { style: "narrow" },
) => {
  format(duration: DurationParts): string;
};

type IntlWithDurationFormat = typeof Intl & {
  DurationFormat?: DurationFormatConstructor;
};

function toDurationParts(durationMs: number): DurationParts {
  let remainingMs = Math.max(0, Math.round(durationMs));

  const days = Math.floor(remainingMs / MS_PER_DAY);
  remainingMs -= days * MS_PER_DAY;

  const hours = Math.floor(remainingMs / MS_PER_HOUR);
  remainingMs -= hours * MS_PER_HOUR;

  const minutes = Math.floor(remainingMs / MS_PER_MINUTE);
  remainingMs -= minutes * MS_PER_MINUTE;

  const seconds = Math.floor(remainingMs / MS_PER_SECOND);
  remainingMs -= seconds * MS_PER_SECOND;

  const duration: DurationParts = {};

  if (days > 0) duration.days = days;
  if (hours > 0) duration.hours = hours;
  if (minutes > 0) duration.minutes = minutes;
  if (seconds > 0) duration.seconds = seconds;
  if (remainingMs > 0 || Object.keys(duration).length === 0) duration.milliseconds = remainingMs;

  return duration;
}

function formatDurationFallback(duration: DurationParts): string {
  const parts = [
    duration.days ? `${duration.days}d` : "",
    duration.hours ? `${duration.hours}h` : "",
    duration.minutes ? `${duration.minutes}m` : "",
    duration.seconds ? `${duration.seconds}s` : "",
    duration.milliseconds ? `${duration.milliseconds}ms` : "",
  ].filter(Boolean);

  return parts.join(" ");
}

export type FormatDurationOptions = {
  /**
   * `"millisecond"` (default) renders sub-second remainders as milliseconds,
   * e.g. `"1s 500ms"`.
   *
   * `"second"` rounds to the nearest whole second and never renders
   * milliseconds. Durations under one second render as `"< 1s"` instead of
   * `"0s"` or raw millisecond values. A raw duration of 24 hours or more
   * rounds to the nearest hour and shows days and hours only. Useful for
   * contexts like the work order timeline where sub-second precision is noise.
   */
  precision?: "millisecond" | "second";
};

export function formatDuration(durationMs: number, options?: FormatDurationOptions): string {
  if (options?.precision === "second") {
    return formatSecondPrecision(durationMs);
  }

  return formatResolvedDuration(toDurationParts(durationMs));
}

function formatSecondPrecision(durationMs: number): string {
  if (!Number.isFinite(durationMs) || durationMs <= 0) return "";
  if (durationMs < MS_PER_SECOND) return "< 1s";
  if (durationMs >= MS_PER_DAY) {
    return formatResolvedDuration(toDurationParts(roundToNearestHour(durationMs)));
  }

  return formatResolvedDuration(toDurationParts(Math.round(durationMs / MS_PER_SECOND) * MS_PER_SECOND));
}

function roundToNearestHour(durationMs: number): number {
  return Math.floor((durationMs + MS_PER_HOUR / 2) / MS_PER_HOUR) * MS_PER_HOUR;
}

function formatResolvedDuration(duration: DurationParts): string {
  const DurationFormat = (Intl as IntlWithDurationFormat).DurationFormat;

  if (typeof DurationFormat === "function") {
    return new DurationFormat(undefined, { style: "narrow" }).format(duration);
  }

  return formatDurationFallback(duration);
}

/**
 * At most two units: `2m 5s`, `1h 30m`, `1d 2h`, or `1w 3d`.
 * Rounds the smaller unit. A lone remainder is omitted (`2h`, not `2h 0m`).
 */
export function formatCompactDuration(durationMs: number): string {
  if (!Number.isFinite(durationMs) || durationMs <= 0) {
    return "";
  }
  if (durationMs < MS_PER_SECOND) {
    return "< 1s";
  }
  if (durationMs >= MS_PER_WEEK) {
    return formatTwoUnits({
      durationMs,
      majorMs: MS_PER_WEEK,
      majorUnit: "w",
      minorMs: MS_PER_DAY,
      minorUnit: "d",
      minorPerMajor: 7,
    });
  }
  if (durationMs >= MS_PER_DAY) {
    return formatTwoUnits({
      durationMs,
      majorMs: MS_PER_DAY,
      majorUnit: "d",
      minorMs: MS_PER_HOUR,
      minorUnit: "h",
      minorPerMajor: 24,
    });
  }
  if (durationMs >= MS_PER_HOUR) {
    return formatTwoUnits({
      durationMs,
      majorMs: MS_PER_HOUR,
      majorUnit: "h",
      minorMs: MS_PER_MINUTE,
      minorUnit: "m",
      minorPerMajor: 60,
    });
  }
  if (durationMs >= MS_PER_MINUTE) {
    return formatTwoUnits({
      durationMs,
      majorMs: MS_PER_MINUTE,
      majorUnit: "m",
      minorMs: MS_PER_SECOND,
      minorUnit: "s",
      minorPerMajor: 60,
    });
  }
  return `${Math.round(durationMs / MS_PER_SECOND)}s`;
}

function formatTwoUnits({
  durationMs,
  majorMs,
  majorUnit,
  minorMs,
  minorUnit,
  minorPerMajor,
}: {
  durationMs: number;
  majorMs: number;
  majorUnit: string;
  minorMs: number;
  minorUnit: string;
  minorPerMajor: number;
}): string {
  let major = Math.floor(durationMs / majorMs);
  let minor = Math.round((durationMs % majorMs) / minorMs);
  if (minor >= minorPerMajor) {
    major += 1;
    minor = 0;
  }
  if (minor > 0) {
    return `${major}${majorUnit} ${minor}${minorUnit}`;
  }
  return `${major}${majorUnit}`;
}

/** Clock time for a scan column: `02:59`, or `1:10:22` after one hour. */
export function formatClockDuration(durationMs: number): string {
  const totalSeconds = Math.max(0, Math.round(durationMs / 1000));
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;
  const mm = String(minutes).padStart(2, "0");
  const ss = String(seconds).padStart(2, "0");
  if (hours > 0) {
    return `${hours}:${mm}:${ss}`;
  }
  return `${mm}:${ss}`;
}

const KNOWN_DURATION_WORDS = new Set(["—", "-", "Running", "Waiting", "Pending"]);

function withoutRunningDurationSuffix(label: string): string {
  return label.replace(/\s+so far$/i, "").trim();
}

export function isSubSecondDurationLabel(label: string): boolean {
  return /^<\s*1s$/i.test(withoutRunningDurationSuffix(label));
}

function parseSpokenDurationMs(label: string): number | null {
  const trimmed = withoutRunningDurationSuffix(label);
  if (!trimmed || KNOWN_DURATION_WORDS.has(trimmed)) {
    return null;
  }
  if (/^<\s*1s$/i.test(trimmed)) {
    return 0;
  }
  const clock = trimmed.match(/^(?:(\d+):)?(\d{1,2}):(\d{2})$/);
  if (clock) {
    const hours = Number(clock[1] ?? 0);
    const minutes = Number(clock[2]);
    const seconds = Number(clock[3]);
    return ((hours * 60 + minutes) * 60 + seconds) * 1000;
  }
  if (!/\d+\s*[hms]/i.test(trimmed)) {
    return null;
  }
  const hours = Number(trimmed.match(/(\d+)\s*h\b/i)?.[1] ?? 0);
  const minutes = Number(trimmed.match(/(\d+)\s*m\b/i)?.[1] ?? 0);
  const seconds = Number(trimmed.match(/(\d+)\s*s\b/i)?.[1] ?? 0);
  return ((hours * 60 + minutes) * 60 + seconds) * 1000;
}

/** Parse a stored label such as `4m so far` into milliseconds. Unknown labels are 0. */
export function durationLabelMs(label: string): number {
  return parseSpokenDurationMs(label) ?? 0;
}

/** Turn a stored label such as `2m 59s` into a clock column value. */
export function formatClockDurationLabel(label: string): string {
  const trimmed = withoutRunningDurationSuffix(label);
  if (!trimmed) {
    return "—";
  }
  const ms = parseSpokenDurationMs(trimmed);
  if (ms === null) {
    return trimmed;
  }
  return formatClockDuration(ms);
}

/** Compact Go-style duration: `30s`, `1m2s`, `1h30m`. */
export function formatGoDuration(durationMs: number): string {
  if (!Number.isFinite(durationMs) || durationMs <= 0) {
    return "";
  }
  if (durationMs < 1000) {
    return "<1s";
  }

  const duration = toDurationParts(Math.round(durationMs / 1000) * 1000);
  return [
    duration.days ? `${duration.days}d` : "",
    duration.hours ? `${duration.hours}h` : "",
    duration.minutes ? `${duration.minutes}m` : "",
    duration.seconds ? `${duration.seconds}s` : "",
  ]
    .filter(Boolean)
    .join("");
}

/** Turn a stored label such as `1m 12s` or `4m so far` into `1m12s` / `4m`. */
export function formatGoDurationLabel(label: string): string {
  const trimmed = withoutRunningDurationSuffix(label);
  if (!trimmed || KNOWN_DURATION_WORDS.has(trimmed)) {
    return "";
  }
  const ms = parseSpokenDurationMs(trimmed);
  if (ms === null) {
    return trimmed;
  }
  return formatGoDuration(ms);
}

export function formatMinutesSecondsDuration(durationMs: number): string {
  if (durationMs <= 0) return "";
  if (durationMs < 1000) return "<1s";

  const totalSeconds = Math.floor(durationMs / 1000);
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;

  if (minutes > 0 && seconds > 0) return `${minutes}m ${seconds}s`;
  if (minutes > 0) return `${minutes}m`;
  return `${seconds}s`;
}
