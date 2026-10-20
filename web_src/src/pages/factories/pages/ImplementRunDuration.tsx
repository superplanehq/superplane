import { useEffect, useId, useRef, useState } from "react";
import { Input } from "@/components/ui/input";
import { ImplementSettingRow } from "./ImplementSettingRow";

export function ImplementRunDuration({ seconds, onChange }: { seconds: unknown; onChange: (seconds: number) => void }) {
  const timeout = typeof seconds === "number" && seconds > 0 ? seconds : 3600;
  const [minutes, setMinutes] = useState(String(timeout / 60));
  const [invalid, setInvalid] = useState(false);
  const pendingSeconds = useRef<number | null>(null);
  const changeRef = useRef(onChange);
  changeRef.current = onChange;
  const id = useId();

  useEffect(() => {
    setMinutes(String(timeout / 60));
    setInvalid(false);
    pendingSeconds.current = null;
  }, [timeout]);

  useEffect(
    () => () => {
      // Dismissal and navigation can unmount the input without a blur event.
      if (pendingSeconds.current !== null) changeRef.current(pendingSeconds.current);
    },
    [],
  );

  const save = () => {
    const next = durationSeconds(minutes);
    pendingSeconds.current = null;
    if (next === null) {
      setInvalid(true);
      return;
    }
    setInvalid(false);
    setMinutes(String(next / 60));
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
          onChange={(event) => {
            setMinutes(event.target.value);
            setInvalid(false);
            const next = durationSeconds(event.target.value);
            pendingSeconds.current = next === timeout ? null : next;
          }}
          onBlur={save}
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
