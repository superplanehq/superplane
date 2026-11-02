import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { Check, Loader2 } from "lucide-react";

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
    <div
      className="max-h-56 overflow-y-auto rounded-lg border border-border"
      role="listbox"
      aria-label={LINEAR_INTAKE_SETUP_COPY.teamLabel}
      data-testid="linear-team-picker"
    >
      <ul className="divide-y divide-border">
        {teams.map((team) => {
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
    </div>
  );
}
