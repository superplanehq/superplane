import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";
import { Check } from "lucide-react";
import { useMemo } from "react";

import type { FactoriesFactoryRepositoryStatusCheck } from "@/api-client";

import { PR_FEEDBACK_SETTINGS_COPY } from "./prFeedbackSettingsModel";

export function StatusCheckPicker({
  names,
  catalog,
  loading,
  loadError,
  onToggle,
}: {
  names: string[];
  catalog: FactoriesFactoryRepositoryStatusCheck[];
  loading?: boolean;
  loadError?: boolean;
  onToggle: (name: string) => void;
}) {
  const selected = useMemo(() => new Set(names.map((name) => name.toLowerCase())), [names]);
  const rows = useMemo(() => statusCheckRows(catalog, names), [catalog, names]);

  return (
    <section>
      <Label>{PR_FEEDBACK_SETTINGS_COPY.checkNamesLabel}</Label>
      <p className="workspace-body-text mt-1 text-muted-foreground">{PR_FEEDBACK_SETTINGS_COPY.checkNamesHelper}</p>
      {loadError ? (
        <p className="workspace-body-text mt-1 text-muted-foreground">
          {PR_FEEDBACK_SETTINGS_COPY.checkNamesLoadError}
        </p>
      ) : null}
      <div
        className="mt-2 max-h-56 overflow-y-auto rounded-lg border border-border"
        role="listbox"
        aria-label={PR_FEEDBACK_SETTINGS_COPY.checkNamesLabel}
        aria-multiselectable="true"
        data-testid="pr-feedback-check-names-picker"
      >
        {loading ? (
          <p
            className="px-3 py-6 text-center text-[13px] text-muted-foreground"
            data-testid="pr-feedback-check-names-loading"
          >
            {PR_FEEDBACK_SETTINGS_COPY.checkNamesLoading}
          </p>
        ) : rows.length === 0 ? (
          <p
            className="px-3 py-6 text-center text-[13px] text-muted-foreground"
            data-testid="pr-feedback-check-names-empty"
          >
            {PR_FEEDBACK_SETTINGS_COPY.checkNamesCatalogEmpty}
          </p>
        ) : (
          <ul className="divide-y divide-border" data-testid="pr-feedback-check-names-list">
            {rows.map((row) => {
              const isSelected = selected.has(row.name.toLowerCase());
              return (
                <li key={row.name}>
                  <button
                    type="button"
                    role="option"
                    aria-selected={isSelected}
                    onClick={() => onToggle(row.name)}
                    className={cn(
                      "flex w-full items-center gap-3 px-3 py-2.5 text-left transition-colors",
                      isSelected ? "bg-accent/50" : "hover:bg-accent/30",
                    )}
                    data-testid={`pr-feedback-check-option-${row.name}`}
                  >
                    <span className="min-w-0 flex-1 truncate text-[13px] font-medium">{row.name}</span>
                    {row.required ? (
                      <span className="text-[11px] text-muted-foreground">
                        {PR_FEEDBACK_SETTINGS_COPY.checkNamesRequired}
                      </span>
                    ) : null}
                    {isSelected ? (
                      <Check className="size-3.5 shrink-0 text-foreground" strokeWidth={2.5} aria-hidden />
                    ) : null}
                  </button>
                </li>
              );
            })}
          </ul>
        )}
      </div>
    </section>
  );
}

function statusCheckRows(
  catalog: FactoriesFactoryRepositoryStatusCheck[],
  names: string[],
): Array<{ name: string; required?: boolean }> {
  const rows: Array<{ name: string; required?: boolean }> = [];
  const seen = new Set<string>();

  for (const check of catalog) {
    const name = check.name?.trim();
    if (!name) {
      continue;
    }
    const key = name.toLowerCase();
    if (seen.has(key)) {
      continue;
    }
    seen.add(key);
    rows.push({ name, required: check.required });
  }

  for (const name of names) {
    const key = name.toLowerCase();
    if (seen.has(key)) {
      continue;
    }
    seen.add(key);
    rows.push({ name });
  }

  return rows;
}
