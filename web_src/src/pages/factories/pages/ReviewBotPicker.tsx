import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { Check, Loader2 } from "lucide-react";
import { useMemo, useState } from "react";

import { PR_FEEDBACK_SETTINGS_COPY } from "./prFeedbackSettingsModel";

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

export function ReviewBotPicker({
  selected,
  catalog,
  loading,
  loadError,
  onToggle,
  onAdd,
}: {
  selected: string[];
  catalog: ReviewBotOption[];
  loading?: boolean;
  loadError?: boolean;
  onToggle: (login: string) => void;
  onAdd: (login: string) => boolean;
}) {
  const [draft, setDraft] = useState("");
  const selectedKeys = useMemo(() => new Set(selected.map((login) => login.toLowerCase())), [selected]);
  const rows = useMemo(() => reviewBotRows(catalog, selected), [catalog, selected]);
  const showFullLoading = Boolean(loading) && rows.length === 0;

  const submitDraft = () => {
    if (!onAdd(draft)) {
      return;
    }
    setDraft("");
  };

  return (
    <section className="space-y-3">
      {loadError ? (
        <p className="workspace-body-text text-muted-foreground">{PR_FEEDBACK_SETTINGS_COPY.wizardBotsLoadError}</p>
      ) : null}
      <div
        className="max-h-56 overflow-y-auto rounded-lg border border-border"
        role="listbox"
        aria-label={PR_FEEDBACK_SETTINGS_COPY.wizardBotsLabel}
        aria-multiselectable="true"
        data-testid="discussion-setup-bots-picker"
      >
        {showFullLoading ? (
          <div
            className="flex items-center justify-center gap-2 px-4 py-6 text-muted-foreground"
            data-testid="discussion-setup-bots-loading"
          >
            <Loader2 className="size-4 animate-spin" aria-hidden />
            <span className="text-[13px]">{PR_FEEDBACK_SETTINGS_COPY.wizardBotsLoading}</span>
          </div>
        ) : rows.length === 0 ? (
          <p
            className="px-3 py-6 text-center text-[13px] text-muted-foreground"
            data-testid="discussion-setup-bots-empty"
          >
            {PR_FEEDBACK_SETTINGS_COPY.wizardBotsEmpty}
          </p>
        ) : (
          <ul className="divide-y divide-border" data-testid="discussion-setup-bots-list">
            {rows.map((bot) => {
              const isSelected = selectedKeys.has(bot.login.toLowerCase());
              return (
                <li key={bot.login}>
                  <button
                    type="button"
                    role="option"
                    aria-selected={isSelected}
                    onClick={() => onToggle(bot.login)}
                    className={cn(
                      "flex w-full items-center gap-3 px-3 py-2.5 text-left transition-colors",
                      isSelected ? "bg-accent/50" : "hover:bg-accent/30",
                    )}
                    data-testid={`discussion-setup-bot-${bot.login}`}
                  >
                    <span className="min-w-0 flex-1 truncate text-[13px] font-medium">{bot.displayName}</span>
                    <span className="flex size-3.5 shrink-0 items-center justify-center">
                      {isSelected ? (
                        <Check className="size-3.5 text-foreground" strokeWidth={2.5} aria-hidden />
                      ) : null}
                    </span>
                  </button>
                </li>
              );
            })}
          </ul>
        )}
      </div>
      <div className="flex gap-2">
        <Input
          id="discussion-setup-bot-manual"
          value={draft}
          placeholder={PR_FEEDBACK_SETTINGS_COPY.wizardBotsAddPlaceholder}
          aria-label={PR_FEEDBACK_SETTINGS_COPY.wizardBotsAddLabel}
          onChange={(event) => setDraft(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === "Enter") {
              event.preventDefault();
              submitDraft();
            }
          }}
          data-testid="discussion-setup-bot-manual"
        />
        <Button
          type="button"
          variant="outline"
          disabled={normalizeReviewBotLogin(draft).length === 0}
          onClick={submitDraft}
          data-testid="discussion-setup-bot-add"
        >
          {PR_FEEDBACK_SETTINGS_COPY.wizardBotsAdd}
        </Button>
      </div>
    </section>
  );
}
