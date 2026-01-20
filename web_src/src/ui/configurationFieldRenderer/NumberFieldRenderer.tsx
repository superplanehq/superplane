import React, { useEffect } from "react";
import { Input } from "@/components/ui/input";
import { FieldRendererProps } from "./types";

export const NumberFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange }) => {
  const numberOptions = field.typeOptions?.number;

  // Set initial value on first render if no value is present but there's a default
  useEffect(() => {
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
      value={(value as string | number) ?? (field.defaultValue as string) ?? ""}
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
