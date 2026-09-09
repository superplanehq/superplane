export function formatCentsPerMillion(cents: number): string {
  return `$${(cents / 100).toFixed(2)} / MTok`;
}

export function formatMicrosPerHour(microsPerSecond: number): string {
  return `$${((microsPerSecond * 3600) / 1_000_000).toFixed(2)} / hour`;
}
