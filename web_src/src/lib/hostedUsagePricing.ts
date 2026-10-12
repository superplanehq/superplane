export type HostedUsagePricing = {
  rate: { Input: number; Output: number; CacheRead: number; CacheWrite: number; Reasoning: number };
  markup_bps: number;
};

/** Use the server's hosted catalog and markup. BYOK streams have no pricing. */
export function withHostedUsageCost(
  usage: Record<string, number>,
  pricing: HostedUsagePricing,
): Record<string, number> {
  const rates = {
    input_tokens: pricing.rate.Input,
    output_tokens: pricing.rate.Output,
    cache_read_input_tokens: pricing.rate.CacheRead,
    cache_creation_input_tokens: pricing.rate.CacheWrite,
    reasoning_tokens: pricing.rate.Reasoning,
  };
  let micros = 0;
  for (const [key, rate] of Object.entries(rates)) {
    const tokens = usage[key] ?? 0;
    if (Number.isFinite(tokens) && Number.isFinite(rate)) {
      micros += Math.floor((Math.max(0, tokens) * Math.max(0, rate)) / 100);
    }
  }
  const billedMicros = Math.floor((micros * (10000 + Math.max(0, pricing.markup_bps))) / 10000);
  return { ...usage, hosted_cost_usd: billedMicros / 1000000 };
}
