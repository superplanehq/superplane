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

  const days = Math.floor(remainingMs / 86_400_000);
  remainingMs -= days * 86_400_000;

  const hours = Math.floor(remainingMs / 3_600_000);
  remainingMs -= hours * 3_600_000;

  const minutes = Math.floor(remainingMs / 60_000);
  remainingMs -= minutes * 60_000;

  const seconds = Math.floor(remainingMs / 1_000);
  remainingMs -= seconds * 1_000;

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
   * `"0s"` or raw millisecond values. Useful for contexts like the work
   * order timeline where sub-second precision is noise.
   */
  precision?: "millisecond" | "second";
};

export function formatDuration(durationMs: number, options?: FormatDurationOptions): string {
  if (options?.precision === "second") {
    if (!Number.isFinite(durationMs) || durationMs <= 0) return "";
    if (durationMs < 1000) return "< 1s";

    durationMs = Math.round(durationMs / 1000) * 1000;
  }

  const duration = toDurationParts(durationMs);
  const DurationFormat = (Intl as IntlWithDurationFormat).DurationFormat;

  if (typeof DurationFormat === "function") {
    return new DurationFormat(undefined, { style: "narrow" }).format(duration);
  }

  return formatDurationFallback(duration);
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

function parseSpokenDurationMs(label: string): number | null {
  const trimmed = label.replace(/\s+so far$/i, "").trim();
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

/** Turn a stored label such as `2m 59s` into a clock column value. */
export function formatClockDurationLabel(label: string): string {
  const trimmed = label.replace(/\s+so far$/i, "").trim();
  if (!trimmed) {
    return "—";
  }
  const ms = parseSpokenDurationMs(trimmed);
  if (ms === null) {
    return trimmed;
  }
  return formatClockDuration(ms);
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
