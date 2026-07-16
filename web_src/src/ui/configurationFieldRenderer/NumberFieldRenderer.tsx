import React, { useEffect, useRef } from "react";
import { Input } from "@/components/ui/input";
import type { FieldRendererProps } from "./types";
import { useSkipDefaultsAfterReadOnly } from "./useSkipDefaultsAfterReadOnly";

export const NumberFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange, readOnly = false }) => {
  const numberOptions = field.typeOptions?.number;
  const hasAppliedDefault = useRef(false);
  const skipDefaultsAfterReadOnly = useSkipDefaultsAfterReadOnly(readOnly);

  // Set initial value on first render if no value is present but there's a default
  useEffect(() => {
    if (readOnly || skipDefaultsAfterReadOnly || hasAppliedDefault.current) {
      return;
    }

    if ((value === undefined || value === null) && field.defaultValue !== undefined) {
      const defaultVal = Number(field.defaultValue);
      if (!isNaN(defaultVal)) {
        hasAppliedDefault.current = true;
        onChange(defaultVal);
      }
    }
  }, [readOnly, skipDefaultsAfterReadOnly, value, field.defaultValue, onChange]);

  return (
    <Input
      type="number"
      value={(value as string | number) ?? (field.defaultValue as string) ?? ""}
      onChange={(e) => {
        const val = e.target.value === "" ? undefined : Number(e.target.value);
        onChange(val);
      }}
      placeholder={field.placeholder || ""}
      min={numberOptions?.min}
      max={numberOptions?.max}
      className=""
      disabled={readOnly}
    />
  );
};
