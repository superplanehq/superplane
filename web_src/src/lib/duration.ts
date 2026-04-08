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

export function formatDuration(durationMs: number): string {
  const duration = toDurationParts(durationMs);
  const DurationFormat = (Intl as IntlWithDurationFormat).DurationFormat;

  if (typeof DurationFormat === "function") {
    return new DurationFormat(undefined, { style: "narrow" }).format(duration);
  }

  return formatDurationFallback(duration);
}
