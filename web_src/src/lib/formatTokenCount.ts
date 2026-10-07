/**
 * Formats a token count using compact notation ("k", "M", "B") instead of
 * the raw number, so large counts stay readable (e.g. 14,616,000 -> "14.6M"
 * rather than "14616k").
 *
 * Rounding rules, applied consistently across the app:
 * - >= 1,000,000,000 -> one decimal place with a "B" suffix.
 * - >= 1,000,000      -> one decimal place with an "M" suffix.
 * - >= 1,000          -> one decimal place with a "k" suffix.
 * - otherwise         -> the plain number.
 *
 * A trailing ".0" is stripped so exact multiples read as "3M" instead of
 * "3.0M" (and "1k" instead of "1.0k").
 */
export function formatCompactTokenValue(tokens: number): string {
  if (tokens >= 1_000_000_000) {
    return `${stripTrailingZero((tokens / 1_000_000_000).toFixed(1))}B`;
  }
  if (tokens >= 1_000_000) {
    return `${stripTrailingZero((tokens / 1_000_000).toFixed(1))}M`;
  }
  if (tokens >= 1_000) {
    return `${stripTrailingZero((tokens / 1_000).toFixed(1))}k`;
  }
  return `${tokens}`;
}

/** Same as {@link formatCompactTokenValue}, with a trailing " tokens" label. */
export function formatCompactTokenLabel(tokens: number): string {
  return `${formatCompactTokenValue(tokens)} tokens`;
}

/**
 * Parses a possibly-string token count and formats it with
 * {@link formatCompactTokenLabel}. Returns `undefined` for missing or
 * non-numeric input so callers can skip rendering a hint entirely.
 */
export function parseAndFormatTokenCount(tokens?: string | number): string | undefined {
  if (tokens == null || tokens === "") {
    return undefined;
  }
  const count = Number(tokens);
  if (!Number.isFinite(count)) {
    return undefined;
  }
  return formatCompactTokenLabel(count);
}

function stripTrailingZero(value: string): string {
  return value.replace(/\.0$/, "");
}
