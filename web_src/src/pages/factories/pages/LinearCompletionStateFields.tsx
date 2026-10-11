import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";

import { LINEAR_INTAKE_SETUP_COPY } from "./linearIntakeSetupCopy";

export type LinearCompletionStateValue = {
  linearMoveOnComplete: boolean;
  linearCompletionState: string;
};

// Linear workflow states belong to teams, and an intake can span every team,
// so there is no single team to offer a state picker for. The state travels
// as a name and resolves inside the issue's own team when the task completes.
export function LinearCompletionStateFields({
  value,
  onChange,
}: {
  value: LinearCompletionStateValue;
  onChange: (next: LinearCompletionStateValue) => void;
}) {
  return (
    <fieldset className="min-w-0" data-testid="linear-completion-state">
      <legend className="workspace-section-title">{LINEAR_INTAKE_SETUP_COPY.completionSection}</legend>
      <div className="mt-2 flex flex-col gap-2">
        <div
          className={cn(
            "flex flex-wrap items-center gap-3 rounded-lg border px-3 py-2.5 transition-colors",
            value.linearMoveOnComplete
              ? "border-foreground/20 bg-accent/50"
              : "border-border bg-card hover:border-foreground/15",
          )}
        >
          <Label htmlFor="linear-move-on-complete" className="flex min-w-0 flex-1 cursor-pointer items-center gap-3">
            <Checkbox
              id="linear-move-on-complete"
              checked={value.linearMoveOnComplete}
              onChange={() => onChange({ ...value, linearMoveOnComplete: !value.linearMoveOnComplete })}
              data-testid="linear-move-on-complete"
            />
            <span className="min-w-0 text-[13px] font-medium tracking-[-0.01em] text-foreground">
              {LINEAR_INTAKE_SETUP_COPY.completionMove}
            </span>
          </Label>
          <Input
            value={value.linearCompletionState}
            aria-label={LINEAR_INTAKE_SETUP_COPY.completionState}
            placeholder={LINEAR_INTAKE_SETUP_COPY.completionStatePlaceholder}
            disabled={!value.linearMoveOnComplete}
            onChange={(event) => onChange({ ...value, linearCompletionState: event.target.value })}
            className="h-8 w-[min(100%,10rem)] shrink-0"
            data-testid="linear-completion-state-input"
          />
        </div>
        <p className="workspace-body-text text-muted-foreground">{LINEAR_INTAKE_SETUP_COPY.completionHelper}</p>
      </div>
    </fieldset>
  );
}
