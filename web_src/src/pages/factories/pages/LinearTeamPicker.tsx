import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { Check, Loader2, Search } from "lucide-react";
import { useMemo, useState } from "react";

import { LINEAR_INTAKE_SETUP_COPY } from "./linearIntakeSetupCopy";

export function LinearTeamPicker({
  teams,
  selectedId,
  loading,
  error,
  onSelect,
  onRetry,
}: {
  teams: Array<{ id?: string; name?: string }>;
  selectedId: string;
  loading: boolean;
  error: boolean;
  onSelect: (id: string) => void;
  onRetry: () => void;
}) {
  const [query, setQuery] = useState("");
  const needsSearch = teams.length > 3;
  const filtered = useMemo(() => {
    if (!needsSearch) return teams;
    const term = query.trim().toLowerCase();
    if (!term) return teams;
    return teams.filter((team) => (team.name ?? team.id ?? "").toLowerCase().includes(term));
  }, [needsSearch, teams, query]);

  if (loading) {
    return (
      <p className="workspace-body-text flex items-center gap-2 text-muted-foreground">
        <Loader2 className="size-4 animate-spin" aria-hidden />
        {LINEAR_INTAKE_SETUP_COPY.wizardTeamsLoading}
      </p>
    );
  }
  if (error) {
    return (
      <div className="space-y-3">
        <p className="workspace-body-text text-destructive">{LINEAR_INTAKE_SETUP_COPY.wizardTeamsError}</p>
        <Button type="button" variant="outline" size="sm" onClick={onRetry}>
          {LINEAR_INTAKE_SETUP_COPY.wizardRetry}
        </Button>
      </div>
    );
  }
  if (teams.length === 0) {
    return <p className="workspace-body-text text-muted-foreground">{LINEAR_INTAKE_SETUP_COPY.wizardTeamsEmpty}</p>;
  }

  return (
    <div className="space-y-3" data-testid="linear-team-picker">
      {needsSearch ? (
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground" />
          <Input
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder={LINEAR_INTAKE_SETUP_COPY.wizardSearchTeams}
            className="h-9 pl-9"
            aria-label={LINEAR_INTAKE_SETUP_COPY.wizardSearchTeams}
          />
        </div>
      ) : null}
      <div
        className={cn("rounded-lg border border-border", needsSearch && "max-h-56 overflow-y-auto overscroll-contain")}
        role="listbox"
        aria-label={LINEAR_INTAKE_SETUP_COPY.teamLabel}
      >
        {filtered.length === 0 ? (
          <p className="px-3 py-6 text-center text-[13px] text-muted-foreground">
            {LINEAR_INTAKE_SETUP_COPY.wizardNoMatchingTeams}
          </p>
        ) : (
          <ul className="divide-y divide-border">
            {filtered.map((team) => {
              const id = team.id ?? "";
              const selected = id === selectedId;
              return (
                <li key={id}>
                  <Button
                    type="button"
                    variant="ghost"
                    role="option"
                    aria-selected={selected}
                    onClick={() => onSelect(id)}
                    data-testid={`linear-team-${id}`}
                    className={cn(
                      "h-auto w-full justify-start gap-3 rounded-none px-3 py-2.5 text-left text-[13px] font-medium",
                      selected ? "bg-accent/50" : "hover:bg-accent/30",
                    )}
                  >
                    <span className="min-w-0 flex-1 truncate">{team.name || "Untitled team"}</span>
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
