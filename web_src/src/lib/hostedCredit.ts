export function centsToDollarInput(cents: number): string {
  return (cents / 100).toFixed(2);
}

export function dollarInputToCents(value: string): number {
  return parseDollarInputToCents(value) ?? 0;
}

export function parseDollarInputToCents(value: string): number | null {
  const trimmed = value.trim();
  if (trimmed === "") {
    return null;
  }

  const parsed = Number(trimmed);
  if (!Number.isFinite(parsed) || parsed < 0) {
    return null;
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

export const HOSTED_CREDIT_REFRESH_INTERVAL_MS = 2000;
export const HOSTED_CREDIT_REFRESH_TIMEOUT_MS = 30_000;

export type HostedCreditRefreshStatus = "idle" | "refreshing" | "added" | "pending";

export function hostedCreditGrantSnapshotKey(organizationId: string) {
  return `hostedCreditGrantSnapshot:${organizationId}`;
}

export function rememberHostedCreditGrantSnapshot(organizationId: string, grantTotalCents: number) {
  writeSessionValue(hostedCreditGrantSnapshotKey(organizationId), String(grantTotalCents));
}

export function readHostedCreditGrantSnapshot(organizationId: string): number | null {
  const raw = readSessionValue(hostedCreditGrantSnapshotKey(organizationId));
  if (raw == null || raw === "") {
    return null;
  }
  const parsed = Number(raw);
  if (!Number.isFinite(parsed)) {
    return null;
  }
  return parsed;
}

export function clearHostedCreditGrantSnapshot(organizationId: string) {
  removeSessionValue(hostedCreditGrantSnapshotKey(organizationId));
}

export function hostedCreditRefreshStatus(args: {
  creditAddedQuery: boolean;
  snapshotCents: number | null;
  grantTotalCents: number;
  timedOut: boolean;
}): HostedCreditRefreshStatus {
  if (!args.creditAddedQuery) {
    return "idle";
  }
  if (args.snapshotCents !== null && args.grantTotalCents > args.snapshotCents) {
    return "added";
  }
  if (args.timedOut) {
    return "pending";
  }
  return "refreshing";
}

export function hostedCreditRefreshMessage(status: HostedCreditRefreshStatus): string | null {
  switch (status) {
    case "refreshing":
      return "Refreshing hosted credit totals.";
    case "added":
      return "Hosted credit was added.";
    case "pending":
      return "Hosted credit is still updating. Refresh the page to see new totals.";
    default:
      return null;
  }
}

function writeSessionValue(key: string, value: string) {
  try {
    sessionStorage.setItem(key, value);
  } catch {
    // Private mode can block sessionStorage. Checkout still proceeds.
  }
}

function readSessionValue(key: string): string | null {
  try {
    return sessionStorage.getItem(key);
  } catch {
    return null;
  }
}

function removeSessionValue(key: string) {
  try {
    sessionStorage.removeItem(key);
  } catch {
    // Ignore storage failures. The next visit starts without a snapshot.
  }
}
