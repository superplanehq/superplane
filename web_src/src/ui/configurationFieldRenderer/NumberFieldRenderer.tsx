import React, { useEffect, useRef } from "react";
import { Input } from "@/components/ui/input";
import type { FieldRendererProps } from "./types";

export const NumberFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange }) => {
  const numberOptions = field.typeOptions?.number;
  const hasInitialized = useRef(false);

  // Apply the default value once, on mount, if no value has been set yet.
  // We deliberately only run this once (guarded by the ref) instead of on
  // every render where `value` is undefined: without the guard, clearing the
  // input emits onChange(undefined), which re-triggers this effect and
  // immediately re-applies the default, so the field snaps back before the
  // user can type a replacement value.
  useEffect(() => {
    if (hasInitialized.current) {
      return;
    }
    hasInitialized.current = true;

    if ((value === undefined || value === null) && field.defaultValue !== undefined) {
      const defaultVal = Number(field.defaultValue);
      if (!isNaN(defaultVal)) {
        onChange(defaultVal);
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <Input
      type="number"
      value={(value as string | number) ?? ""}
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
