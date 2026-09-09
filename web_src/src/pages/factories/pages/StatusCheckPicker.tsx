import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";
import { Check, Loader2 } from "lucide-react";
import { useMemo } from "react";

import type { FactoriesFactoryRepositoryStatusCheck } from "@/api-client";

import { PR_FEEDBACK_SETTINGS_COPY } from "./prFeedbackSettingsModel";

export function StatusCheckPicker({
  names,
  catalog,
  loading,
  loadError,
  onToggle,
  hideHeading = false,
}: {
  names: string[];
  catalog: FactoriesFactoryRepositoryStatusCheck[];
  loading?: boolean;
  loadError?: boolean;
  onToggle: (name: string) => void;
  hideHeading?: boolean;
}) {
  const selected = useMemo(() => new Set(names.map((name) => name.toLowerCase())), [names]);
  const rows = useMemo(() => statusCheckRows(catalog, names), [catalog, names]);
  const showFullLoading = Boolean(loading) && rows.length === 0;
  const showCatalogLoading = Boolean(loading) && catalog.length === 0 && rows.length > 0;

  return (
    <section>
      {hideHeading ? null : (
        <>
          <Label>{PR_FEEDBACK_SETTINGS_COPY.checkNamesLabel}</Label>
          <p className="workspace-body-text mt-1 text-muted-foreground">{PR_FEEDBACK_SETTINGS_COPY.checkNamesHelper}</p>
        </>
      )}
      {loadError ? (
        <p className="workspace-body-text mt-1 text-muted-foreground">
          {PR_FEEDBACK_SETTINGS_COPY.checkNamesLoadError}
        </p>
      ) : null}
      <div
        className={cn("max-h-56 overflow-y-auto rounded-lg border border-border", hideHeading ? undefined : "mt-2")}
        role="listbox"
        aria-label={PR_FEEDBACK_SETTINGS_COPY.checkNamesLabel}
        aria-multiselectable="true"
        data-testid="pr-feedback-check-names-picker"
      >
        {showFullLoading ? (
          <div
            className="flex flex-col items-center gap-2 px-4 py-6 text-center"
            data-testid="pr-feedback-check-names-loading"
          >
            <Loader2 className="size-5 animate-spin text-muted-foreground" aria-hidden />
            <p className="text-[13px] font-medium text-foreground">{PR_FEEDBACK_SETTINGS_COPY.checkNamesLoading}</p>
            <p className="workspace-body-text text-muted-foreground">
              {PR_FEEDBACK_SETTINGS_COPY.checkNamesLoadingDetail}
            </p>
          </div>
        ) : rows.length === 0 ? (
          <p
            className="px-3 py-6 text-center text-[13px] text-muted-foreground"
            data-testid="pr-feedback-check-names-empty"
          >
            {PR_FEEDBACK_SETTINGS_COPY.checkNamesCatalogEmpty}
          </p>
        ) : (
          <>
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
            {showCatalogLoading ? (
              <div
                className="flex items-center gap-2 border-t border-border px-3 py-2.5"
                data-testid="pr-feedback-check-names-loading"
              >
                <Loader2 className="size-3.5 animate-spin text-muted-foreground" aria-hidden />
                <p className="workspace-body-text text-muted-foreground">
                  {PR_FEEDBACK_SETTINGS_COPY.checkNamesLoadingMore}
                </p>
              </div>
            ) : null}
          </>
        )}
      </div>
    </section>
  );
}

export function statusCheckRows(
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
    const trimmed = name.trim();
    if (!trimmed) {
      continue;
    }
    const key = trimmed.toLowerCase();
    if (seen.has(key)) {
      continue;
    }
    seen.add(key);
    rows.push({ name: trimmed });
  }

  return rows;
}
