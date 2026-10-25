import { useId } from "react";
import { Input } from "@/components/ui/input";
import { useImplementDurationDraft } from "./implementDurationState";
import { implementDurationMinutes } from "./implementSettingsDraft";
import { ImplementSettingRow } from "./ImplementSettingRow";

export function ImplementRunDuration({
  seconds,
  disabled = false,
  onChange,
}: {
  seconds: unknown;
  disabled?: boolean;
  onChange: (seconds: number) => void;
}) {
  const duration = useImplementDurationDraft();
  const timeout = typeof seconds === "number" && seconds > 0 ? seconds : 3600;
  const minutes = duration?.minutes ?? implementDurationMinutes(seconds);
  const invalid = duration?.invalid ?? false;
  const id = useId();

  const commit = (value: string) => {
    const next = durationSeconds(value);
    if (next === null) {
      duration?.setState({ minutes: value, invalid: true });
      return;
    }
    duration?.setState({ minutes: value, invalid: false });
    if (next !== timeout) onChange(next);
  };

  return (
    <div className="flex flex-col gap-2">
      <ImplementSettingRow
        label="Maximum run duration"
        description="Includes setup. Runs stop when the limit is reached."
        htmlFor={id}
      >
        <Input
          id={id}
          name="implementationRunDuration"
          aria-label="Maximum run duration (minutes)"
          type="number"
          min={1 / 60}
          max={1440}
          step="any"
          value={minutes}
          disabled={disabled}
          onChange={(event) => commit(event.target.value)}
          onBlur={() => {
            const next = durationSeconds(minutes);
            if (next === null) {
              duration?.setState({ minutes, invalid: true });
              return;
            }
            duration?.setState({ minutes: String(next / 60), invalid: false });
            if (next !== timeout) onChange(next);
          }}
          onKeyDown={(event) => {
            if (event.key !== "Enter") return;
            event.preventDefault();
            event.currentTarget.blur();
          }}
          aria-invalid={invalid}
          aria-describedby={`${id}-help${invalid ? ` ${id}-error` : ""}`}
          className="w-24 tabular-nums shadow-none max-sm:text-base"
        />
        <p aria-hidden className="w-14 text-base text-muted-foreground sm:text-sm">
          minutes
        </p>
      </ImplementSettingRow>
      {invalid ? (
        <p id={`${id}-error`} role="alert" className="text-base text-destructive sm:w-64 sm:self-end sm:text-sm">
          Enter a duration from 1 second to 1440 minutes.
        </p>
      ) : null}
    </div>
  );
}

function durationSeconds(minutes: string): number | null {
  const seconds = Math.round(Number(minutes) * 60);
  return Number.isFinite(seconds) && seconds >= 1 && seconds <= 86400 ? seconds : null;
}
