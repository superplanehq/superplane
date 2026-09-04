import React, { useEffect, useRef } from "react";
import { Input } from "@/components/ui/input";
import type { FieldRendererProps } from "./types";

export const NumberFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange }) => {
  const numberOptions = field.typeOptions?.number;
  const seededDefaultRef = useRef(false);

  // Set the default value on mount if no value is present. The guard is armed on
  // the effect's first run so a later clear (value -> undefined) never re-seeds
  // the default, which would snap the input back (see issue #6399).
  useEffect(() => {
    if (seededDefaultRef.current) return;
    seededDefaultRef.current = true;
    if ((value === undefined || value === null) && field.defaultValue !== undefined) {
      const defaultVal = Number(field.defaultValue);
      if (!isNaN(defaultVal)) {
        onChange(defaultVal);
      }
    }
  }, [value, field.defaultValue, onChange]);

  return (
    <Input
      type="number"
      value={(value ?? "") as string | number}
      onChange={(e) => {
        const val = e.target.value === "" ? undefined : Number(e.target.value);
        onChange(val);
      }}
      placeholder={field.placeholder || ""}
      min={numberOptions?.min}
      max={numberOptions?.max}
      className=""
    />
  );
};
