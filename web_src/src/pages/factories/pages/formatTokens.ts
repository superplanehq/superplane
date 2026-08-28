function formatScaled(value: number, suffix: string): string {
  return `${value.toFixed(1).replace(/\.0$/, "")}${suffix}`;
}

export function formatTokens(value: number): string {
  if (value >= 1_000_000) {
    return formatScaled(value / 1_000_000, "M");
  }
  if (value >= 1_000) {
    return formatScaled(value / 1_000, "k");
  }
  return String(value);
}
