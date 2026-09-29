export interface ReviewBotOption {
  login: string;
  displayName: string;
}

/** Trim, strip a leading @, and drop a trailing [bot] so the login matches discovery. */
export function normalizeReviewBotLogin(value: string): string {
  return value
    .trim()
    .replace(/^@+/, "")
    .replace(/\[bot\]$/i, "")
    .trim();
}

/** Selected bots first, then the rest of the catalog. Keep manual entries visible. */
export function reviewBotRows(catalog: ReviewBotOption[], selected: string[]): ReviewBotOption[] {
  const rows: ReviewBotOption[] = [];
  const seen = new Set<string>();
  const catalogByLogin = new Map<string, ReviewBotOption>();
  for (const bot of catalog) {
    const key = bot.login.trim().toLowerCase();
    if (!key || catalogByLogin.has(key)) {
      continue;
    }
    catalogByLogin.set(key, bot);
  }

  for (const login of selected) {
    const trimmed = login.trim();
    if (!trimmed) {
      continue;
    }
    const key = trimmed.toLowerCase();
    if (seen.has(key)) {
      continue;
    }
    seen.add(key);
    const catalogBot = catalogByLogin.get(key);
    rows.push(catalogBot ?? { login: trimmed, displayName: trimmed });
  }

  for (const bot of catalog) {
    const key = bot.login.trim().toLowerCase();
    if (!key || seen.has(key)) {
      continue;
    }
    seen.add(key);
    rows.push(bot);
  }

  return rows;
}
