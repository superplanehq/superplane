import { formatCompactTokenLabel } from "@/lib/formatTokenCount";

import { formatUsdCents } from "../../lib/workOrderUsage";

export type LiveHeaderSpend = {
  tokens: number;
  cents: number;
};

export const EMPTY_LIVE_HEADER_SPEND: LiveHeaderSpend = { tokens: 0, cents: 0 };

export function sumLiveHeaderSpend(spends: Iterable<LiveHeaderSpend>): LiveHeaderSpend {
  let tokens = 0;
  let cents = 0;
  for (const spend of spends) {
    tokens += spend.tokens;
    cents += spend.cents;
  }
  return { tokens, cents };
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
  return {
    costUsd: liveCents > savedCents ? formatUsdCents(liveCents) : savedCostUsd,
    tokensLabel: liveTokens > savedTokens ? formatCompactTokenLabel(liveTokens) : savedTokensLabel,
  };
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
