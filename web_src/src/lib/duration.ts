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

const DurationFormat = (Intl as typeof Intl & { DurationFormat: DurationFormatConstructor }).DurationFormat;

const durationFormatter = new DurationFormat(undefined, {
  style: "narrow",
});

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

export function formatDuration(durationMs: number): string {
  return durationFormatter.format(toDurationParts(durationMs));
}
