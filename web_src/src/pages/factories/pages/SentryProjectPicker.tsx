import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { Check, Loader2, Search } from "lucide-react";
import { useMemo, useState } from "react";

import { SENTRY_INTAKE_SETUP_COPY } from "./sentryIntakeSetupCopy";

export function SentryProjectPicker({
  projects,
  selectedIds,
  loading,
  error,
  onToggle,
  onRetry,
}: {
  projects: Array<{ id?: string; name?: string }>;
  selectedIds: string[];
  loading: boolean;
  error: boolean;
  onToggle: (id: string) => void;
  onRetry: () => void;
}) {
  const [query, setQuery] = useState("");
  const filtered = useMemo(() => {
    const term = query.trim().toLowerCase();
    if (!term) {
      return projects;
    }
    return projects.filter((project) => (project.name ?? project.id ?? "").toLowerCase().includes(term));
  }, [projects, query]);

  if (loading) {
    return (
      <p className="workspace-body-text flex items-center gap-2 text-muted-foreground">
        <Loader2 className="size-4 animate-spin" aria-hidden />
        {SENTRY_INTAKE_SETUP_COPY.wizardProjectsLoading}
      </p>
    );
  }
  if (error) {
    return (
      <div className="space-y-3">
        <p className="workspace-body-text text-destructive">{SENTRY_INTAKE_SETUP_COPY.wizardProjectsError}</p>
        <Button type="button" variant="outline" size="sm" onClick={onRetry}>
          {SENTRY_INTAKE_SETUP_COPY.wizardRetry}
        </Button>
      </div>
    );
  }
  if (projects.length === 0) {
    return <p className="workspace-body-text text-muted-foreground">{SENTRY_INTAKE_SETUP_COPY.wizardProjectsEmpty}</p>;
  }

  return (
    <div className="space-y-3" data-testid="sentry-project-picker">
      <div className="relative">
        <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground" />
        <Input
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          placeholder={SENTRY_INTAKE_SETUP_COPY.wizardSearchProjects}
          className="h-9 pl-9"
          aria-label={SENTRY_INTAKE_SETUP_COPY.wizardSearchProjects}
        />
      </div>
      <div
        className="max-h-56 overflow-y-auto rounded-lg border border-border"
        role="listbox"
        aria-label="Projects"
        aria-multiselectable="true"
      >
        {filtered.length === 0 ? (
          <p className="px-3 py-6 text-center text-[13px] text-muted-foreground">
            {SENTRY_INTAKE_SETUP_COPY.wizardNoMatchingProjects}
          </p>
        ) : (
          <ul className="divide-y divide-border">
            {filtered.map((project) => {
              const id = project.id ?? "";
              const selected = selectedIds.includes(id);
              return (
                <li key={id}>
                  <Button
                    type="button"
                    variant="ghost"
                    role="option"
                    aria-selected={selected}
                    onClick={() => onToggle(id)}
                    data-testid={`sentry-project-${id}`}
                    className={cn(
                      "h-auto w-full justify-start gap-3 rounded-none px-3 py-2.5 text-left text-[13px] font-medium",
                      selected ? "bg-accent/50" : "hover:bg-accent/30",
                    )}
                  >
                    <span className="min-w-0 flex-1 truncate">{project.name || "Untitled project"}</span>
                    {selected ? (
                      <Check className="size-3.5 shrink-0 text-foreground" strokeWidth={2.5} aria-hidden />
                    ) : null}
                  </Button>
                </li>
              );
            })}
          </ul>
        )}
      </div>
    </div>
  );
}
