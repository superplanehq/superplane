export function formatCentsPerMillionUsd(cents: number): string {
  return `$${(cents / 100).toFixed(2)}`;
}

export function formatMicrosPerSecondUsdPerMinute(microsPerSecond: number): string {
  const dollarsPerMinute = (microsPerSecond * 60) / 1_000_000;
  if (dollarsPerMinute === 0) {
    return "$0.00 / min";
  }

  const formatted = dollarsPerMinute.toFixed(6).replace(/\.?0+$/, "");
  return `$${formatted} / min`;
}

export function formatMatchMode(mode: string): string {
  switch (mode) {
    case "prefix":
      return "Prefix";
    case "family":
      return "Family";
    case "exact":
      return "Exact";
    default:
      return mode;
  }
}
