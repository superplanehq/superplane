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
