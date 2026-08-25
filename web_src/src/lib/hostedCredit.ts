export function centsToDollarInput(cents: number): string {
  return (cents / 100).toFixed(2);
}

export function dollarInputToCents(value: string): number {
  const parsed = Number(value);
  if (!Number.isFinite(parsed) || parsed < 0) {
    return 0;
  }

  return Math.round(parsed * 100);
}

export function bpsToPercentInput(bps: number): string {
  return String(bps / 100);
}

export function percentInputToBps(value: string): number {
  const parsed = Number(value);
  if (!Number.isFinite(parsed) || parsed < 0) {
    return 0;
  }

  return Math.round(parsed * 100);
}

export function hostedProviderLabel(provider: string): string {
  switch (provider) {
    case "anthropic":
      return "Anthropic";
    case "openai":
      return "OpenAI";
    case "openrouter":
      return "OpenRouter";
    default:
      return provider;
  }
}
