import React, { useEffect } from "react";
import { TimePickerField } from "./TimePickerField";
import { FieldRendererProps } from "./types";

export const TimeFieldRenderer: React.FC<FieldRendererProps> = ({
  field,
  value,
  onChange,
  hasError,
  allValues = {},
}) => {
  useEffect(() => {
    if ((value === undefined || value === null) && field.defaultValue !== undefined) {
      const defaultVal = field.defaultValue as string;
      if (defaultVal && defaultVal !== "") {
        onChange(defaultVal);
      }
    }
  }, [value, field.defaultValue, onChange]);

  return (
    <TimePickerField
      field={field}
      value={value}
      onChange={onChange}
      hasError={hasError}
      allValues={allValues}
    />
  );
};
