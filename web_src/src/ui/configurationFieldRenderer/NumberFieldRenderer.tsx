import React, { useEffect, useRef, useState } from "react";
import { Input } from "@/components/ui/input";
import type { FieldRendererProps } from "./types";

export const NumberFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange }) => {
  const numberOptions = field.typeOptions?.number;

  // The input's text is kept as a local draft so a transiently empty field is
  // allowed while the user edits: clearing must not snap back to the default.
  const [draft, setDraft] = useState<string>(() =>
    value !== undefined && value !== null
      ? String(value)
      : field.defaultValue !== undefined
        ? String(field.defaultValue)
        : "",
  );

  // The last value this component reported upward. Lets the sync effect below
  // distinguish our own echoes from genuinely external value changes.
  const lastEmitted = useRef<unknown>(value);

  // Apply the field default ONCE on mount when no value is present. Running
  // this on every undefined value would fight the user clearing the field.
  const initialized = useRef(false);
  useEffect(() => {
    if (initialized.current) return;
    initialized.current = true;
    if ((value === undefined || value === null) && field.defaultValue !== undefined) {
      const defaultVal = Number(field.defaultValue);
      if (!isNaN(defaultVal)) {
        lastEmitted.current = defaultVal;
        onChange(defaultVal);
      }
    }
  }, [value, field.defaultValue, onChange]);

  // External value changes (form reset, YAML edit, template apply) sync into
  // the draft; our own emitted values do not, so mid-edit state is preserved.
  useEffect(() => {
    if (value === lastEmitted.current) return;
    lastEmitted.current = value;
    setDraft(value === undefined || value === null ? "" : String(value));
  }, [value]);

  return (
    <Input
      type="number"
      value={draft}
      onChange={(e) => {
        const raw = e.target.value;
        setDraft(raw);
        const parsed = raw === "" ? undefined : Number(raw);
        lastEmitted.current = parsed;
        onChange(parsed);
      }}
      placeholder={field.placeholder || ""}
      min={numberOptions?.min}
      max={numberOptions?.max}
      className=""
    />
  );
};
