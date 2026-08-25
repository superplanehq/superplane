import type { RunOverlayProvider } from "../work-order-run-overlay/workOrderRunOverlayMocks";

export function formatCostCents(cents?: string): string | undefined {
  if (cents == null || cents === "") {
    return undefined;
  }
  const amount = Number(cents);
  if (!Number.isFinite(amount)) {
    return undefined;
  }
  return `$${(amount / 100).toFixed(2)}`;
}

export function formatTokenCount(tokens?: string): string | undefined {
  if (tokens == null || tokens === "") {
    return undefined;
  }
  const count = Number(tokens);
  if (!Number.isFinite(count)) {
    return undefined;
  }
  if (count >= 1000) {
    const thousands = count / 1000;
    const compact = thousands >= 10 ? thousands.toFixed(0) : thousands.toFixed(1).replace(/\.0$/, "");
    return `${compact}k tokens`;
  }
  return `${count} tokens`;
}

export function clockLabel(iso?: string): string {
  if (!iso) {
    return "—";
  }
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) {
    return "—";
  }
  return date.toLocaleTimeString("en-GB", { hour: "2-digit", minute: "2-digit", second: "2-digit", hour12: false });
}

export function providerForName(name: string): RunOverlayProvider {
  const label = name.toLowerCase();
  if (
    label.includes("pull") ||
    label.includes("pr") ||
    label.includes("github") ||
    label.includes("branch") ||
    label.includes("ci")
  ) {
    return "github";
  }
  return "superplane";
}
