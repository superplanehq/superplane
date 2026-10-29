export const MODEL_STATUS_PROVIDERS = ["anthropic", "openai", "openrouter"] as const;

export type ModelStatusProvider = (typeof MODEL_STATUS_PROVIDERS)[number];

export const MODEL_STATUS_WINDOWS = ["5m", "15m", "1h", "24h"] as const;

export type ModelStatusWindow = (typeof MODEL_STATUS_WINDOWS)[number];

export const MODEL_STATUS_DEFAULT_WINDOW: ModelStatusWindow = "5m";

export type ModelStatusCell = {
  provider?: string;
  model?: string;
  score?: number;
  callCount?: number;
};

export type ModelStatusBand = "available" | "degraded" | "unavailable" | "no-data";

export type ModelStatusMatrixCell = {
  provider: ModelStatusProvider;
  model: string;
  score?: number;
  callCount: number;
  band: ModelStatusBand;
};

export type ModelStatusMatrixRow = {
  model: string;
  cells: Record<ModelStatusProvider, ModelStatusMatrixCell>;
  lowestScore: number | null;
};

const PROVIDER_LABELS: Record<ModelStatusProvider, string> = {
  anthropic: "Anthropic",
  openai: "OpenAI",
  openrouter: "OpenRouter",
};

const WINDOW_LABELS: Record<ModelStatusWindow, string> = {
  "5m": "5 minutes",
  "15m": "15 minutes",
  "1h": "1 hour",
  "24h": "24 hours",
};

export function isModelStatusWindow(value: string): value is ModelStatusWindow {
  return (MODEL_STATUS_WINDOWS as readonly string[]).includes(value);
}

function isModelStatusProvider(provider: string): provider is ModelStatusProvider {
  return provider === "anthropic" || provider === "openai" || provider === "openrouter";
}

export function modelStatusProviderLabel(provider: string): string {
  if (isModelStatusProvider(provider)) {
    return PROVIDER_LABELS[provider];
  }
  return provider;
}

export function modelStatusWindowLabel(window: string): string {
  if (isModelStatusWindow(window)) {
    return WINDOW_LABELS[window];
  }
  return WINDOW_LABELS[MODEL_STATUS_DEFAULT_WINDOW];
}

export function modelStatusBand(score: number | undefined): ModelStatusBand {
  if (score == null || Number.isNaN(score)) {
    return "no-data";
  }
  if (score >= 80) {
    return "available";
  }
  if (score >= 50) {
    return "degraded";
  }
  return "unavailable";
}

export function modelStatusCallCountLabel(count: number): string {
  return count === 1 ? "1 call" : `${count} calls`;
}

function groupModelStatusCells(
  cells: ModelStatusCell[],
): Map<string, Partial<Record<ModelStatusProvider, ModelStatusCell>>> {
  const byModel = new Map<string, Partial<Record<ModelStatusProvider, ModelStatusCell>>>();
  for (const cell of cells) {
    const provider = cell.provider?.trim() ?? "";
    const model = cell.model?.trim() ?? "";
    if (!model || !isModelStatusProvider(provider)) {
      continue;
    }
    const current = byModel.get(model) ?? {};
    current[provider] = cell;
    byModel.set(model, current);
  }
  return byModel;
}

function matrixRowFromGrouped(
  model: string,
  grouped: Partial<Record<ModelStatusProvider, ModelStatusCell>>,
): ModelStatusMatrixRow {
  const rowCells = {} as Record<ModelStatusProvider, ModelStatusMatrixCell>;
  let lowestScore: number | null = null;
  for (const provider of MODEL_STATUS_PROVIDERS) {
    const source = grouped[provider];
    const score = typeof source?.score === "number" ? source.score : undefined;
    const callCount = typeof source?.callCount === "number" ? source.callCount : 0;
    if (score != null && (lowestScore == null || score < lowestScore)) {
      lowestScore = score;
    }
    rowCells[provider] = {
      provider,
      model,
      score,
      callCount,
      band: modelStatusBand(score),
    };
  }
  return { model, cells: rowCells, lowestScore };
}

function compareModelStatusRows(a: ModelStatusMatrixRow, b: ModelStatusMatrixRow): number {
  const aScore = a.lowestScore ?? Number.POSITIVE_INFINITY;
  const bScore = b.lowestScore ?? Number.POSITIVE_INFINITY;
  if (aScore !== bScore) {
    return aScore - bScore;
  }
  return a.model.localeCompare(b.model);
}

export function modelStatusMatrixRows(cells: ModelStatusCell[]): ModelStatusMatrixRow[] {
  const rows: ModelStatusMatrixRow[] = [];
  for (const [model, grouped] of groupModelStatusCells(cells)) {
    rows.push(matrixRowFromGrouped(model, grouped));
  }
  return rows.sort(compareModelStatusRows);
}
