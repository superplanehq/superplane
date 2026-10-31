import { combinedMeasurementState, type UsageMeasurementState } from "./agentRunTelemetry";
import { formatCompactTokenLabel } from "@/lib/formatTokenCount";

export type LiveHeaderSpend = {
  tokens: number;
  cents: number;
  usageState?: UsageMeasurementState;
};

export const EMPTY_LIVE_HEADER_SPEND: LiveHeaderSpend = { tokens: 0, cents: 0 };

export function sumLiveHeaderSpend(spends: Iterable<LiveHeaderSpend>): LiveHeaderSpend {
  let tokens = 0;
  let cents = 0;
  const states: Array<UsageMeasurementState | undefined> = [];
  for (const spend of spends) {
    states.push(spend.usageState);
    tokens += spend.tokens;
    cents += spend.cents;
  }
  const state = combinedMeasurementState(states).usage_state;
  return { tokens, cents, ...(state ? { usageState: state } : {}) };
}

export function overlayHeaderSpend(
  savedCostUsd: string,
  savedTokensLabel: string,
  live: LiveHeaderSpend | undefined,
): { costUsd: string; tokensLabel: string } {
  const liveCents = live?.cents ?? 0;
  const liveTokens = live?.tokens ?? 0;
  const savedCents = parseCostUsdToCents(savedCostUsd);
  const savedTokens = parseTokensLabel(savedTokensLabel);
  const tokens = liveTokens > savedTokens ? formatCompactTokenLabel(liveTokens) : savedTokensLabel;
  const state = live?.usageState;
  return {
    costUsd: liveCents > savedCents ? formatUsdCents(liveCents) : savedCostUsd,
    tokensLabel: measurementTokenLabel(tokens, liveTokens + savedTokens, state),
  };
}

function measurementTokenLabel(label: string, tokens: number, state?: UsageMeasurementState): string {
  if (tokens === 0 && state === "unavailable") return "Tokens unavailable";
  if (tokens === 0 && state === "pending") return "Waiting for token usage";
  if (state === "partial" || state === "unavailable") return `${label} (partial)`;
  return label;
}

function formatUsdCents(cents: number): string {
  return `$${(cents / 100).toFixed(2)}`;
}

function parseCostUsdToCents(costUsd: string): number {
  const match = /\$(\d+)\.(\d{2})/.exec(costUsd);
  if (!match) {
    return 0;
  }
  return Number(match[1]) * 100 + Number(match[2]);
}

function parseTokensLabel(label: string): number {
  const match = /^([\d.]+)\s*([kMB])?\s*tokens$/i.exec(label.trim());
  if (!match) {
    return 0;
  }
  const amount = Number(match[1]);
  if (!Number.isFinite(amount)) {
    return 0;
  }
  if (match[2] === "k") {
    return Math.round(amount * 1_000);
  }
  if (match[2] === "M") {
    return Math.round(amount * 1_000_000);
  }
  if (match[2] === "B") {
    return Math.round(amount * 1_000_000_000);
  }
  return amount;
}
